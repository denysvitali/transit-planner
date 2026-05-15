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

	feedID, err := upsertFeed(ctx, tx, opts.Feed)
	if err != nil {
		return ImportResult{}, err
	}
	if _, err := tx.ExecContext(ctx, `update feed_versions set active = 0 where feed_id = ?`, feedID); err != nil {
		return ImportResult{}, err
	}
	versionID, err := insertFeedVersion(ctx, tx, feedID, opts, aggregateHash)
	if err != nil {
		return ImportResult{}, err
	}

	var totalRows int
	for _, file := range files {
		rows, err := importGTFSFile(ctx, tx, feedID, versionID, file)
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

func upsertFeed(ctx context.Context, tx *sql.Tx, feed FeedMetadata) (int64, error) {
	_, err := tx.ExecContext(ctx, `
insert into feeds (
	code, name, country_code, region, publisher_name, license_name, source_url, attribution_text
) values (?, ?, ?, ?, ?, ?, ?, ?)
on conflict(code) do update set
	name = excluded.name,
	country_code = excluded.country_code,
	region = excluded.region,
	publisher_name = excluded.publisher_name,
	license_name = excluded.license_name,
	source_url = excluded.source_url,
	attribution_text = excluded.attribution_text`,
		feed.Code, feed.Name, feed.CountryCode, feed.Region, feed.Publisher, feed.License, feed.SourceURL, feed.Attribution,
	)
	if err != nil {
		return 0, err
	}

	var id int64
	if err := tx.QueryRowContext(ctx, `select id from feeds where code = ?`, feed.Code).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func insertFeedVersion(ctx context.Context, tx *sql.Tx, feedID int64, opts ImportOptions, hash string) (int64, error) {
	result, err := tx.ExecContext(ctx, `
insert into feed_versions (
	feed_id, source_path, source_url, sha256, imported_at, active
) values (?, ?, ?, ?, ?, 1)`,
		feedID, opts.SourcePath, opts.Feed.SourceURL, hash, opts.ImportedAt.UTC().Format(time.RFC3339),
	)
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

func importGTFSFile(ctx context.Context, tx *sql.Tx, feedID, versionID int64, file gtfsFile) (int, error) {
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

	fileResult, err := tx.ExecContext(ctx, `
insert into gtfs_files (
	feed_id, feed_version_id, filename, sha256, header_json, raw_csv
) values (?, ?, ?, ?, ?, ?)`,
		feedID, versionID, file.name, file.sum, string(headerJSON), file.data,
	)
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
		if _, err := tx.ExecContext(ctx, `
insert into gtfs_rows (
	feed_id, feed_version_id, file_id, filename, row_index, row_json
) values (?, ?, ?, ?, ?, ?)`,
			feedID, versionID, fileID, file.name, rows, string(rowJSON),
		); err != nil {
			return 0, err
		}
		if err := insertTypedRow(ctx, tx, feedID, versionID, file.name, rows, row); err != nil {
			return 0, fmt.Errorf("%s row %d: %w", file.name, rows+2, err)
		}
		rows++
	}

	if _, err := tx.ExecContext(ctx, `update gtfs_files set row_count = ? where id = ?`, rows, fileID); err != nil {
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

func insertTypedRow(ctx context.Context, tx *sql.Tx, feedID, versionID int64, filename string, rowIndex int, row map[string]string) error {
	switch filename {
	case "agency.txt":
		return exec(ctx, tx, `insert into agencies values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			feedID, versionID, firstNonEmpty(row["agency_id"], "agency"), row["agency_name"], row["agency_url"], row["agency_timezone"], row["agency_lang"], row["agency_phone"], row["agency_fare_url"], row["agency_email"])
	case "stops.txt":
		return exec(ctx, tx, `insert into stops values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			feedID, versionID, row["stop_id"], row["stop_code"], row["stop_name"], row["stop_desc"], nullableFloat(row["stop_lat"]), nullableFloat(row["stop_lon"]), row["zone_id"], row["stop_url"], nullableInt(row["location_type"]), row["parent_station"], row["stop_timezone"], row["wheelchair_boarding"], row["platform_code"])
	case "routes.txt":
		return exec(ctx, tx, `insert into routes values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			feedID, versionID, row["route_id"], row["agency_id"], row["route_short_name"], row["route_long_name"], row["route_desc"], nullableInt(row["route_type"]), row["route_url"], row["route_color"], row["route_text_color"], row["route_sort_order"], row["continuous_pickup"])
	case "trips.txt":
		return exec(ctx, tx, `insert into trips values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			feedID, versionID, row["route_id"], row["service_id"], row["trip_id"], row["trip_headsign"], row["trip_short_name"], nullableInt(row["direction_id"]), row["block_id"], row["shape_id"], row["wheelchair_accessible"], row["bikes_allowed"], row["cars_allowed"])
	case "stop_times.txt":
		return exec(ctx, tx, `insert into stop_times values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			feedID, versionID, rowIndex, row["trip_id"], row["arrival_time"], row["departure_time"], row["stop_id"], nullableInt(row["stop_sequence"]), row["stop_headsign"], row["pickup_type"], row["drop_off_type"], row["continuous_pickup"], row["continuous_drop_off"], nullableFloat(row["shape_dist_traveled"]), row["timepoint"])
	case "calendar.txt":
		return exec(ctx, tx, `insert into calendars values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			feedID, versionID, row["service_id"], boolInt(row["monday"]), boolInt(row["tuesday"]), boolInt(row["wednesday"]), boolInt(row["thursday"]), boolInt(row["friday"]), boolInt(row["saturday"]), boolInt(row["sunday"]), row["start_date"], row["end_date"])
	case "calendar_dates.txt":
		return exec(ctx, tx, `insert into calendar_dates values (?, ?, ?, ?, ?)`,
			feedID, versionID, row["service_id"], row["date"], nullableInt(row["exception_type"]))
	case "transfers.txt":
		return exec(ctx, tx, `insert into transfers values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			feedID, versionID, row["from_stop_id"], row["to_stop_id"], row["from_route_id"], row["to_route_id"], row["from_trip_id"], row["to_trip_id"], nullableInt(row["transfer_type"]), nullableInt(row["min_transfer_time"]))
	case "shapes.txt":
		return exec(ctx, tx, `insert into shapes values (?, ?, ?, ?, ?, ?)`,
			feedID, versionID, row["shape_id"], nullableFloat(row["shape_pt_lat"]), nullableFloat(row["shape_pt_lon"]), nullableInt(row["shape_pt_sequence"]))
	case "feed_info.txt":
		return exec(ctx, tx, `insert into feed_info values (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			feedID, versionID, row["feed_publisher_name"], row["feed_publisher_url"], row["feed_lang"], row["default_lang"], row["feed_start_date"], row["feed_end_date"], row["feed_version"], row["feed_contact_email"])
	default:
		return nil
	}
}

func exec(ctx context.Context, tx *sql.Tx, query string, args ...any) error {
	_, err := tx.ExecContext(ctx, query, args...)
	return err
}

func nullableInt(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return nil
	}
	return parsed
}

func nullableFloat(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil
	}
	return parsed
}

func boolInt(value string) any {
	if value == "" {
		return nil
	}
	if value == "1" || strings.EqualFold(value, "true") {
		return 1
	}
	return 0
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
