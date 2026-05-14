package router

import (
	"strings"
	"testing"
)

// TestLoadGTFSZipToeiTrain exercises LoadGTFSZip against the vendored
// Tokyo Metropolitan Bureau of Transportation feed under
// assets/sample_toei_train/Toei-Train-GTFS.zip. The numbers below are a
// loose sanity check against the 2026-03-14 timetable; tighten them when
// the vendored fixture is refreshed.
func TestLoadGTFSZipToeiTrain(t *testing.T) {
	feed, err := LoadGTFSZip("../assets/sample_toei_train/Toei-Train-GTFS.zip")
	if err != nil {
		t.Fatalf("LoadGTFSZip: %v", err)
	}

	if got := len(feed.Stops); got < 100 {
		t.Errorf("stops = %d, want >= 100", got)
	}
	if got := len(feed.Routes); got < 4 {
		t.Errorf("routes = %d, want >= 4 (Toei runs at least 4 subway lines)", got)
	}
	if got := len(feed.Trips); got < 1000 {
		t.Errorf("trips = %d, want >= 1000", got)
	}

	// Asakusa Line (浅草線) — route_id "1" in the Toei feed — must exist and
	// have a Japanese long name. This guards against header-mapping or
	// UTF-8/BOM regressions in the CSV reader.
	asakusa, ok := feed.Routes["1"]
	if !ok {
		t.Fatal("route_id=1 (Asakusa Line) missing from Toei Train feed")
	}
	if asakusa.LongName == "" {
		t.Error("Asakusa Line long name is empty")
	}
	if !strings.Contains(asakusa.LongName, "浅草") {
		t.Errorf("Asakusa Line long name = %q, want it to contain 浅草", asakusa.LongName)
	}

	// At least one trip must have at least two timed stops — the empty-time
	// skip in loadStopTimes must not have nuked everything.
	var hasMultiStopTrip bool
	for _, trip := range feed.Trips {
		if len(trip.StopTimes) >= 2 {
			hasMultiStopTrip = true
			break
		}
	}
	if !hasMultiStopTrip {
		t.Error("no trip has >= 2 timed stop_times — parser may have dropped too many rows")
	}
}
