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

The app opens the bundled default feed on startup so launch never downloads a
multi-country catalog before the user asks for one. The Settings network
selector can then load the Transitland coverage collection or smaller country and
regional collections through the same local GTFS router path. Transitland
discovery runs in tooling/CI; the API key is never embedded in the Flutter app.

Sources are open and license-tagged:

- **[Mobility Database](https://mobilitydatabase.org)** — the canonical
  global catalog (6000+ GTFS feeds across 99+ countries; their CSV at
  `https://files.mobilitydatabase.org/feeds_v2.csv` is what we cross-check
  against). The fetcher can also use this catalog directly with `-complete`
  to build fuller no-key country bundles for fragmented countries such as
  Japan and Italy.
- **[Transitland REST API](https://www.transit.land/documentation/rest-api/feeds)** —
  authenticated feed discovery and latest-version downloads. Use
  `TRANSITLAND_API_KEY` with `-complete-source transitland`; the key is sent in
  the `apikey` header and is not written to manifests.
- **[ODPT public bucket](https://www.odpt.org)** — `api-public.odpt.org`
  hosts the Tokyo Metropolitan Bureau of Transportation feeds (CC-BY 4.0).
- **[opentransportdata.swiss](https://opentransportdata.swiss)** — official
  nationwide Swiss static GTFS for the current timetable year.
- **Italian regional and city portals** — official no-key GTFS from Rome,
  Milan, Lombardy/Trenord, Tuscany, and Trentino.

Currently catalogued (no API key required):

| Country | Region | Feeds |
|---------|--------|-------|
| CH | Nationwide | `ch-aggregate-2026` |
| IT | Lazio | `it-rome` |
| IT | Lombardy | `it-milan-atm`, `it-lombardy-trenord` |
| IT | Tuscany | `it-tuscany-autolinee`, `it-tuscany-trenitalia`, `it-tuscany-tft`, `it-tuscany-toremar`, `it-tuscany-gest`, `it-tuscany-colbus-school`, `it-tuscany-colbus-nonschool`, `it-tuscany-at-school`, `it-tuscany-at-nonschool` |
| IT | Trentino-Alto Adige | `it-trentino-urban`, `it-trentino-extraurban` |
| JP | Tokyo | `toei-bus`, `toei-train` |
| JP | Hyogo | `kobe-shiokaze`, `kobe-satoyama`, `himeji-ieshima`, `takarazuka-runrunbus`, `nishinomiya-sakurayamanami` |
| JP | Nara | `yamatokoriyama-kingyobus` |
| JP | Wakayama | `rinkan-koyasan` |
| JP | Ishikawa | `kanazawa-flatbus`, `kanazawa-hakusan-meguru`, `kanazawa-tsubata-bus` |

For fuller country builds, use Transitland discovery where the API key is
available. Mobility Database remains available as a no-key fallback and
cross-checking source:

```sh
TRANSITLAND_API_KEY=... go run ./tool/fetch_gtfs -country JP -complete -complete-source transitland
TRANSITLAND_API_KEY=... go run ./tool/fetch_gtfs -country IT -complete -complete-source transitland
TRANSITLAND_API_KEY=... go run ./tool/fetch_gtfs -country CH -complete -complete-source transitland
go run ./tool/fetch_gtfs -country JP -complete
```

Japan still has important rail gaps: major private rail (JR, Tokyo Metro,
Hankyu, Hanshin, Nankai, Keihan, Kintetsu) and the Shinkansen may require
ODPT registration or operator-specific terms, so they are intentionally absent
from the no-key app catalog unless a redistributable public ZIP exists.

Licences vary per feed — see [`LICENSES_THIRD_PARTY.md`](LICENSES_THIRD_PARTY.md)
and each downloaded `MANIFEST.json` for the exact attribution string.

```sh
go run ./tool/fetch_gtfs -list                  # show every known feed
go run ./tool/fetch_gtfs -list -country JP      # filter to one country
go run ./tool/fetch_gtfs -feed toei-train       # ~750 KB zip
go run ./tool/fetch_gtfs -country CH            # fetch the Swiss national feed
go run ./tool/fetch_gtfs -country IT            # fetch curated Italian feeds
go run ./tool/fetch_gtfs -country JP -complete  # fetch active no-key JP feeds from Mobility Database
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
  -feed toei-train=assets/sample_toei_train/Toei-Train-GTFS.zip \
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
