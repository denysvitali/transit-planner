package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFixture(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// buildFixture mirrors the GTFS fixture shape used in router/router_test.go.
func buildFixture(t *testing.T) string {
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

func TestRunInfo(t *testing.T) {
	dir := buildFixture(t)
	var out bytes.Buffer
	if err := run([]string{"info", "-feed", dir}, &out); err != nil {
		t.Fatalf("run info: %v", err)
	}
	got := out.String()
	for _, want := range []string{"stops:", "routes:", "trips:", "transfers:", "4", "2", "1"} {
		if !strings.Contains(got, want) {
			t.Errorf("info output missing %q\nGot:\n%s", want, got)
		}
	}
}

func TestRunStops(t *testing.T) {
	dir := buildFixture(t)
	var out bytes.Buffer
	if err := run([]string{"stops", "-feed", dir, "-prefix", "a"}, &out); err != nil {
		t.Fatalf("run stops: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "Alpha") {
		t.Errorf("stops output missing Alpha\nGot:\n%s", got)
	}
	if strings.Contains(got, "Beta") {
		t.Errorf("stops output should not include Beta for prefix=a\nGot:\n%s", got)
	}
}

func TestRunStopsNoPrefix(t *testing.T) {
	dir := buildFixture(t)
	var out bytes.Buffer
	if err := run([]string{"stops", "-feed", dir}, &out); err != nil {
		t.Fatalf("run stops: %v", err)
	}
	got := out.String()
	for _, name := range []string{"Alpha", "Beta", "Central", "Delta"} {
		if !strings.Contains(got, name) {
			t.Errorf("stops output missing %q\nGot:\n%s", name, got)
		}
	}
}

func TestRunRoute(t *testing.T) {
	dir := buildFixture(t)
	var out bytes.Buffer
	err := run([]string{
		"route",
		"-feed", dir,
		"-from", "A",
		"-to", "D",
		"-depart", "07:55",
		"-max-transfers", "2",
	}, &out)
	if err != nil {
		t.Fatalf("run route: %v", err)
	}
	got := out.String()
	for _, want := range []string{"Arrival", "Alpha", "Delta", "08:25:00"} {
		if !strings.Contains(got, want) {
			t.Errorf("route output missing %q\nGot:\n%s", want, got)
		}
	}
}

func TestRunUnknownSubcommand(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"nope"}, &out)
	if err == nil {
		t.Fatal("expected error for unknown subcommand")
	}
}

func TestRunRouteMissingFlags(t *testing.T) {
	var out bytes.Buffer
	if err := run([]string{"route", "-feed", "x"}, &out); err == nil {
		t.Fatal("expected error when route flags are missing")
	}
}

func TestParseClockTime(t *testing.T) {
	cases := []struct {
		in   string
		want int
		ok   bool
	}{
		{"08:00", 8 * 3600, true},
		{"08:00:30", 8*3600 + 30, true},
		{"23:59:59", 23*3600 + 59*60 + 59, true},
		{"bad", 0, false},
		{"25:99", 0, false},
		{"08:60:00", 0, false},
	}
	for _, c := range cases {
		got, err := parseClockTime(c.in)
		if c.ok && err != nil {
			t.Errorf("parseClockTime(%q) unexpected error: %v", c.in, err)
			continue
		}
		if !c.ok && err == nil {
			t.Errorf("parseClockTime(%q) expected error, got %d", c.in, got)
			continue
		}
		if c.ok && got != c.want {
			t.Errorf("parseClockTime(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}
