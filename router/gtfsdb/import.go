package gtfsdb

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	sqlcdb "github.com/denysvitali/transit-planner/router/gtfsdb/db"

	_ "modernc.org/sqlite"
)

type FeedMetadata struct {
	Code        string
	Name        string
	CountryCode string
	Region      string
	Publisher   string
	License     string
	SourceURL   string
	Attribution string
}

type ImportOptions struct {
	DBPath     string
	SourcePath string
	Feed       FeedMetadata
	ImportedAt time.Time
}

type ImportResult struct {
	FeedID        int64
	FeedVersionID int64
	Files         int
	Rows          int
	SHA256        string
}

type gtfsFile struct {
	name string
	data []byte
	sum  string
}

func ImportFeed(ctx context.Context, opts ImportOptions) (ImportResult, error) {
	if strings.TrimSpace(opts.DBPath) == "" {
		return ImportResult{}, errors.New("DBPath is required")
	}
	if strings.TrimSpace(opts.SourcePath) == "" {
		return ImportResult{}, errors.New("SourcePath is required")
	}
	opts.Feed.Code = strings.TrimSpace(opts.Feed.Code)
	if opts.Feed.Code == "" {
		return ImportResult{}, errors.New("Feed.Code is required")
	}
	if opts.ImportedAt.IsZero() {
		opts.ImportedAt = time.Now().UTC()
	}

	files, aggregateHash, err := readGTFSFiles(opts.SourcePath)
	if err != nil {
		return ImportResult{}, err
	}
	if len(files) == 0 {
		return ImportResult{}, fmt.Errorf("%s: no GTFS .txt files found", opts.SourcePath)
	}

	db, err := sql.Open("sqlite", opts.DBPath)
	if err != nil {
		return ImportResult{}, err
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		return ImportResult{}, err
	}
	if _, err := db.ExecContext(ctx, schemaSQL); err != nil {
		return ImportResult{}, err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return ImportResult{}, err
	}
	defer tx.Rollback()
	q := sqlcdb.New(tx)

	feedID, err := upsertFeed(ctx, q, opts.Feed)
	if err != nil {
		return ImportResult{}, err
	}
	if err := q.DeactivateFeedVersions(ctx, feedID); err != nil {
		return ImportResult{}, err
	}
	versionID, err := insertFeedVersion(ctx, q, feedID, opts, aggregateHash)
	if err != nil {
		return ImportResult{}, err
	}

	var totalRows int
	for _, file := range files {
		rows, err := importGTFSFile(ctx, q, feedID, versionID, file)
		if err != nil {
			return ImportResult{}, err
		}
		totalRows += rows
	}
	if err := tx.Commit(); err != nil {
		return ImportResult{}, err
	}
	return ImportResult{
		FeedID:        feedID,
		FeedVersionID: versionID,
		Files:         len(files),
		Rows:          totalRows,
		SHA256:        aggregateHash,
	}, nil
}

func upsertFeed(ctx context.Context, q *sqlcdb.Queries, feed FeedMetadata) (int64, error) {
	if err := q.UpsertFeed(ctx, sqlcdb.UpsertFeedParams{
		Code:            feed.Code,
		Name:            feed.Name,
		CountryCode:     textValue(feed.CountryCode),
		Region:          textValue(feed.Region),
		PublisherName:   textValue(feed.Publisher),
		LicenseName:     textValue(feed.License),
		SourceUrl:       textValue(feed.SourceURL),
		AttributionText: textValue(feed.Attribution),
	}); err != nil {
		return 0, err
	}

	return q.GetFeedIDByCode(ctx, feed.Code)
}

func insertFeedVersion(ctx context.Context, q *sqlcdb.Queries, feedID int64, opts ImportOptions, hash string) (int64, error) {
	result, err := q.InsertFeedVersion(ctx, sqlcdb.InsertFeedVersionParams{
		FeedID:     feedID,
		SourcePath: opts.SourcePath,
		SourceUrl:  textValue(opts.Feed.SourceURL),
		Sha256:     hash,
		ImportedAt: opts.ImportedAt.UTC().Format(time.RFC3339),
	})
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func readGTFSFiles(sourcePath string) ([]gtfsFile, string, error) {
	info, err := os.Stat(sourcePath)
	if err != nil {
		return nil, "", err
	}

	var files []gtfsFile
	if info.IsDir() {
		files, err = readGTFSFilesFromFS(os.DirFS(sourcePath))
	} else {
		files, err = readGTFSFilesFromZip(sourcePath)
	}
	if err != nil {
		return nil, "", err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].name < files[j].name })

	hash := sha256.New()
	for _, file := range files {
		hash.Write([]byte(file.name))
		hash.Write([]byte{0})
		hash.Write([]byte(file.sum))
		hash.Write([]byte{0})
	}
	return files, hex.EncodeToString(hash.Sum(nil)), nil
}

func readGTFSFilesFromFS(fsys fs.FS) ([]gtfsFile, error) {
	var files []gtfsFile
	err := fs.WalkDir(fsys, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !isGTFSFile(name) {
			return nil
		}
		data, err := fs.ReadFile(fsys, name)
		if err != nil {
			return err
		}
		files = append(files, newGTFSFile(path.Base(name), data))
		return nil
	})
	return dedupeFiles(files), err
}

func readGTFSFilesFromZip(sourcePath string) ([]gtfsFile, error) {
	reader, err := zip.OpenReader(sourcePath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var files []gtfsFile
	for _, file := range reader.File {
		if file.FileInfo().IsDir() || !isGTFSFile(file.Name) {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return nil, err
		}
		data, readErr := io.ReadAll(rc)
		closeErr := rc.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		files = append(files, newGTFSFile(path.Base(file.Name), data))
	}
	return dedupeFiles(files), nil
}

func isGTFSFile(name string) bool {
	base := path.Base(filepath.ToSlash(name))
	return strings.HasSuffix(base, ".txt") && !strings.HasPrefix(base, ".")
}

func newGTFSFile(name string, data []byte) gtfsFile {
	sum := sha256.Sum256(data)
	return gtfsFile{name: name, data: data, sum: hex.EncodeToString(sum[:])}
}

func dedupeFiles(files []gtfsFile) []gtfsFile {
	byName := map[string]gtfsFile{}
	for _, file := range files {
		if _, ok := byName[file.name]; !ok {
			byName[file.name] = file
		}
	}
	out := make([]gtfsFile, 0, len(byName))
	for _, file := range byName {
		out = append(out, file)
	}
	return out
}

func importGTFSFile(ctx context.Context, q *sqlcdb.Queries, feedID, versionID int64, file gtfsFile) (int, error) {
	reader := csv.NewReader(bytes.NewReader(file.data))
	reader.FieldsPerRecord = -1

	header, err := reader.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return 0, nil
		}
		return 0, fmt.Errorf("%s header: %w", file.name, err)
	}
	if len(header) > 0 {
		header[0] = strings.TrimPrefix(header[0], "\ufeff")
	}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return 0, err
	}

	fileResult, err := q.InsertGTFSFile(ctx, sqlcdb.InsertGTFSFileParams{
		FeedID:        feedID,
		FeedVersionID: versionID,
		Filename:      file.name,
		Sha256:        file.sum,
		HeaderJson:    string(headerJSON),
		RawCsv:        file.data,
	})
	if err != nil {
		return 0, err
	}
	fileID, err := fileResult.LastInsertId()
	if err != nil {
		return 0, err
	}

	var rows int
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("%s row %d: %w", file.name, rows+2, err)
		}
		row := rowMap(header, record)
		rowJSON, err := json.Marshal(row)
		if err != nil {
			return 0, err
		}
		if err := q.InsertGTFSRow(ctx, sqlcdb.InsertGTFSRowParams{
			FeedID:        feedID,
			FeedVersionID: versionID,
			FileID:        fileID,
			Filename:      file.name,
			RowIndex:      int64(rows),
			RowJson:       string(rowJSON),
		}); err != nil {
			return 0, err
		}
		if err := insertTypedRow(ctx, q, feedID, versionID, file.name, rows, row); err != nil {
			return 0, fmt.Errorf("%s row %d: %w", file.name, rows+2, err)
		}
		rows++
	}

	if err := q.UpdateGTFSFileRowCount(ctx, sqlcdb.UpdateGTFSFileRowCountParams{
		RowCount: int64(rows),
		ID:       fileID,
	}); err != nil {
		return 0, err
	}
	return rows, nil
}

func rowMap(header, record []string) map[string]string {
	row := make(map[string]string, len(header))
	for i, name := range header {
		if name == "" {
			continue
		}
		value := ""
		if i < len(record) {
			value = record[i]
		}
		row[name] = value
	}
	return row
}

func insertTypedRow(ctx context.Context, q *sqlcdb.Queries, feedID, versionID int64, filename string, rowIndex int, row map[string]string) error {
	switch filename {
	case "agency.txt":
		return q.InsertAgency(ctx, sqlcdb.InsertAgencyParams{
			FeedID:         feedID,
			FeedVersionID:  versionID,
			AgencyID:       firstNonEmpty(row["agency_id"], "agency"),
			AgencyName:     textValue(row["agency_name"]),
			AgencyUrl:      textValue(row["agency_url"]),
			AgencyTimezone: textValue(row["agency_timezone"]),
			AgencyLang:     textValue(row["agency_lang"]),
			AgencyPhone:    textValue(row["agency_phone"]),
			AgencyFareUrl:  textValue(row["agency_fare_url"]),
			AgencyEmail:    textValue(row["agency_email"]),
		})
	case "stops.txt":
		return q.InsertStop(ctx, sqlcdb.InsertStopParams{
			FeedID:             feedID,
			FeedVersionID:      versionID,
			StopID:             row["stop_id"],
			StopCode:           textValue(row["stop_code"]),
			StopName:           textValue(row["stop_name"]),
			StopDesc:           textValue(row["stop_desc"]),
			StopLat:            nullableFloat(row["stop_lat"]),
			StopLon:            nullableFloat(row["stop_lon"]),
			ZoneID:             textValue(row["zone_id"]),
			StopUrl:            textValue(row["stop_url"]),
			LocationType:       nullableInt(row["location_type"]),
			ParentStation:      textValue(row["parent_station"]),
			StopTimezone:       textValue(row["stop_timezone"]),
			WheelchairBoarding: textValue(row["wheelchair_boarding"]),
			PlatformCode:       textValue(row["platform_code"]),
		})
	case "routes.txt":
		return q.InsertRoute(ctx, sqlcdb.InsertRouteParams{
			FeedID:           feedID,
			FeedVersionID:    versionID,
			RouteID:          row["route_id"],
			AgencyID:         textValue(row["agency_id"]),
			RouteShortName:   textValue(row["route_short_name"]),
			RouteLongName:    textValue(row["route_long_name"]),
			RouteDesc:        textValue(row["route_desc"]),
			RouteType:        nullableInt(row["route_type"]),
			RouteUrl:         textValue(row["route_url"]),
			RouteColor:       textValue(row["route_color"]),
			RouteTextColor:   textValue(row["route_text_color"]),
			RouteSortOrder:   textValue(row["route_sort_order"]),
			ContinuousPickup: textValue(row["continuous_pickup"]),
		})
	case "trips.txt":
		return q.InsertTrip(ctx, sqlcdb.InsertTripParams{
			FeedID:               feedID,
			FeedVersionID:        versionID,
			RouteID:              row["route_id"],
			ServiceID:            row["service_id"],
			TripID:               row["trip_id"],
			TripHeadsign:         textValue(row["trip_headsign"]),
			TripShortName:        textValue(row["trip_short_name"]),
			DirectionID:          nullableInt(row["direction_id"]),
			BlockID:              textValue(row["block_id"]),
			ShapeID:              textValue(row["shape_id"]),
			WheelchairAccessible: textValue(row["wheelchair_accessible"]),
			BikesAllowed:         textValue(row["bikes_allowed"]),
			CarsAllowed:          textValue(row["cars_allowed"]),
		})
	case "stop_times.txt":
		return q.InsertStopTime(ctx, sqlcdb.InsertStopTimeParams{
			FeedID:            feedID,
			FeedVersionID:     versionID,
			RowIndex:          int64(rowIndex),
			TripID:            row["trip_id"],
			ArrivalTime:       textValue(row["arrival_time"]),
			DepartureTime:     textValue(row["departure_time"]),
			StopID:            row["stop_id"],
			StopSequence:      nullableInt(row["stop_sequence"]),
			StopHeadsign:      textValue(row["stop_headsign"]),
			PickupType:        textValue(row["pickup_type"]),
			DropOffType:       textValue(row["drop_off_type"]),
			ContinuousPickup:  textValue(row["continuous_pickup"]),
			ContinuousDropOff: textValue(row["continuous_drop_off"]),
			ShapeDistTraveled: nullableFloat(row["shape_dist_traveled"]),
			Timepoint:         textValue(row["timepoint"]),
		})
	case "calendar.txt":
		return q.InsertCalendar(ctx, sqlcdb.InsertCalendarParams{
			FeedID:        feedID,
			FeedVersionID: versionID,
			ServiceID:     row["service_id"],
			Monday:        boolInt(row["monday"]),
			Tuesday:       boolInt(row["tuesday"]),
			Wednesday:     boolInt(row["wednesday"]),
			Thursday:      boolInt(row["thursday"]),
			Friday:        boolInt(row["friday"]),
			Saturday:      boolInt(row["saturday"]),
			Sunday:        boolInt(row["sunday"]),
			StartDate:     textValue(row["start_date"]),
			EndDate:       textValue(row["end_date"]),
		})
	case "calendar_dates.txt":
		return q.InsertCalendarDate(ctx, sqlcdb.InsertCalendarDateParams{
			FeedID:        feedID,
			FeedVersionID: versionID,
			ServiceID:     row["service_id"],
			Date:          row["date"],
			ExceptionType: nullableInt(row["exception_type"]),
		})
	case "transfers.txt":
		return q.InsertTransfer(ctx, sqlcdb.InsertTransferParams{
			FeedID:          feedID,
			FeedVersionID:   versionID,
			FromStopID:      textValue(row["from_stop_id"]),
			ToStopID:        textValue(row["to_stop_id"]),
			FromRouteID:     textValue(row["from_route_id"]),
			ToRouteID:       textValue(row["to_route_id"]),
			FromTripID:      textValue(row["from_trip_id"]),
			ToTripID:        textValue(row["to_trip_id"]),
			TransferType:    nullableInt(row["transfer_type"]),
			MinTransferTime: nullableInt(row["min_transfer_time"]),
		})
	case "shapes.txt":
		return q.InsertShape(ctx, sqlcdb.InsertShapeParams{
			FeedID:          feedID,
			FeedVersionID:   versionID,
			ShapeID:         row["shape_id"],
			ShapePtLat:      nullableFloat(row["shape_pt_lat"]),
			ShapePtLon:      nullableFloat(row["shape_pt_lon"]),
			ShapePtSequence: nullableInt(row["shape_pt_sequence"]),
		})
	case "feed_info.txt":
		return q.InsertFeedInfo(ctx, sqlcdb.InsertFeedInfoParams{
			FeedID:            feedID,
			FeedVersionID:     versionID,
			FeedPublisherName: textValue(row["feed_publisher_name"]),
			FeedPublisherUrl:  textValue(row["feed_publisher_url"]),
			FeedLang:          textValue(row["feed_lang"]),
			DefaultLang:       textValue(row["default_lang"]),
			FeedStartDate:     textValue(row["feed_start_date"]),
			FeedEndDate:       textValue(row["feed_end_date"]),
			FeedVersion:       textValue(row["feed_version"]),
			FeedContactEmail:  textValue(row["feed_contact_email"]),
		})
	default:
		return nil
	}
}

func textValue(value string) sql.NullString {
	return sql.NullString{String: value, Valid: true}
}

func nullableInt(value string) sql.NullInt64 {
	if strings.TrimSpace(value) == "" {
		return sql.NullInt64{}
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(parsed), Valid: true}
}

func nullableFloat(value string) sql.NullFloat64 {
	if strings.TrimSpace(value) == "" {
		return sql.NullFloat64{}
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: parsed, Valid: true}
}

func boolInt(value string) sql.NullInt64 {
	if value == "" {
		return sql.NullInt64{}
	}
	if value == "1" || strings.EqualFold(value, "true") {
		return sql.NullInt64{Int64: 1, Valid: true}
	}
	return sql.NullInt64{Int64: 0, Valid: true}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
