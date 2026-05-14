package index

import (
	"bytes"
	"testing"

	"github.com/denysvitali/transit-planner/router"
)

// sampleFeed builds a tiny synthetic GTFS feed entirely in memory so the
// tests do not depend on any on-disk fixtures.
func sampleFeed() *router.Feed {
	return &router.Feed{
		Stops: map[string]router.Stop{
			"A": {ID: "A", Name: "Alpha", Lat: 47.0, Lon: 8.0},
			"B": {ID: "B", Name: "Bravo", Lat: 47.1, Lon: 8.1},
			"C": {ID: "C", Name: "Charlie", Lat: 47.2, Lon: 8.2},
		},
		Routes: map[string]router.Route{
			"R1": {ID: "R1", ShortName: "1", LongName: "Line One", Type: 3},
			"R2": {ID: "R2", ShortName: "2", LongName: "Line Two", Type: 1},
		},
		Trips: map[string]router.Trip{
			"T1": {
				ID:        "T1",
				RouteID:   "R1",
				ServiceID: "WK",
				StopTimes: []router.StopTime{
					{StopID: "A", Sequence: 1, Arrival: 28800, Departure: 28830},
					{StopID: "B", Sequence: 2, Arrival: 28920, Departure: 28950},
					{StopID: "C", Sequence: 3, Arrival: 29040, Departure: 29040},
				},
			},
			"T2": {
				ID:        "T2",
				RouteID:   "R2",
				ServiceID: "WK",
				StopTimes: []router.StopTime{
					{StopID: "C", Sequence: 1, Arrival: 32400, Departure: 32400},
					{StopID: "A", Sequence: 2, Arrival: 32700, Departure: 32700},
				},
			},
		},
		Transfers: []router.Transfer{
			{FromStopID: "B", ToStopID: "C", Duration: 120},
		},
	}
}

func TestCompileShape(t *testing.T) {
	c := Compile(sampleFeed())

	if got, want := len(c.Stops), 3; got != want {
		t.Fatalf("stops: got %d, want %d", got, want)
	}
	if got, want := len(c.Routes), 2; got != want {
		t.Fatalf("routes: got %d, want %d", got, want)
	}
	if got, want := len(c.Trips), 2; got != want {
		t.Fatalf("trips: got %d, want %d", got, want)
	}
	if got, want := len(c.Transfers), 1; got != want {
		t.Fatalf("transfers: got %d, want %d", got, want)
	}

	// Stops are emitted in sorted key order, so A=0, B=1, C=2.
	if c.StopIndex("A") != 0 || c.StopIndex("B") != 1 || c.StopIndex("C") != 2 {
		t.Fatalf("stop indexes wrong: A=%d B=%d C=%d", c.StopIndex("A"), c.StopIndex("B"), c.StopIndex("C"))
	}
	if c.StopIndex("missing") != -1 {
		t.Fatalf("missing stop should be -1, got %d", c.StopIndex("missing"))
	}

	// Locate T1 in the compiled trips and check it references the dense ids.
	var t1 *CompiledTrip
	for i := range c.Trips {
		if c.Trips[i].ID == "T1" {
			t1 = &c.Trips[i]
			break
		}
	}
	if t1 == nil {
		t.Fatal("trip T1 missing after compile")
	}
	if t1.RouteIdx != c.RouteIndex("R1") {
		t.Fatalf("T1 route idx: got %d, want %d", t1.RouteIdx, c.RouteIndex("R1"))
	}
	if len(t1.StopTimes) != 3 {
		t.Fatalf("T1 stop_times: got %d, want 3", len(t1.StopTimes))
	}
	if t1.StopTimes[0].StopIdx != 0 || t1.StopTimes[2].StopIdx != 2 {
		t.Fatalf("T1 stop indices wrong: %+v", t1.StopTimes)
	}

	if c.Transfers[0].FromStopIdx != 1 || c.Transfers[0].ToStopIdx != 2 || c.Transfers[0].Duration != 120 {
		t.Fatalf("transfer mismatch: %+v", c.Transfers[0])
	}
}

func TestRoundTrip(t *testing.T) {
	original := Compile(sampleFeed())

	var buf bytes.Buffer
	n, err := original.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	if n != int64(buf.Len()) {
		t.Fatalf("WriteTo reported %d bytes but buffer has %d", n, buf.Len())
	}
	if buf.Len() < 8 {
		t.Fatalf("payload too small: %d bytes", buf.Len())
	}
	head := buf.Bytes()[:4]
	if string(head) != "TPFD" {
		t.Fatalf("magic header: got %q, want %q", head, "TPFD")
	}

	loaded, err := ReadFrom(&buf)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if len(loaded.Stops) != len(original.Stops) {
		t.Fatalf("stop count: got %d, want %d", len(loaded.Stops), len(original.Stops))
	}
	if len(loaded.Trips) != len(original.Trips) {
		t.Fatalf("trip count: got %d, want %d", len(loaded.Trips), len(original.Trips))
	}
	if len(loaded.Routes) != len(original.Routes) {
		t.Fatalf("route count: got %d, want %d", len(loaded.Routes), len(original.Routes))
	}
	if len(loaded.Transfers) != len(original.Transfers) {
		t.Fatalf("transfer count: got %d, want %d", len(loaded.Transfers), len(original.Transfers))
	}

	// Spot-check specific field values to confirm payload fidelity.
	if loaded.Stops[1].Name != "Bravo" {
		t.Fatalf("stop[1].Name: got %q, want %q", loaded.Stops[1].Name, "Bravo")
	}
	if loaded.Stops[2].Lat != 47.2 {
		t.Fatalf("stop[2].Lat: got %v, want %v", loaded.Stops[2].Lat, 47.2)
	}
	if loaded.StopIndex("C") != 2 {
		t.Fatalf("StopIndex(C) after load: got %d, want 2", loaded.StopIndex("C"))
	}
	if loaded.RouteIndex("R2") != 1 {
		t.Fatalf("RouteIndex(R2) after load: got %d, want 1", loaded.RouteIndex("R2"))
	}
}

func TestReadFromBadMagic(t *testing.T) {
	payload := append([]byte("ZZZZ"), 0, 0, 0, 1)
	if _, err := ReadFrom(bytes.NewReader(payload)); err == nil {
		t.Fatal("expected error for bad magic, got nil")
	}
}

func TestReadFromBadVersion(t *testing.T) {
	payload := append([]byte("TPFD"), 0, 0, 0, 99)
	if _, err := ReadFrom(bytes.NewReader(payload)); err == nil {
		t.Fatal("expected error for unsupported version, got nil")
	}
}

func TestCompileNil(t *testing.T) {
	c := Compile(nil)
	if c == nil {
		t.Fatal("Compile(nil) returned nil")
	}
	if len(c.Stops) != 0 || len(c.Routes) != 0 || len(c.Trips) != 0 || len(c.Transfers) != 0 {
		t.Fatalf("Compile(nil) should produce empty feed, got %+v", c)
	}
}
