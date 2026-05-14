# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project shape

Two-language codebase: a Flutter Material 3 app under `lib/` and a Go GTFS routing core under `router/`. The two are bridged by a JSON-over-FFI surface (`router/cffi` → `cmd/libtransitplanner` → `dart:ffi` in `lib/src/go_ffi_router*.dart`). Architecture goal is **local-first**: the phone runs the router against bundled or downloaded GTFS, with no backend service.

Authoritative architecture doc: `docs/architecture.md`. If it disagrees with the code, the code wins — but update the doc in the same change.

## Common commands

Inside `devenv shell` (or `nix run nixpkgs#devenv -- shell`):

```sh
flutter pub get
flutter analyze --no-fatal-infos --no-fatal-warnings
flutter test
go test ./...
```

Run a single Go test:

```sh
go test ./router -run TestRouterEarliestArrival -v
```

Exercise the Go router CLI against a feed or whole country:

```sh
go run ./cmd/router info  -country jp
go run ./cmd/router stops -feed assets/sample_toei_train/Toei-Train-GTFS.zip
go run ./cmd/router route -country jp -from toei:A07 -to kobe:0001 -depart 08:00
```

Fetch real GTFS feeds (lands under `assets/real_gtfs/<country>/<feed>/`, gitignored):

```sh
go run ./tool/fetch_gtfs -list
go run ./tool/fetch_gtfs -country JP
go run ./tool/fetch_gtfs -feed toei-train
```

Build the FFI shared library (host):

```sh
CGO_ENABLED=1 go build -buildmode=c-shared \
    -o build/libtransit_planner.so ./cmd/libtransitplanner
```

App icons are regenerated from `assets/icon/app_icon.svg` (the platform PNGs are gitignored). CI runs this before any Flutter build:

```sh
python3 tool/generate_app_icon.py   # needs rsvg-convert + Pillow
```

## Architecture

### Layers

```
Flutter UI (lib/)
   └── LocalTransitRouter interface (lib/src/local_router.dart)
        ├── MockTransitRouter   — pure-Dart fake, used by tests/UI dev
        └── GoFfiRouter         — dart:ffi → libtransit_planner
                                   ├── go_ffi_router_io.dart   (mobile/desktop)
                                   └── go_ffi_router_stub.dart (web fallback)
                                          │
                                          ▼
                              cmd/libtransitplanner (cgo wrapper, TP_Route/TP_Free)
                                          │
                                          ▼
                              router/cffi (pure-Go JSON entrypoint, no cgo)
                                          │
                                          ▼
                              router/ (GTFS parser + RAPTOR engine + index/snap/merge)
```

The UI only ever sees Dart value objects from `lib/src/models.dart` (`TransitStop`, `Itinerary`, `ItineraryLeg`, `RouteRequest`, `TransitMode`). Anything touching GTFS or the routing engine must stay behind `LocalTransitRouter`.

### Go router (`router/`)

- `gtfs.go` — CSV/zip parser for `stops`, `routes`, `trips`, `stop_times`, `transfers`. Permissive (tolerates UTF-8 BOM, extra columns, blank arrival/departure passing-only rows). Times are seconds-from-midnight ints.
- `calendar.go` — `calendar.txt` / `calendar_dates.txt` service-day logic.
- `router.go` — single-criterion round-based RAPTOR, bounded by `MaxTransfers + 1` rounds.
- `mcraptor.go` — multi-criteria variant.
- `range.go` — Range RAPTOR (departure-window queries).
- `snap.go` — haversine stop snapping for free-form origins/destinations.
- `merge.go` — namespaces IDs as `<prefix>:<id>` so multiple feeds become one routable network; stitches same-named stops via cross-feed transfers.
- `index/` — compact in-memory indexes.
- `cffi/` — pure-Go JSON request/response (`RouteJSON`); the cgo wrapper in `cmd/libtransitplanner` just re-exports `TP_Route` / `TP_Free`.

The Go module is stdlib-only — keep it that way unless a dependency is unavoidable. Single-call lifecycle: feed reload happens per call today (will gain caching once the Flutter side's lifecycle is settled).

### GTFS data

- `assets/sample_gtfs/` — tiny synthetic Bern fixture for deterministic unit tests in `router/router_test.go`. Hand-written expected itineraries; do not extend without updating the test.
- `assets/sample_toei_train/` — vendored real Toei subway feed used by the FFI integration test and as the bundled CI feed.
- `assets/real_gtfs/<country>/<feed>/` — fetched feeds, gitignored. Each ships with a `MANIFEST.json` (source URL, fetch time, SHA-256).
- Country/feed catalog: `lib/src/feed_catalog.dart` (Dart side) and the `fetch_gtfs` tool (Go side); IDs are ISO 3166-1 alpha-2 country codes.

Only no-API-key feeds are in scope. Major private rail (JR, Tokyo Metro, Hankyu, etc.) requires authenticated ODPT access and is intentionally excluded.

## Conventions specific to this repo

- Keep Flutter UI under `lib/`, tests under `test/`. Keep Go under `router/` (library) and `cmd/` (binaries).
- Do not introduce server-side routing assumptions. Local-first is a hard requirement.
- Avoid new dependencies unless they materially simplify MapLibre, GTFS parsing, FFI, or tests.
- Any new field on `Itinerary` / `ItineraryLeg` must round-trip through the JSON FFI codec (`router/cffi/cffi.go` ↔ `lib/src/go_ffi_router*.dart`). Update both sides in the same change.
- When extending GTFS support in `router/gtfs.go`, check whether the synthetic Bern fixture exercises the new path; if not, add coverage rather than relying solely on the Toei feed.
- CI builds the `c-shared` library and packages it into `android/app/src/main/jniLibs/`. If FFI symbol names change, update both the `nm` check in `.github/workflows/ci.yml` and the Dart `lookup` calls.

## CI

`.github/workflows/ci.yml` runs four jobs: `flutter` (analyze+test), `go` (test + host c-shared build), `build-go-android` (NDK cross-compile for arm64-v8a and x86_64 → jniLibs artifact), and `build-android` (consumes jniLibs, builds and signs a release APK, publishes a prerelease tag on `main`/`master`/`develop`/tags). Pushes to those branches always cut a prerelease GitHub Release.
