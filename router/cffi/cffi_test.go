//go:build cgo

package cffi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeFixture(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeMiniFeed(t *testing.T) string {
	t.Helper()
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
	return dir
}

// TestRouteJSONRoundTrip exercises the pure-Go core that powers TP_Route.
// We can't `import "C"` from a _test.go file in a package that already uses
// cgo, so the test drives the JSON helper instead. This still validates the
// full request → response shape that the FFI exposes.
func TestRouteJSONRoundTrip(t *testing.T) {
	dir := writeMiniFeed(t)

	req := routeRequest{
		FeedDir:      dir,
		From:         "A",
		To:           "D",
		Departure:    7*3600 + 55*60,
		MaxTransfers: 2,
	}
	reqBytes, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	respJSON := RouteJSON(string(reqBytes))

	var maybeErr errorResponse
	if err := json.Unmarshal([]byte(respJSON), &maybeErr); err == nil && maybeErr.Error != "" {
		t.Fatalf("routeJSON returned error: %s", maybeErr.Error)
	}

	var resp routeResponse
	if err := json.Unmarshal([]byte(respJSON), &resp); err != nil {
		t.Fatalf("unmarshal response: %v (raw=%s)", err, respJSON)
	}

	if want := 8*3600 + 25*60; resp.Arrival != want {
		t.Fatalf("arrival = %d, want %d", resp.Arrival, want)
	}
	if resp.Transfers != 1 {
		t.Fatalf("transfers = %d, want 1", resp.Transfers)
	}
	if len(resp.Legs) != 3 {
		t.Fatalf("legs = %d, want 3", len(resp.Legs))
	}
	if resp.Legs[1].Mode != "walk" {
		t.Fatalf("middle leg mode = %q, want walk", resp.Legs[1].Mode)
	}
}

func TestRouteJSONMissingFeed(t *testing.T) {
	req := routeRequest{
		FeedDir: filepath.Join(t.TempDir(), "does-not-exist"),
		From:    "A",
		To:      "B",
	}
	reqBytes, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	respJSON := RouteJSON(string(reqBytes))

	var resp errorResponse
	if err := json.Unmarshal([]byte(respJSON), &resp); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if resp.Error == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestRouteJSONInvalidPayload(t *testing.T) {
	respJSON := RouteJSON("not-json")

	var resp errorResponse
	if err := json.Unmarshal([]byte(respJSON), &resp); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if resp.Error == "" {
		t.Fatal("expected non-empty error message for invalid JSON")
	}
}

func TestRouteJSONEmptyPayload(t *testing.T) {
	respJSON := RouteJSON("")

	var resp errorResponse
	if err := json.Unmarshal([]byte(respJSON), &resp); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if resp.Error == "" {
		t.Fatal("expected non-empty error message for empty payload")
	}
}
