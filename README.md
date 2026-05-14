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

## Sample data

A tiny synthetic GTFS feed ships under [`assets/sample_gtfs/`](assets/sample_gtfs/)
for local development and testing. It contains ten stops around Bern central
station, three routes (two trams and a bus), three trips per route running
between 07:00 and 08:22 on weekdays, and a handful of walking transfers.

Use it from the Go side via:

```go
feed, err := router.LoadGTFS("assets/sample_gtfs")
```

See [`assets/sample_gtfs/README.md`](assets/sample_gtfs/README.md) for the
feed layout, and [`docs/architecture.md`](docs/architecture.md) for how the
data, the Go routing core, and the Flutter UI fit together.
