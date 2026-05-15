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
generate-sqlc
check-go
check-flutter
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

### App icons

The source of truth is [`assets/icon/app_icon.svg`](assets/icon/app_icon.svg).
The platform-specific PNGs (Android mipmaps, iOS AppIcon set, web/PWA icons,
favicon) are git-ignored and regenerated from the SVG with:

```sh
python3 tool/generate_app_icon.py
```

The script needs `rsvg-convert` (Debian/Ubuntu: `librsvg2-bin`) and Pillow
(`python3-pil`). CI runs it automatically before `flutter build`.

## Next steps

- Replace the Dart mock router with FFI or platform-channel calls into Go.
- Compile GTFS into compact binary indexes for fast mobile startup.
- Add local stop search and origin/destination snapping.
- Add OSM pedestrian routing for realistic first/last-mile walking.
- Extend the Go router to McRAPTOR and return Pareto-optimal alternatives.

## Transit data

The app discovers GTFS feeds from the Transitland REST API at runtime and
caches the discovered catalog locally. Settings lists the loaded Transitland
feeds with country, region, and feed-level checkboxes; the checked feeds are
loaded as one merged local GTFS router network. Feed downloads use Transitland
REST feed download endpoints only. Pass a Transitland API key at build/run time
with `--dart-define=TRANSITLAND_API_KEY=...`, set `TRANSITLAND_API_KEY` in the
environment, or put it in a local uncommitted `.env`; the key is sent as an
`apikey` header and is not stored in feed URLs.

Sources are Transitland-backed and license-tagged:

- **[Transitland REST API](https://www.transit.land/documentation/rest-api/feeds)** -
  authenticated feed discovery and latest-version downloads. The app does not
  carry a manually-maintained feed inventory.

For fuller country builds, use Transitland discovery:

```sh
TRANSITLAND_API_KEY=... go run ./tool/fetch_gtfs -country JP -complete -complete-source transitland
TRANSITLAND_API_KEY=... go run ./tool/fetch_gtfs -country IT -complete -complete-source transitland
TRANSITLAND_API_KEY=... go run ./tool/fetch_gtfs -country CH -complete -complete-source transitland
```

Coverage depends on the current Transitland feed records and the feeds selected
locally in Settings.

Licences vary per feed — see [`LICENSES_THIRD_PARTY.md`](LICENSES_THIRD_PARTY.md)
and each downloaded `MANIFEST.json` for the exact attribution string.

```sh
TRANSITLAND_API_KEY=... go run ./tool/fetch_gtfs -country IT -complete -complete-source transitland
```

Downloads land under `assets/real_gtfs/<country>/<feed>/<feed>.zip` with a
`MANIFEST.json` recording source URL, timestamp, and SHA-256. The directory
is gitignored; commit only vendored fixtures under `assets/sample_*`.

### Building a unified GTFS SQLite database

Use `tool/build_gtfs_db` to pour one or more GTFS directories or ZIPs into a
single SQLite database while preserving feed attribution:

```sh
go run ./tool/build_gtfs_db -db /tmp/gtfs.sqlite \
  -feed sample-toei=assets/sample_toei_train/Toei-Train-GTFS.zip \
  -feed sample-bern=assets/sample_gtfs
```

The database keeps a `feeds` table and `feed_versions` table, stores every
GTFS `.txt` file in raw `gtfs_files` / `gtfs_rows` tables, and also populates
feed-scoped query tables for core GTFS entities such as `stops`, `routes`,
`trips`, `stop_times`, `calendar_dates`, `transfers`, and `shapes`. Every
imported table carries `feed_id` and `feed_version_id`, with indexes for the
common feed, trip, stop, and attribution lookups. Re-importing the same feed
creates a new `feed_versions` row and flips the active pointer; use the
`active_*` views for app queries that should ignore historical versions.

### Loading and merging feeds

Load a single feed:

```go
feed, err := router.LoadGTFSZip("assets/sample_toei_train/Toei-Train-GTFS.zip")
// or an extracted directory of CSVs:
feed, err = router.LoadGTFS("assets/real_gtfs/jp/toei_bus")
```

Merge several feeds into one routable network (IDs get namespaced as
`<prefix>:<id>`; same-named stops are stitched into cross-feed transfers
automatically):

```go
merged := router.Merge(map[string]*router.Feed{
    "toei": toeiTrain,
    "kobe": kobeShiokaze,
})
engine := router.NewEngine(merged)
```

The CLI accepts multiple feeds or an entire country directory:

```sh
go run ./cmd/router info -country jp
go run ./cmd/router route \
    -country jp \
    -from "toei:A07" -to "kobe:0001" \
    -depart 08:00
```

### Synthetic test fixture

A tiny synthetic feed also ships under
[`assets/sample_gtfs/`](assets/sample_gtfs/) (ten Bern stops, three routes,
weekday-only service). It exists only as a small, deterministic unit-test
fixture for [`router/router_test.go`](router/router_test.go); production
code paths use real GTFS-JP via `LoadGTFSZip`.

See [`docs/architecture.md`](docs/architecture.md) for how the data, the Go
routing core, and the Flutter UI fit together.
