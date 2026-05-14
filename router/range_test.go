package router

import (
	"testing"
)

func TestRouteRangeProducesParetoSet(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "stops.txt", `stop_id,stop_name,stop_lat,stop_lon
A,Alpha,46.0,7.0
D,Delta,46.3,7.3
`)
	writeFixture(t, dir, "routes.txt", `route_id,route_short_name,route_long_name,route_type
R1,1,Line 1,3
`)
	writeFixture(t, dir, "trips.txt", `route_id,service_id,trip_id
R1,weekday,T1
R1,weekday,T2
R1,weekday,T3
R1,weekday,T4
`)
	// Four trips:
	//   T1 dep 08:00 -> arr 08:30 (slow)
	//   T2 dep 08:05 -> arr 08:35 (slow)
	//   T3 dep 08:10 -> arr 08:25 (fast; dominates T1 & T2)
	//   T4 dep 08:20 -> arr 08:40 (later but slow; not dominated by T3)
	writeFixture(t, dir, "stop_times.txt", `trip_id,arrival_time,departure_time,stop_id,stop_sequence
T1,08:00:00,08:00:00,A,1
T1,08:30:00,08:30:00,D,2
T2,08:05:00,08:05:00,A,1
T2,08:35:00,08:35:00,D,2
T3,08:10:00,08:10:00,A,1
T3,08:25:00,08:25:00,D,2
T4,08:20:00,08:20:00,A,1
T4,08:40:00,08:40:00,D,2
`)

	feed, err := LoadGTFS(dir)
	if err != nil {
		t.Fatal(err)
	}
	engine := NewEngine(feed)

	results, err := engine.RouteRange("A", "D", 8*3600, 8*3600+30*60, Options{MaxTransfers: 0})
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 2 {
		t.Fatalf("got %d itineraries, want 2 (Pareto-optimal pairs)", len(results))
	}

	type pair struct{ dep, arr int }
	want := []pair{
		{dep: 8*3600 + 10*60, arr: 8*3600 + 25*60},
		{dep: 8*3600 + 20*60, arr: 8*3600 + 40*60},
	}
	for i, r := range results {
		if len(r.Legs) == 0 {
			t.Fatalf("itinerary %d has no legs", i)
		}
		dep := r.Legs[0].Departure
		if dep != want[i].dep || r.Arrival != want[i].arr {
			t.Fatalf("itinerary %d = (dep=%d, arr=%d), want (dep=%d, arr=%d)",
				i, dep, r.Arrival, want[i].dep, want[i].arr)
		}
	}
}

func TestRouteRangeEmptyWindow(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "stops.txt", `stop_id,stop_name,stop_lat,stop_lon
A,Alpha,46.0,7.0
D,Delta,46.3,7.3
`)
	writeFixture(t, dir, "routes.txt", `route_id,route_short_name,route_long_name,route_type
R1,1,Line 1,3
`)
	writeFixture(t, dir, "trips.txt", `route_id,service_id,trip_id
R1,weekday,T1
`)
	writeFixture(t, dir, "stop_times.txt", `trip_id,arrival_time,departure_time,stop_id,stop_sequence
T1,09:00:00,09:00:00,A,1
T1,09:30:00,09:30:00,D,2
`)

	feed, err := LoadGTFS(dir)
	if err != nil {
		t.Fatal(err)
	}
	engine := NewEngine(feed)

	results, err := engine.RouteRange("A", "D", 8*3600, 8*3600+30*60, Options{MaxTransfers: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected no itineraries outside window, got %d", len(results))
	}
}

func TestRouteRangeUnknownStop(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "stops.txt", `stop_id,stop_name,stop_lat,stop_lon
A,Alpha,46.0,7.0
`)
	writeFixture(t, dir, "routes.txt", `route_id,route_short_name,route_long_name,route_type
`)
	writeFixture(t, dir, "trips.txt", `route_id,service_id,trip_id
`)
	writeFixture(t, dir, "stop_times.txt", `trip_id,arrival_time,departure_time,stop_id,stop_sequence
`)

	feed, err := LoadGTFS(dir)
	if err != nil {
		t.Fatal(err)
	}
	engine := NewEngine(feed)

	if _, err := engine.RouteRange("A", "ZZZ", 0, 3600, Options{}); err == nil {
		t.Fatal("expected error for unknown destination")
	}
	if _, err := engine.RouteRange("ZZZ", "A", 0, 3600, Options{}); err == nil {
		t.Fatal("expected error for unknown origin")
	}
}
