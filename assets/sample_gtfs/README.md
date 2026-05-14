# Sample GTFS feed

A tiny synthetic GTFS feed used as a developer fixture for the transit-planner
prototype. It is **not** a real network: stop names and coordinates are taken
from real locations in Bern, Switzerland, but routes, trips, and timings are
fabricated.

## Goals

- Small enough to read by eye while debugging the Go router or Dart UI.
- Realistic-looking coordinates so MapLibre renders something recognisable.
- Exercises every code path in `router/gtfs.go`:
  - multi-stop trips,
  - shared stops between routes,
  - inter-route transfers via `transfers.txt`,
  - weekday-only `calendar.txt` with one `calendar_dates.txt` exception.

## Contents

| File | Rows | Notes |
| --- | --- | --- |
| `agency.txt` | 1 | Single sample agency, `Europe/Zurich`. |
| `stops.txt` | 10 | Around Bern central station. |
| `routes.txt` | 3 | Two trams (`route_type=0`) and one bus (`route_type=3`). |
| `trips.txt` | 9 | 3 trips per route. |
| `stop_times.txt` | 45 | 07:00–08:22 service window. |
| `calendar.txt` | 1 | `WEEKDAY` service, Mon–Fri, 2026. |
| `calendar_dates.txt` | 1 | Removes service on 2026-01-01 (New Year). |
| `transfers.txt` | 4 | Two bi-directional walking transfers. |

## Stops

| ID | Name | Lat | Lon |
| --- | --- | --- | --- |
| S01 | Bern Bahnhof | 46.948900 | 7.439900 |
| S02 | Bern Baerenplatz | 46.947700 | 7.444100 |
| S03 | Bern Zytglogge | 46.947900 | 7.447600 |
| S04 | Bern Casinoplatz | 46.946800 | 7.447700 |
| S05 | Bern Helvetiaplatz | 46.944400 | 7.449600 |
| S06 | Bern Bundesplatz | 46.946600 | 7.444200 |
| S07 | Bern Hirschengraben | 46.946900 | 7.439100 |
| S08 | Bern Lorraine | 46.952100 | 7.442100 |
| S09 | Bern Breitenrain | 46.955800 | 7.448000 |
| S10 | Bern Wankdorf | 46.963300 | 7.464000 |

## Routes and patterns

- **Tram 6** (`R_TRAM6`): `S08 -> S01 -> S02 -> S03 -> S05`
- **Tram 9** (`R_TRAM9`): `S07 -> S01 -> S06 -> S03 -> S04 -> S05`
- **Bus 10** (`R_BUS10`): `S01 -> S08 -> S09 -> S10`

Each route runs three trips, roughly every 30 minutes between 07:00 and 08:22.
Tram 6 and Tram 9 cross at `S01` (Bahnhof) and `S03` (Zytglogge), so the feed
exposes a meaningful transfer between the two even without walking transfers.

## Transfers

`transfers.txt` declares walking transfers between physically adjacent stops:

- `S03 <-> S04` (Zytglogge <-> Casinoplatz), 90 s.
- `S01 <-> S07` (Bahnhof <-> Hirschengraben), 180 s.

`transfer_type=2` means "minimum time required", which matches what the Go
parser stores in `Transfer.Duration`.

## How to use

From the repository root:

```sh
go test ./router/...
```

To use the feed at runtime, point the loader at this directory:

```go
feed, err := router.LoadGTFS("assets/sample_gtfs")
```

The feed is also a good target for the Dart side when wiring up FFI: the small
size means responses are easy to inspect end-to-end.

## Caveats

- Coordinates are approximate and not snapped to OSM stop nodes.
- There is no `shapes.txt`, `frequencies.txt`, or fares data; add them only if
  you also extend the Go parser.
- The service window is intentionally narrow. If you need to test late-night
  trips or service that wraps past 24:00:00, extend `stop_times.txt` rather
  than treating this fixture as authoritative.
