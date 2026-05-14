package router

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRouteEarliestArrivalWithTransfer(t *testing.T) {
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
`)
	writeFixture(t, dir, "trips.txt", `route_id,service_id,trip_id
R1,weekday,T1
R2,weekday,T2
`)
	writeFixture(t, dir, "stop_times.txt", `trip_id,arrival_time,departure_time,stop_id,stop_sequence
T1,08:00:00,08:00:00,A,1
T1,08:10:00,08:10:00,B,2
T2,08:12:00,08:12:00,C,1
T2,08:25:00,08:25:00,D,2
`)
	writeFixture(t, dir, "transfers.txt", `from_stop_id,to_stop_id,transfer_type,min_transfer_time
B,C,2,60
`)

	feed, err := LoadGTFS(dir)
	if err != nil {
		t.Fatal(err)
	}
	engine := NewEngine(feed)

	itinerary, err := engine.Route("A", "D", 7*3600+55*60, Options{MaxTransfers: 2})
	if err != nil {
		t.Fatal(err)
	}

	if itinerary.Arrival != 8*3600+25*60 {
		t.Fatalf("arrival = %d, want %d", itinerary.Arrival, 8*3600+25*60)
	}
	if itinerary.Transfers != 1 {
		t.Fatalf("transfers = %d, want 1", itinerary.Transfers)
	}
	if len(itinerary.Legs) != 3 {
		t.Fatalf("legs = %d, want 3", len(itinerary.Legs))
	}
	if itinerary.Legs[1].Mode != "walk" {
		t.Fatalf("middle leg mode = %q, want walk", itinerary.Legs[1].Mode)
	}
}

func writeFixture(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
