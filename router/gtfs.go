package router

import (
	"archive/zip"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Stop struct {
	ID   string
	Name string
	Lat  float64
	Lon  float64
}

type Route struct {
	ID        string
	ShortName string
	LongName  string
	Type      int
}

type Trip struct {
	ID        string
	RouteID   string
	ServiceID string
	StopTimes []StopTime
}

type StopTime struct {
	StopID    string
	Sequence  int
	Arrival   int
	Departure int
}

type Transfer struct {
	FromStopID string
	ToStopID   string
	Duration   int
}

const sameStationTransferDuration = 120
const sameStationTransferMaxMeters = 500.0

type Feed struct {
	Stops     map[string]Stop
	Routes    map[string]Route
	Trips     map[string]Trip
	Transfers []Transfer
}

// LoadGTFS reads a GTFS feed from a directory containing the standard CSV
// files (stops.txt, routes.txt, ...).
func LoadGTFS(dir string) (*Feed, error) {
	return loadFeed(os.DirFS(dir))
}

// LoadGTFSZip reads a GTFS feed directly from a zip archive. Useful for the
// real-world feeds published by agencies (e.g. ODPT for Tokyo's Toei lines)
// without having to extract them to disk first.
func LoadGTFSZip(path string) (*Feed, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return loadFeed(&r.Reader)
}

// findGTFSRoot returns a subtree of fsys rooted at the directory that contains
// stops.txt. Some agencies publish GTFS zips where every file lives under a
// single top-level folder (e.g. "agency-name/stops.txt"), so opening
// "stops.txt" off the zip root fails. We scan once for stops.txt and rebase
// the FS so the rest of the loader can address files by their bare names.
func findGTFSRoot(fsys fs.FS) (fs.FS, error) {
	if _, err := fs.Stat(fsys, "stops.txt"); err == nil {
		return fsys, nil
	}
	var found string
	walkErr := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() == "stops.txt" {
			found = path
			return fs.SkipAll
		}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	if found == "" {
		return nil, fs.ErrNotExist
	}
	dir := strings.TrimSuffix(found, "stops.txt")
	dir = strings.TrimSuffix(dir, "/")
	if dir == "" || dir == "." {
		return fsys, nil
	}
	return fs.Sub(fsys, dir)
}

func loadFeed(raw fs.FS) (*Feed, error) {
	fsys, err := findGTFSRoot(raw)
	if err != nil {
		// Preserve the historical error shape so callers (and tests) keep
		// seeing "open stops.txt: file does not exist" for empty zips.
		return nil, &fs.PathError{Op: "open", Path: "stops.txt", Err: fs.ErrNotExist}
	}
	stops, err := loadStops(fsys)
	if err != nil {
		return nil, err
	}
	routes, err := loadRoutes(fsys)
	if err != nil {
		return nil, err
	}
	trips, err := loadTrips(fsys)
	if err != nil {
		return nil, err
	}
	stopTimes, err := loadStopTimes(fsys)
	if err != nil {
		return nil, err
	}
	for tripID, times := range stopTimes {
		trip := trips[tripID]
		trip.StopTimes = times
		trips[tripID] = trip
	}
	transfers, err := loadTransfers(fsys)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	transfers = appendSameStationTransfers(stops, transfers)
	return &Feed{
		Stops:     stops,
		Routes:    routes,
		Trips:     trips,
		Transfers: transfers,
	}, nil
}

func loadStops(fsys fs.FS) (map[string]Stop, error) {
	rows, err := readCSV(fsys, "stops.txt")
	if err != nil {
		return nil, err
	}
	stops := make(map[string]Stop, len(rows))
	for _, row := range rows {
		lat, err := strconv.ParseFloat(row["stop_lat"], 64)
		if err != nil {
			return nil, fmt.Errorf("stop %q latitude: %w", row["stop_id"], err)
		}
		lon, err := strconv.ParseFloat(row["stop_lon"], 64)
		if err != nil {
			return nil, fmt.Errorf("stop %q longitude: %w", row["stop_id"], err)
		}
		stops[row["stop_id"]] = Stop{
			ID:   row["stop_id"],
			Name: row["stop_name"],
			Lat:  lat,
			Lon:  lon,
		}
	}
	return stops, nil
}

func loadRoutes(fsys fs.FS) (map[string]Route, error) {
	rows, err := readCSV(fsys, "routes.txt")
	if err != nil {
		return nil, err
	}
	routes := make(map[string]Route, len(rows))
	for _, row := range rows {
		routeType, _ := strconv.Atoi(row["route_type"])
		routes[row["route_id"]] = Route{
			ID:        row["route_id"],
			ShortName: firstNonEmpty(row["route_short_name"], row["route_long_name"]),
			LongName:  row["route_long_name"],
			Type:      routeType,
		}
	}
	return routes, nil
}

func loadTrips(fsys fs.FS) (map[string]Trip, error) {
	rows, err := readCSV(fsys, "trips.txt")
	if err != nil {
		return nil, err
	}
	trips := make(map[string]Trip, len(rows))
	for _, row := range rows {
		trips[row["trip_id"]] = Trip{
			ID:        row["trip_id"],
			RouteID:   row["route_id"],
			ServiceID: row["service_id"],
		}
	}
	return trips, nil
}

func loadStopTimes(fsys fs.FS) (map[string][]StopTime, error) {
	file, reader, header, err := openCSV(fsys, "stop_times.txt")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	tripIDCol, ok := header["trip_id"]
	if !ok {
		return nil, errors.New("stop_times.txt: missing trip_id")
	}
	arrivalCol, ok := header["arrival_time"]
	if !ok {
		return nil, errors.New("stop_times.txt: missing arrival_time")
	}
	departureCol, ok := header["departure_time"]
	if !ok {
		return nil, errors.New("stop_times.txt: missing departure_time")
	}
	stopIDCol, ok := header["stop_id"]
	if !ok {
		return nil, errors.New("stop_times.txt: missing stop_id")
	}
	sequenceCol, ok := header["stop_sequence"]
	if !ok {
		return nil, errors.New("stop_times.txt: missing stop_sequence")
	}

	times := map[string][]StopTime{}
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		// Real feeds (e.g. Toei) leave arrival_time and departure_time blank
		// for non-timepoint intermediate stops. Skip them so the trip pattern
		// still loads; the router boards/alights only at timed stops.
		tripID := csvValue(record, tripIDCol)
		arrivalRaw := csvValue(record, arrivalCol)
		departureRaw := csvValue(record, departureCol)
		stopID := csvValue(record, stopIDCol)
		sequenceRaw := csvValue(record, sequenceCol)
		if arrivalRaw == "" && departureRaw == "" {
			continue
		}
		arrival, err := parseGTFSTime(firstNonEmpty(arrivalRaw, departureRaw))
		if err != nil {
			return nil, err
		}
		departure, err := parseGTFSTime(firstNonEmpty(departureRaw, arrivalRaw))
		if err != nil {
			return nil, err
		}
		sequence, err := strconv.Atoi(sequenceRaw)
		if err != nil {
			return nil, fmt.Errorf("stop sequence for trip %q: %w", tripID, err)
		}
		times[tripID] = append(times[tripID], StopTime{
			StopID:    stopID,
			Sequence:  sequence,
			Arrival:   arrival,
			Departure: departure,
		})
	}
	for tripID := range times {
		sort.Slice(times[tripID], func(i, j int) bool {
			return times[tripID][i].Sequence < times[tripID][j].Sequence
		})
	}
	return times, nil
}

func loadTransfers(fsys fs.FS) ([]Transfer, error) {
	rows, err := readCSV(fsys, "transfers.txt")
	if err != nil {
		return nil, err
	}
	transfers := make([]Transfer, 0, len(rows))
	for _, row := range rows {
		duration, _ := strconv.Atoi(firstNonEmpty(row["min_transfer_time"], "0"))
		transfers = append(transfers, Transfer{
			FromStopID: row["from_stop_id"],
			ToStopID:   row["to_stop_id"],
			Duration:   duration,
		})
	}
	return transfers, nil
}

func appendSameStationTransfers(stops map[string]Stop, transfers []Transfer) []Transfer {
	byName := map[string][]Stop{}
	for _, stop := range stops {
		if stop.Name == "" {
			continue
		}
		byName[stop.Name] = append(byName[stop.Name], stop)
	}

	seen := map[string]bool{}
	for _, transfer := range transfers {
		seen[transfer.FromStopID+"\x00"+transfer.ToStopID] = true
	}
	for _, namedStops := range byName {
		if len(namedStops) < 2 {
			continue
		}
		for _, from := range namedStops {
			for _, to := range namedStops {
				if from.ID == to.ID {
					continue
				}
				if HaversineMeters(from.Lat, from.Lon, to.Lat, to.Lon) > sameStationTransferMaxMeters {
					continue
				}
				key := from.ID + "\x00" + to.ID
				if seen[key] {
					continue
				}
				transfers = append(transfers, Transfer{
					FromStopID: from.ID,
					ToStopID:   to.ID,
					Duration:   sameStationTransferDuration,
				})
				seen[key] = true
			}
		}
	}
	return transfers
}

func readCSV(fsys fs.FS, name string) ([]map[string]string, error) {
	file, err := fsys.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	// Some GTFS-JP feeds (e.g. transitland-f-saga-current-jp) emit bare
	// double quotes inside otherwise unquoted fields, which trips the strict
	// RFC 4180 parser. LazyQuotes makes the reader treat those as literal
	// characters instead of refusing the file outright.
	reader.LazyQuotes = true
	header, err := reader.Read()
	if err != nil {
		return nil, err
	}
	for i := range header {
		header[i] = strings.TrimSpace(header[i])
	}
	// Strip a UTF-8 BOM from the first column header — some GTFS-JP feeds
	// emit one, which otherwise turns "stop_id" into "\ufeffstop_id".
	if len(header) > 0 {
		header[0] = strings.TrimPrefix(header[0], "\ufeff")
	}

	var rows []map[string]string
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		row := make(map[string]string, len(header))
		for i, name := range header {
			if i < len(record) {
				row[name] = strings.TrimSpace(record[i])
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func openCSV(fsys fs.FS, name string) (fs.File, *csv.Reader, map[string]int, error) {
	file, err := fsys.Open(name)
	if err != nil {
		return nil, nil, nil, err
	}

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true
	rawHeader, err := reader.Read()
	if err != nil {
		file.Close()
		return nil, nil, nil, err
	}
	header := make(map[string]int, len(rawHeader))
	for i, name := range rawHeader {
		name = strings.TrimSpace(name)
		if i == 0 {
			name = strings.TrimPrefix(name, "\ufeff")
		}
		header[name] = i
	}
	return file, reader, header, nil
}

func csvValue(record []string, index int) string {
	if index < 0 || index >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[index])
}

func parseGTFSTime(value string) (int, error) {
	parts := strings.Split(value, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid GTFS time %q", value)
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, err
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, err
	}
	second, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0, err
	}
	return int((time.Duration(hour)*time.Hour + time.Duration(minute)*time.Minute + time.Duration(second)*time.Second).Seconds()), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
