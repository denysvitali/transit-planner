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

// TestRouteJSONRoundTrip drives the JSON entry point that powers the FFI
// surface. The cgo wrapper in cmd/libtransitplanner only forwards strings to
// and from this function, so testing it here covers the production path
// without needing a C toolchain on CI.
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
	if resp.Legs[0].RouteName != "1" {
		t.Fatalf("route name = %q, want short name 1", resp.Legs[0].RouteName)
	}
}

func TestRouteJSONUsesCoordinateAccessAndEgressCandidates(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "stops.txt", `stop_id,stop_name,stop_lat,stop_lon
O1,Origin nearest,0,0
O2,Origin service,0,0.01
D2,Destination service,0,0.03
D1,Destination nearest,0,0.04
`)
	writeFixture(t, dir, "routes.txt", `route_id,route_short_name,route_long_name,route_type
R1,1,Line 1,3
`)
	writeFixture(t, dir, "trips.txt", `route_id,service_id,trip_id
R1,weekday,T1
`)
	writeFixture(t, dir, "stop_times.txt", `trip_id,arrival_time,departure_time,stop_id,stop_sequence
T1,08:10:00,08:10:00,O2,1
T1,08:20:00,08:20:00,D2,2
`)

	fromLat, fromLon := 0.0, 0.0
	toLat, toLon := 0.0, 0.04
	req := routeRequest{
		FeedDir:      dir,
		From:         "O1",
		To:           "D1",
		FromName:     "Origin point",
		FromLat:      &fromLat,
		FromLon:      &fromLon,
		ToName:       "Destination point",
		ToLat:        &toLat,
		ToLon:        &toLon,
		Departure:    7*3600 + 55*60,
		MaxTransfers: 0,
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
	if len(resp.Legs) != 3 {
		t.Fatalf("legs = %d, want access walk, transit, egress walk", len(resp.Legs))
	}
	if resp.Legs[0].Mode != "walk" || resp.Legs[0].FromStop.ID != "__origin" || resp.Legs[0].ToStop.ID != "O2" {
		t.Fatalf("access leg = %#v, want synthetic origin walk to O2", resp.Legs[0])
	}
	if resp.Legs[1].Mode != "transit" || resp.Legs[1].FromStop.ID != "O2" || resp.Legs[1].ToStop.ID != "D2" {
		t.Fatalf("transit leg = %#v, want O2 to D2", resp.Legs[1])
	}
	if resp.Legs[2].Mode != "walk" || resp.Legs[2].FromStop.ID != "D2" || resp.Legs[2].ToStop.ID != "__destination" {
		t.Fatalf("egress leg = %#v, want D2 walk to synthetic destination", resp.Legs[2])
	}
	if resp.Arrival <= 8*3600+20*60 {
		t.Fatalf("arrival = %d, want after transit arrival due to egress walk", resp.Arrival)
	}
}

func TestRouteJSONWalksWhenCoordinateEndpointsHaveNoNearbyStops(t *testing.T) {
	dir := writeMiniFeed(t)

	fromLat, fromLon := 20.0, 20.0
	toLat, toLon := 20.0, 20.01
	req := routeRequest{
		FeedDir:      dir,
		From:         "A",
		To:           "D",
		FromName:     "Remote origin",
		FromLat:      &fromLat,
		FromLon:      &fromLon,
		ToName:       "Remote destination",
		ToLat:        &toLat,
		ToLon:        &toLon,
		Departure:    8 * 3600,
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
	if resp.Transfers != 0 {
		t.Fatalf("transfers = %d, want 0", resp.Transfers)
	}
	if len(resp.Legs) != 1 {
		t.Fatalf("legs = %d, want direct walk only", len(resp.Legs))
	}
	leg := resp.Legs[0]
	if leg.Mode != "walk" || leg.FromStop.ID != "__origin" || leg.ToStop.ID != "__destination" {
		t.Fatalf("leg = %#v, want synthetic direct walk", leg)
	}
	if leg.Arrival <= leg.Departure {
		t.Fatalf("walk arrival = %d, want after departure %d", leg.Arrival, leg.Departure)
	}
	if leg.FromStop.Name != "Remote origin" || leg.ToStop.Name != "Remote destination" {
		t.Fatalf("walk names = %q -> %q, want remote endpoint names", leg.FromStop.Name, leg.ToStop.Name)
	}
}

// TestRouteJSONFeedZipToei wires the JSON surface through the real Toei feed
// vendored under assets/sample_toei_train/. It only checks that the request
// is accepted and a JSON response is produced — the underlying router is
// already covered by router/router_test.go. The point here is to prove that
// "load a GTFS zip and route through the FFI" works end-to-end.
func TestRouteJSONFeedZipToei(t *testing.T) {
	const zipPath = "../../assets/sample_toei_train/Toei-Train-GTFS.zip"
	if _, err := os.Stat(zipPath); err != nil {
		t.Skipf("vendored Toei zip missing: %v", err)
	}

	// Asakusa Line stop_ids 101..127 from west (西馬込) to east (押上). We
	// pick a same-line origin/destination so the test does not depend on
	// transfer modelling; any non-zero arrival time and any non-empty legs
	// slice is enough to demonstrate the wiring.
	req := routeRequest{
		FeedZip:      zipPath,
		From:         "101",
		To:           "108",
		Departure:    5 * 3600,
		MaxTransfers: 0,
	}
	reqBytes, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	respJSON := RouteJSON(string(reqBytes))

	var maybeErr errorResponse
	if err := json.Unmarshal([]byte(respJSON), &maybeErr); err == nil && maybeErr.Error != "" {
		t.Fatalf("Toei route returned error: %s", maybeErr.Error)
	}
	var resp routeResponse
	if err := json.Unmarshal([]byte(respJSON), &resp); err != nil {
		t.Fatalf("unmarshal response: %v (raw=%s)", err, respJSON)
	}
	if resp.Arrival <= 5*3600 {
		t.Fatalf("arrival = %d, expected > departure (%d)", resp.Arrival, 5*3600)
	}
	if len(resp.Legs) == 0 {
		t.Fatal("expected at least one leg in the Toei route")
	}
}

func TestRouteJSONFeedZipToeiCrossLineUsesSyntheticInterchange(t *testing.T) {
	const zipPath = "../../assets/sample_toei_train/Toei-Train-GTFS.zip"
	if _, err := os.Stat(zipPath); err != nil {
		t.Skipf("vendored Toei zip missing: %v", err)
	}

	// The Toei feed models interchanges as separate stop IDs with matching
	// names and omits transfers.txt. This route changes from Asakusa Line
	// Mita (108) to Mita Line Mita (204) before continuing to Hibiya (208).
	req := routeRequest{
		FeedZip:      zipPath,
		From:         "101",
		To:           "208",
		Departure:    8 * 3600,
		MaxTransfers: 2,
	}
	reqBytes, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	respJSON := RouteJSON(string(reqBytes))

	var maybeErr errorResponse
	if err := json.Unmarshal([]byte(respJSON), &maybeErr); err == nil && maybeErr.Error != "" {
		t.Fatalf("Toei cross-line route returned error: %s", maybeErr.Error)
	}
	var resp routeResponse
	if err := json.Unmarshal([]byte(respJSON), &resp); err != nil {
		t.Fatalf("unmarshal response: %v (raw=%s)", err, respJSON)
	}
	if len(resp.Legs) < 3 {
		t.Fatalf("legs = %d, want at least 3", len(resp.Legs))
	}
	foundWalk := false
	for _, leg := range resp.Legs {
		if leg.Mode == "walk" && leg.FromStop.Name == leg.ToStop.Name {
			foundWalk = true
			break
		}
	}
	if !foundWalk {
		t.Fatalf("expected same-station walk leg, got %#v", resp.Legs)
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

// TestHandleLifecycle exercises the OpenJSON / StopsJSON / RouteJSON (with
// a handle) / CloseJSON pipeline against the vendored Toei subway feed —
// the exact code path the Flutter app will use.
func TestHandleLifecycle(t *testing.T) {
	const zipPath = "../../assets/sample_toei_train/Toei-Train-GTFS.zip"
	if _, err := os.Stat(zipPath); err != nil {
		t.Skipf("vendored Toei zip missing: %v", err)
	}

	openReq, _ := json.Marshal(openRequest{FeedZip: zipPath})
	openResp := OpenJSON(string(openReq))
	var open openResponse
	if err := json.Unmarshal([]byte(openResp), &open); err != nil {
		t.Fatalf("unmarshal open response: %v (raw=%s)", err, openResp)
	}
	if open.Handle == 0 {
		t.Fatalf("OpenJSON returned empty handle: %s", openResp)
	}
	t.Cleanup(func() {
		closeReq, _ := json.Marshal(closeRequest{Handle: open.Handle})
		CloseJSON(string(closeReq))
	})

	stopsReq, _ := json.Marshal(stopsRequest{Handle: open.Handle})
	var stops stopsResponse
	if err := json.Unmarshal([]byte(StopsJSON(string(stopsReq))), &stops); err != nil {
		t.Fatalf("unmarshal stops response: %v", err)
	}
	if len(stops.Stops) < 100 {
		t.Fatalf("stops = %d, want >= 100", len(stops.Stops))
	}

	routeReq, _ := json.Marshal(routeRequest{
		Handle:    open.Handle,
		From:      "101",
		To:        "108",
		Departure: 5 * 3600,
	})
	var route routeResponse
	if err := json.Unmarshal([]byte(RouteJSON(string(routeReq))), &route); err != nil {
		t.Fatalf("unmarshal route response: %v", err)
	}
	if route.Arrival <= 5*3600 {
		t.Fatalf("arrival = %d, want > %d", route.Arrival, 5*3600)
	}

	// After close the handle must no longer route.
	closeReq, _ := json.Marshal(closeRequest{Handle: open.Handle})
	if got := CloseJSON(string(closeReq)); got != `{}` {
		t.Fatalf("CloseJSON = %q, want {}", got)
	}
	var afterErr errorResponse
	json.Unmarshal([]byte(RouteJSON(string(routeReq))), &afterErr)
	if afterErr.Error == "" {
		t.Fatal("expected error routing against closed handle")
	}
}

func TestRouteJSONRejectsBothFeedSources(t *testing.T) {
	req := routeRequest{
		FeedDir: "/tmp/whatever",
		FeedZip: "/tmp/whatever.zip",
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
		t.Fatal("expected error when both feedDir and feedZip are set")
	}
}
