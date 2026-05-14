package router

import (
	"testing"
)

func TestRouteMultiParetoIncludesFastAndZeroTransfer(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "stops.txt", `stop_id,stop_name,stop_lat,stop_lon
A,Alpha,46.0,7.0
B,Beta,46.1,7.1
C,Central,46.2,7.2
D,Delta,46.3,7.3
`)
	writeFixture(t, dir, "routes.txt", `route_id,route_short_name,route_long_name,route_type
R1,1,Line 1,3
R2,2,Line 2,3
R3,3,Line 3,3
`)
	writeFixture(t, dir, "trips.txt", `route_id,service_id,trip_id
R1,weekday,T1
R2,weekday,T2
R3,weekday,T3
`)
	// T1: A -> B (fast first leg)
	// T2: C -> D (fast second leg)
	// T3: A -> D direct (slow, no transfer)
	writeFixture(t, dir, "stop_times.txt", `trip_id,arrival_time,departure_time,stop_id,stop_sequence
T1,08:00:00,08:00:00,A,1
T1,08:05:00,08:05:00,B,2
T2,08:08:00,08:08:00,C,1
T2,08:20:00,08:20:00,D,2
T3,08:00:00,08:00:00,A,1
T3,08:40:00,08:40:00,D,2
`)
	writeFixture(t, dir, "transfers.txt", `from_stop_id,to_stop_id,transfer_type,min_transfer_time
B,C,2,60
`)

	feed, err := LoadGTFS(dir)
	if err != nil {
		t.Fatal(err)
	}
	engine := NewEngine(feed)

	results, err := engine.RouteMulti("A", "D", 7*3600+55*60, MultiOptions{MaxTransfers: 3})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) < 2 {
		t.Fatalf("expected at least 2 Pareto itineraries, got %d", len(results))
	}

	var foundFast, foundDirect bool
	for _, it := range results {
		switch {
		case it.Arrival == 8*3600+20*60 && it.Transfers == 1:
			foundFast = true
		case it.Arrival == 8*3600+40*60 && it.Transfers == 0:
			foundDirect = true
		}
	}
	if !foundFast {
		t.Errorf("expected fast itinerary (arrival 08:20, 1 transfer) in result set; got %+v", results)
	}
	if !foundDirect {
		t.Errorf("expected direct itinerary (arrival 08:40, 0 transfers) in result set; got %+v", results)
	}
}

func TestRouteMultiExcludesDominated(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "stops.txt", `stop_id,stop_name,stop_lat,stop_lon
A,Alpha,46.0,7.0
B,Beta,46.1,7.1
C,Central,46.2,7.2
D,Delta,46.3,7.3
`)
	writeFixture(t, dir, "routes.txt", `route_id,route_short_name,route_long_name,route_type
R1,1,Line 1,3
R2,2,Line 2,3
R3,3,Line 3,3
R4,4,Line 4,3
`)
	writeFixture(t, dir, "trips.txt", `route_id,service_id,trip_id
R1,weekday,T1
R2,weekday,T2
R3,weekday,T3
R4,weekday,T4
`)
	// T1+T2 via B->C transfer: arrival 08:20, 1 transfer, walk 60.
	// T3+T2 via B2->C transfer: arrival 08:30 (slower), 1 transfer, walk 60. Strictly dominated.
	writeFixture(t, dir, "stop_times.txt", `trip_id,arrival_time,departure_time,stop_id,stop_sequence
T1,08:00:00,08:00:00,A,1
T1,08:05:00,08:05:00,B,2
T2,08:08:00,08:08:00,C,1
T2,08:20:00,08:20:00,D,2
T3,08:00:00,08:00:00,A,1
T3,08:15:00,08:15:00,B,2
T4,08:18:00,08:18:00,C,1
T4,08:30:00,08:30:00,D,2
`)
	writeFixture(t, dir, "transfers.txt", `from_stop_id,to_stop_id,transfer_type,min_transfer_time
B,C,2,60
`)

	feed, err := LoadGTFS(dir)
	if err != nil {
		t.Fatal(err)
	}
	engine := NewEngine(feed)

	results, err := engine.RouteMulti("A", "D", 7*3600+55*60, MultiOptions{MaxTransfers: 3})
	if err != nil {
		t.Fatal(err)
	}

	for _, it := range results {
		if it.Arrival == 8*3600+30*60 && it.Transfers == 1 {
			t.Errorf("dominated itinerary (arrival 08:30, 1 transfer) should not be in result set; got %+v", results)
		}
	}

	var foundFast bool
	for _, it := range results {
		if it.Arrival == 8*3600+20*60 && it.Transfers == 1 {
			foundFast = true
		}
	}
	if !foundFast {
		t.Errorf("expected best itinerary (arrival 08:20, 1 transfer) in result set; got %+v", results)
	}
}
