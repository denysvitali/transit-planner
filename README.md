# Transit Planner

Local-first transit planner prototype using Flutter, MapLibre vector maps, GTFS,
and a Go routing core.

## What is implemented

- Flutter Material 3 app shell with a MapLibre map, trip inputs, mode filters,
  transfer limit, and itinerary cards.
- Local router abstraction in Dart with a mock implementation for UI work.
- Go GTFS parser and RAPTOR-style earliest-arrival routing engine under
  `router/`.
- GitHub Actions CI for Flutter analysis/tests and Go tests.
- `devenv` configuration for reproducible local tooling.

## Routing approach

The Go core uses a RAPTOR-style round-based search:

1. Load local GTFS static files.
2. Index trips by served stop.
3. Start from the origin stop at the requested departure time.
4. For each transfer round, board trips that can be caught from currently
   reached stops.
5. Propagate arrivals to downstream stops.
6. Apply local walking transfers from `transfers.txt`.

This is intentionally small, but it keeps the right shape for a future
McRAPTOR implementation with multiple criteria such as arrival time, transfers,
walking time, accessibility, and fares.

## Development

Preferred:

```sh
devenv shell
flutter pub get
flutter analyze --no-fatal-infos --no-fatal-warnings
flutter test
go test ./...
```

If `devenv` is not installed:

```sh
nix run nixpkgs#devenv -- shell
```

Or run Flutter directly through Nix:

```sh
nix run nixpkgs#flutter -- pub get
nix run nixpkgs#flutter -- analyze --no-fatal-infos --no-fatal-warnings
nix run nixpkgs#flutter -- test
```

## Next steps

- Replace the Dart mock router with FFI or platform-channel calls into Go.
- Compile GTFS into compact binary indexes for fast mobile startup.
- Add local stop search and origin/destination snapping.
- Add OSM pedestrian routing for realistic first/last-mile walking.
- Extend the Go router to McRAPTOR and return Pareto-optimal alternatives.

## Transit data

The router is wired against **real, no-API-key GTFS feeds** from Japan's
Public Transportation Open Data Center (ODPT). The first integration covers
the Tokyo Metropolitan Bureau of Transportation (都営):

- **Toei subway lines** (浅草, 三田, 新宿, 大江戸, 日暮里舎人, 都電荒川) —
  ~150 stations, ~5 600 trips. Vendored as the CI fixture under
  [`assets/sample_toei_train/`](assets/sample_toei_train/).
- **Toei municipal bus network** — ~5 400 stops, ~47 000 trips. Downloaded
  on demand into `assets/real_gtfs/` (gitignored) via the fetcher tool.

Both feeds are CC-BY 4.0 — see
[`LICENSES_THIRD_PARTY.md`](LICENSES_THIRD_PARTY.md) for the required
attribution string.

```sh
go run ./tool/fetch_gtfs -list                  # show known feeds
go run ./tool/fetch_gtfs -feed toei-bus         # ~6 MB zip
go run ./tool/fetch_gtfs -feed toei-train       # ~750 KB zip
```

Load a feed from Go:

```go
feed, err := router.LoadGTFSZip("assets/sample_toei_train/Toei-Train-GTFS.zip")
// or, for an extracted directory of CSVs:
feed, err = router.LoadGTFS("assets/real_gtfs/toei_bus")
```

### Synthetic test fixture

A tiny synthetic feed also ships under
[`assets/sample_gtfs/`](assets/sample_gtfs/) (ten Bern stops, three routes,
weekday-only service). It exists only as a small, deterministic unit-test
fixture for [`router/router_test.go`](router/router_test.go); production
code paths use real GTFS-JP via `LoadGTFSZip`.

See [`docs/architecture.md`](docs/architecture.md) for how the data, the Go
routing core, and the Flutter UI fit together.
