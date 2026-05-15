package router

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// TestLoadGTFSZipUsesNestedRoot proves that a GTFS zip with everything nested
// under a single top-level directory still loads. Several GTFS-JP feeds we
// pull from Transitland (sakaisi-bus, gunkanjima, miike-shimabara, ...) are
// packaged this way, and used to fail with "open stops.txt: file does not
// exist".
func TestLoadGTFSZipUsesNestedRoot(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "nested.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	write := func(name, body string) {
		t.Helper()
		fw, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(fw, body); err != nil {
			t.Fatal(err)
		}
	}
	write("operator-name/stops.txt", `stop_id,stop_name,stop_lat,stop_lon
A,Alpha,46.0,7.0
B,Beta,46.1,7.1
`)
	write("operator-name/routes.txt", `route_id,route_short_name,route_long_name,route_type
R1,1,Line 1,3
`)
	write("operator-name/trips.txt", `route_id,service_id,trip_id
R1,weekday,T1
`)
	write("operator-name/stop_times.txt", `trip_id,arrival_time,departure_time,stop_id,stop_sequence
T1,08:00:00,08:00:00,A,1
T1,08:10:00,08:10:00,B,2
`)
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	feed, err := LoadGTFSZip(zipPath)
	if err != nil {
		t.Fatalf("LoadGTFSZip: %v", err)
	}
	if _, ok := feed.Stops["A"]; !ok {
		t.Fatalf("stop A missing; stops = %#v", feed.Stops)
	}
	if _, ok := feed.Stops["B"]; !ok {
		t.Fatalf("stop B missing; stops = %#v", feed.Stops)
	}
}

// TestLoadGTFSZipEmptyReturnsStopsTxtMissing keeps the historical error shape
// for the genuinely-broken case (no stops.txt anywhere): the cffi layer relies
// on the string starting with "open stops.txt" for its per-feed skip warning.
func TestLoadGTFSZipEmptyReturnsStopsTxtMissing(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "empty.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = LoadGTFSZip(zipPath)
	if err == nil {
		t.Fatal("expected error for empty zip")
	}
	if got := err.Error(); got != "open stops.txt: file does not exist" {
		t.Fatalf("err = %q, want %q", got, "open stops.txt: file does not exist")
	}
}

// TestLoadGTFSToleratesBareQuotes proves that the loader survives the
// non-RFC-4180 CSVs that some GTFS-JP feeds emit, where a stray `"` inside an
// otherwise-unquoted field used to abort the parse with `parse error on line
// 1, column 4: bare " in non-quoted-field`.
func TestLoadGTFSToleratesBareQuotes(t *testing.T) {
	dir := t.TempDir()
	must := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// The literal `26"` appears in real-world Saga-current-jp stop_name rows.
	must("stops.txt", `stop_id,stop_name,stop_lat,stop_lon
A,Bus stop 26" west,33.2,130.3
B,Beta,33.3,130.4
`)
	must("routes.txt", `route_id,route_short_name,route_long_name,route_type
R1,1,Line 1,3
`)
	must("trips.txt", `route_id,service_id,trip_id
R1,weekday,T1
`)
	must("stop_times.txt", `trip_id,arrival_time,departure_time,stop_id,stop_sequence
T1,08:00:00,08:00:00,A,1
T1,08:10:00,08:10:00,B,2
`)

	feed, err := LoadGTFS(dir)
	if err != nil {
		t.Fatalf("LoadGTFS: %v", err)
	}
	stop, ok := feed.Stops["A"]
	if !ok {
		t.Fatalf("stop A missing; stops = %#v", feed.Stops)
	}
	if stop.Name == "" {
		t.Fatalf("stop A name is empty")
	}
}
