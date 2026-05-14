package router

import (
	"math"
	"testing"
)

func TestHaversineMetersBernZurich(t *testing.T) {
	// Bern (Bundesplatz) and Zürich HB are roughly 95–96 km apart.
	const (
		bernLat, bernLon     = 46.9469, 7.4446
		zurichLat, zurichLon = 47.3769, 8.5417
	)
	got := HaversineMeters(bernLat, bernLon, zurichLat, zurichLon)
	const want = 95500.0 // ~95.5 km reference
	tolerance := want * 0.02
	if math.Abs(got-want) > tolerance {
		t.Fatalf("HaversineMeters Bern→Zürich = %.0f m, want %.0f ±%.0f m", got, want, tolerance)
	}
}

func TestHaversineMetersSymmetricAndZero(t *testing.T) {
	const lat, lon = 46.9469, 7.4446
	if d := HaversineMeters(lat, lon, lat, lon); d != 0 {
		t.Fatalf("HaversineMeters of identical points = %v, want 0", d)
	}
	a := HaversineMeters(46.9, 7.4, 47.3, 8.5)
	b := HaversineMeters(47.3, 8.5, 46.9, 7.4)
	if math.Abs(a-b) > 1e-9 {
		t.Fatalf("haversine not symmetric: %v vs %v", a, b)
	}
}

func newTestFeed() *Feed {
	stops := map[string]Stop{
		"bern":   {ID: "bern", Name: "Bern", Lat: 46.9469, Lon: 7.4446},
		"zurich": {ID: "zurich", Name: "Zürich HB", Lat: 47.3769, Lon: 8.5417},
		"thun":   {ID: "thun", Name: "Thun", Lat: 46.7541, Lon: 7.6296},
		"basel":  {ID: "basel", Name: "Basel SBB", Lat: 47.5476, Lon: 7.5896},
		"geneva": {ID: "geneva", Name: "Genève", Lat: 46.2103, Lon: 6.1422},
	}
	return &Feed{Stops: stops}
}

func TestNearbyStopsRadiusAndOrder(t *testing.T) {
	feed := newTestFeed()
	// Query near Bern: Thun (~25 km) should be inside 40 km, Bern itself ~0 m.
	results := feed.NearbyStops(46.9469, 7.4446, 40000)
	if len(results) != 2 {
		t.Fatalf("expected 2 stops within 40 km of Bern, got %d: %+v", len(results), results)
	}
	if results[0].Stop.ID != "bern" {
		t.Fatalf("expected nearest stop to be bern, got %q", results[0].Stop.ID)
	}
	if results[1].Stop.ID != "thun" {
		t.Fatalf("expected second stop to be thun, got %q", results[1].Stop.ID)
	}
	if results[0].DistanceMeters > results[1].DistanceMeters {
		t.Fatalf("results not sorted by ascending distance: %v", results)
	}
	if results[0].DistanceMeters > 1 {
		t.Fatalf("distance from Bern to itself should be ~0, got %v", results[0].DistanceMeters)
	}
}

func TestNearbyStopsEmptyWhenOutOfRange(t *testing.T) {
	feed := newTestFeed()
	// 1 km around an empty patch of Mediterranean: no stops.
	results := feed.NearbyStops(40.0, 5.0, 1000)
	if len(results) != 0 {
		t.Fatalf("expected no stops, got %d: %+v", len(results), results)
	}
}

func TestNearbyStopsNilOrEmptyFeed(t *testing.T) {
	var feed *Feed
	if got := feed.NearbyStops(0, 0, 1000); got != nil {
		t.Fatalf("expected nil for nil feed, got %v", got)
	}
	empty := &Feed{Stops: map[string]Stop{}}
	if got := empty.NearbyStops(0, 0, 1000); len(got) != 0 {
		t.Fatalf("expected empty for empty feed, got %v", got)
	}
}

func TestNearestStop(t *testing.T) {
	feed := newTestFeed()
	// Query near Zürich coordinates.
	stop, dist, ok := feed.NearestStop(47.3769, 8.5417)
	if !ok {
		t.Fatalf("expected ok=true with stops in feed")
	}
	if stop.ID != "zurich" {
		t.Fatalf("expected nearest=zurich, got %q", stop.ID)
	}
	if dist > 1 {
		t.Fatalf("expected ~0 m distance, got %v", dist)
	}
}

func TestNearestStopEmptyFeed(t *testing.T) {
	empty := &Feed{Stops: map[string]Stop{}}
	if _, _, ok := empty.NearestStop(0, 0); ok {
		t.Fatalf("expected ok=false for empty feed")
	}
	var nilFeed *Feed
	if _, _, ok := nilFeed.NearestStop(0, 0); ok {
		t.Fatalf("expected ok=false for nil feed")
	}
}
