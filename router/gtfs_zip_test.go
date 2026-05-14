package router

import (
	"strings"
	"testing"
)

// TestRouteKanazawaStationToOmicho verifies that the bundled Kanazawa Flat
// Bus feed can plan a trip between two of the city's flagship stops:
//   - 金沢駅 (Kanazawa Station, stop_id 101_01)
//   - 近江町市場・市姫神社 (Omicho Market / Ichihime Shrine, stop_id 108_01)
//
// Both are served by the 此花 (Konohana) loop (route_id "10"); a trip
// departing 08:24 from Kanazawa Station should reach Omicho around 08:33.
// The test guards against router regressions and confirms the feed stays
// usable as a generic, no-key fixture for Japanese transit routing.
func TestRouteKanazawaStationToOmicho(t *testing.T) {
	feed, err := LoadGTFSZip("../assets/real_gtfs/jp/kanazawa_flatbus/kanazawa-flatbus.zip")
	if err != nil {
		t.Fatalf("LoadGTFSZip: %v", err)
	}

	const (
		kanazawaStation = "101_01"
		omichoMarket    = "108_01"
	)

	if stop, ok := feed.Stops[kanazawaStation]; !ok {
		t.Fatalf("stop %q missing from Kanazawa Flat Bus feed", kanazawaStation)
	} else if !strings.Contains(stop.Name, "金沢駅") {
		t.Errorf("stop %q name = %q, want it to contain 金沢駅", kanazawaStation, stop.Name)
	}
	if stop, ok := feed.Stops[omichoMarket]; !ok {
		t.Fatalf("stop %q missing from Kanazawa Flat Bus feed", omichoMarket)
	} else if !strings.Contains(stop.Name, "近江町") {
		t.Errorf("stop %q name = %q, want it to contain 近江町", omichoMarket, stop.Name)
	}

	engine := NewEngine(feed)
	itinerary, err := engine.Route(
		kanazawaStation,
		omichoMarket,
		8*3600, // depart 08:00 — earliest Konohana run is 08:24
		Options{MaxTransfers: 1},
	)
	if err != nil {
		t.Fatalf("Route Kanazawa Station -> Omicho: %v", err)
	}
	if len(itinerary.Legs) == 0 {
		t.Fatal("itinerary has no legs")
	}

	var transitLegs int
	for _, leg := range itinerary.Legs {
		if leg.Mode == "transit" {
			transitLegs++
		}
	}
	if transitLegs == 0 {
		t.Fatal("no transit legs in itinerary; route should ride the Flat Bus")
	}
	// Konohana loop's first morning run boards at 08:24 and reaches Omicho
	// at 08:33. A wider window catches future timetable shifts without
	// losing the regression value.
	if itinerary.Arrival < 8*3600+24*60 || itinerary.Arrival > 12*3600 {
		t.Errorf("arrival = %d, want between 08:24 and 12:00", itinerary.Arrival)
	}
}

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
