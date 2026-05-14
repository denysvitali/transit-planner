package router

import (
	"strings"
	"testing"
)

func TestMergeNamespacesIDs(t *testing.T) {
	a := &Feed{
		Stops: map[string]Stop{
			"s1": {ID: "s1", Name: "Tokyo", Lat: 35.681, Lon: 139.767},
		},
		Routes: map[string]Route{
			"r1": {ID: "r1", ShortName: "A"},
		},
		Trips: map[string]Trip{
			"t1": {
				ID:      "t1",
				RouteID: "r1",
				StopTimes: []StopTime{
					{StopID: "s1", Sequence: 1, Arrival: 0, Departure: 30},
				},
			},
		},
	}
	b := &Feed{
		Stops: map[string]Stop{
			"s1": {ID: "s1", Name: "Tokyo", Lat: 35.681, Lon: 139.767},
		},
		Routes: map[string]Route{
			"r1": {ID: "r1", ShortName: "B"},
		},
		Trips: map[string]Trip{
			"t1": {
				ID:      "t1",
				RouteID: "r1",
				StopTimes: []StopTime{
					{StopID: "s1", Sequence: 1, Arrival: 0, Departure: 30},
				},
			},
		},
	}

	merged := Merge(map[string]*Feed{"toei": a, "kobe": b})

	if _, ok := merged.Stops["toei:s1"]; !ok {
		t.Fatalf("expected toei:s1 in merged stops, have keys: %v", keys(merged.Stops))
	}
	if _, ok := merged.Stops["kobe:s1"]; !ok {
		t.Fatalf("expected kobe:s1 in merged stops, have keys: %v", keys(merged.Stops))
	}

	trip := merged.Trips["toei:t1"]
	if trip.RouteID != "toei:r1" {
		t.Errorf("trip route id = %q, want toei:r1", trip.RouteID)
	}
	if len(trip.StopTimes) != 1 || trip.StopTimes[0].StopID != "toei:s1" {
		t.Errorf("trip stop_times not namespaced: %+v", trip.StopTimes)
	}

	// appendSameStationTransfers should connect the two same-named "Tokyo"
	// stops across feeds — that's the bridge that makes a Tokyo→Osaka query
	// routable when each operator publishes its own GTFS.
	var foundCross bool
	for _, tr := range merged.Transfers {
		if (tr.FromStopID == "toei:s1" && tr.ToStopID == "kobe:s1") ||
			(tr.FromStopID == "kobe:s1" && tr.ToStopID == "toei:s1") {
			foundCross = true
			break
		}
	}
	if !foundCross {
		t.Errorf("missing cross-feed name transfer between toei:s1 and kobe:s1; transfers=%+v", merged.Transfers)
	}
}

func TestMergeEmpty(t *testing.T) {
	merged := Merge(nil)
	if merged == nil {
		t.Fatal("Merge(nil) returned nil")
	}
	if len(merged.Stops) != 0 || len(merged.Routes) != 0 || len(merged.Trips) != 0 {
		t.Errorf("expected empty feed, got %+v", merged)
	}
}

// TestMergeRoutesAcrossFeeds is the end-to-end sanity check: two independent
// feeds (think Tokyo operator + Osaka operator) share a same-named bridge
// stop, and the router stitches them via the name-based transfer that Merge
// regenerates after namespacing. This is the shape of a real Tokyo→Osaka
// query once both operators' feeds are fetched.
func TestMergeRoutesAcrossFeeds(t *testing.T) {
	tokyo := &Feed{
		Stops: map[string]Stop{
			"A":     {ID: "A", Name: "Shinjuku", Lat: 35.690, Lon: 139.700},
			"BRIDGE": {ID: "BRIDGE", Name: "Tokyo", Lat: 35.681, Lon: 139.767},
		},
		Routes: map[string]Route{
			"R1": {ID: "R1", ShortName: "Yamanote"},
		},
		Trips: map[string]Trip{
			"T1": {
				ID:      "T1",
				RouteID: "R1",
				StopTimes: []StopTime{
					{StopID: "A", Sequence: 1, Arrival: 8 * 3600, Departure: 8 * 3600},
					{StopID: "BRIDGE", Sequence: 2, Arrival: 8*3600 + 600, Departure: 8*3600 + 600},
				},
			},
		},
	}
	osaka := &Feed{
		Stops: map[string]Stop{
			"BRIDGE": {ID: "BRIDGE", Name: "Tokyo", Lat: 35.681, Lon: 139.767},
			"Z":      {ID: "Z", Name: "Umeda", Lat: 34.703, Lon: 135.499},
		},
		Routes: map[string]Route{
			"R2": {ID: "R2", ShortName: "Tokaido"},
		},
		Trips: map[string]Trip{
			"T2": {
				ID:      "T2",
				RouteID: "R2",
				StopTimes: []StopTime{
					{StopID: "BRIDGE", Sequence: 1, Arrival: 9 * 3600, Departure: 9 * 3600},
					{StopID: "Z", Sequence: 2, Arrival: 11 * 3600, Departure: 11 * 3600},
				},
			},
		},
	}

	merged := Merge(map[string]*Feed{"tokyo": tokyo, "osaka": osaka})
	engine := NewEngine(merged)

	itinerary, err := engine.Route("tokyo:A", "osaka:Z", 7*3600+30*60, Options{MaxTransfers: 3})
	if err != nil {
		t.Fatalf("Route: %v", err)
	}
	if itinerary.Arrival != 11*3600 {
		t.Errorf("arrival = %d, want %d (11:00)", itinerary.Arrival, 11*3600)
	}
	if itinerary.Transfers < 1 {
		t.Errorf("expected at least 1 transfer across feeds, got %d", itinerary.Transfers)
	}
}

func TestMergePanicsOnEmptyPrefix(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on empty prefix")
		} else if !strings.Contains(r.(string), "empty feed prefix") {
			t.Errorf("unexpected panic: %v", r)
		}
	}()
	Merge(map[string]*Feed{"": {}})
}

func keys(m map[string]Stop) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
