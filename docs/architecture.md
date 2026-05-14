# Architecture overview

This document describes the local-first architecture of `transit-planner`. It
is aimed at contributors who want to understand how the Flutter UI, the Go
routing core, and the GTFS-derived data fit together, and where the project is
headed. Treat it as a living map of the codebase, not as a frozen spec; if
something in this file contradicts the code, the code wins and the doc should
be updated.

## Design goals

1. **Local-first.** The phone is the primary execution environment. The app
   should plan trips with no network, given a pre-bundled or downloaded GTFS
   feed.
2. **Predictable latency.** All routing queries run on-device against indexes
   built once at ingest time. We treat 100 ms as the soft budget for an
   interactive plan on a mid-range phone.
3. **Small, audited surface.** A small Go core that we fully control,
   integrated with Flutter via well-defined boundaries. No hidden runtime
   services.
4. **Replaceable parts.** Each layer (storage, routing, map, UI) should be
   swappable without rewriting the others. The Dart router today is a mock; it
   will become an FFI shim without touching the UI.

## Component map

```
+--------------------------------------------------------------+
|                          Flutter app                         |
|  lib/main.dart, lib/src/home_page.dart, settings_page.dart   |
|  lib/src/models.dart, lib/src/local_router.dart (interface)  |
+----------------------------+---------------------------------+
                             |
                             |  Future<List<Itinerary>> route(...)
                             v
+--------------------------------------------------------------+
|              LocalTransitRouter implementations              |
|  - MockTransitRouter (current, in-Dart)                      |
|  - GoFfiRouter      (planned, cgo FFI to router/)            |
+----------------------------+---------------------------------+
                             |
                             |  cgo / FFI boundary (planned)
                             v
+--------------------------------------------------------------+
|                       Go routing core                        |
|  router/gtfs.go     - GTFS parser                            |
|  router/router.go   - RAPTOR-style engine                    |
|  router/router_test.go - regression tests                    |
+----------------------------+---------------------------------+
                             |
                             |  reads files / mmap'd indexes
                             v
+--------------------------------------------------------------+
|                       On-device data                         |
|  assets/sample_toei_train/ - real Toei feed (149 stops,      |
|                              6 lines, 5.6k trips) -- CI/dev  |
|  assets/sample_gtfs/       - synthetic Bern unit-test feed   |
|  assets/real_gtfs/         - fetched feeds (gitignored)      |
|  app documents dir         - downloaded GTFS + built indexes |
+--------------------------------------------------------------+
```

Files referenced above:

- Flutter shell: `lib/main.dart`, `lib/src/home_page.dart`,
  `lib/src/settings_page.dart`, `lib/src/theme.dart`, `lib/src/app_log.dart`.
- Dart domain types: `lib/src/models.dart` (`TransitStop`, `Itinerary`,
  `ItineraryLeg`, `RouteRequest`, `TransitMode`).
- Router interface: `lib/src/local_router.dart`.
- Go core: `router/gtfs.go`, `router/router.go`.
- Real GTFS fixture: `assets/sample_toei_train/` (Tokyo Toei subway).
- Synthetic unit-test fixture: `assets/sample_gtfs/` (Bern).
- Fetcher tool: `tool/fetch_gtfs/`.
- Attribution: `LICENSES_THIRD_PARTY.md`.

## Flutter app

The app is a small Material 3 shell built around a MapLibre map. The home page
hosts the input form (origin, destination, departure time, mode filters,
maximum transfers) and a list of `Itinerary` cards. The map is rendered with
`maplibre_gl` and styled with a vector style URL configurable from the
settings view.

Key types in `lib/src/models.dart`:

- `TransitStop` carries `id`, `name`, `latitude`, `longitude`.
- `ItineraryLeg` represents one continuous movement (walk or transit) with
  `mode`, `from`, `to`, `departure`, `arrival`, optional `routeName` and
  `tripId`.
- `Itinerary` aggregates legs, total `transfers`, and total `walking`.
- `RouteRequest` is what the UI hands to `LocalTransitRouter.route`.

The UI never talks to GTFS or the Go engine directly. It only sees Dart value
objects, which keeps the FFI boundary narrow.

## MapLibre rendering

MapLibre handles all rendering. We rely on it for:

- Vector tile rendering of OSM data (via a configurable style URL).
- Marker placement for origin, destination, transfer points, and itinerary
  geometry.
- Camera control and basic gesture handling.

We deliberately avoid stop-level overlays for now. Once the Go side returns
geometry instead of just stop pairs, we will draw itineraries as GeoJSON
sources rather than ad hoc markers.

## Go routing core

`router/` is a tiny self-contained Go module with no external dependencies
beyond the standard library. It owns three concerns: GTFS parsing, in-memory
indexing, and the route search itself.

### GTFS ingest (`router/gtfs.go`)

The parser has two entry points, both backed by an `io/fs.FS` view of the
feed so the file readers don't care about the source:

- `LoadGTFS(dir)` reads CSV files from a directory.
- `LoadGTFSZip(path)` reads them directly out of a zip archive — this is the
  shape the real ODPT feeds ship in, and the path that production code
  (mobile, eventually FFI) should use.

It reads the minimum subset needed for routing today:

- `stops.txt` (`stop_id`, `stop_name`, `stop_lat`, `stop_lon`).
- `routes.txt` (`route_id`, `route_short_name`, `route_long_name`,
  `route_type`).
- `trips.txt` (`route_id`, `service_id`, `trip_id`).
- `stop_times.txt` (`trip_id`, `arrival_time`, `departure_time`, `stop_id`,
  `stop_sequence`). Rows where both `arrival_time` and `departure_time` are
  blank are skipped: in GTFS-JP feeds these mark `timepoint=0` passing-only
  stops (Toei Train has ~540 of them) and the router boards/alights only at
  timed stops anyway.
- `transfers.txt` (`from_stop_id`, `to_stop_id`, optional
  `min_transfer_time`); optional file.

Times are stored as seconds-from-midnight ints. The parser is permissive about
extra columns and tolerates a UTF-8 BOM on the first column header. It does
not currently understand `calendar.txt`, `calendar_dates.txt`,
`frequencies.txt`, `shapes.txt`, or the GTFS-JP-specific files
(`translations.txt`, `agency_jp.txt`, `office_jp.txt`).

### Routing (`router/router.go`)

The engine is a single-criterion, round-based RAPTOR variant that minimises
arrival time. For each round k:

1. Apply pending walking transfers from `transfers.txt` to all stops marked in
   round k.
2. For every reached stop, scan trips serving that stop, find the earliest
   boardable trip given the current arrival time, and propagate arrivals to
   every downstream stop.
3. The reached set for round k+1 is the union of new arrivals.

The number of rounds is bounded by `Options.MaxTransfers + 1`. After the final
round, transfers are applied once more, and the best label over all rounds at
the destination stop is reconstructed into an `Itinerary`.

This is intentionally close to a textbook RAPTOR, with two simplifications:

- Routes are not bucketed; we iterate trips per stop. For small feeds this is
  fine. For larger feeds we will switch to the standard route-pattern
  representation.
- Transfers are only those declared in `transfers.txt`. There is no implicit
  walking graph yet.

## Local data pipeline

The end-to-end flow we are targeting:

```
GTFS zip --> parse --> validate --> index --> persist --> load --> route
```

Stages:

1. **Acquire.** A GTFS bundle either ships with the app (for the sample feed)
   or is downloaded from a configured source URL into the app's documents
   directory.
2. **Parse.** `router.LoadGTFS` reads CSV files. The current implementation is
   line-by-line and allocation-heavy; we accept this for now because the
   sample feed is tiny.
3. **Validate.** Out of scope today. Future work: enforce referential
   integrity (every `trip.route_id` exists, every `stop_times.stop_id`
   exists), check for monotonic stop sequences, flag impossible times.
4. **Index.** Today everything lives in maps in memory. The next step is to
   precompute and persist:
   - Stop-id -> compact integer index.
   - Route-pattern table (one entry per distinct stop sequence).
   - Trips sorted by route pattern and departure, for log-time boarding
     lookup.
   - Transfer adjacency in CSR (compressed sparse row) form.
5. **Persist.** SQLite is the obvious first target because Flutter already has
   good bindings. A binary mmap-friendly format (think FlatBuffers or a custom
   layout) is the long-term goal for cold-start latency.
6. **Load.** The Go engine takes either a directory of CSVs or, eventually,
   an opened index file. The Dart side never touches either; it only sees the
   resulting itineraries.

## Routing flow

A user trip plan involves four conceptual steps. The current code only
implements the middle two; the rest are stubbed or handled by the UI.

1. **Snap.** Given a free-form origin/destination (lat/lon or text), find the
   nearest set of candidate stops. Today the UI passes stop IDs directly,
   which is fine for the mock but not for real users.
2. **Access.** Compute walking time from the origin to each candidate stop.
   MVP: haversine distance times a configurable speed (default 1.3 m/s).
   Eventually: shortest-path on an OSM pedestrian graph.
3. **Rounds.** Run the round-based search from the access set. Each candidate
   stop is seeded into round 0 with its access time added to the requested
   departure.
4. **Egress.** Symmetric to access: walking time from each reachable stop to
   the destination. The best total arrival time across all egress candidates
   wins.

The current engine collapses steps 1, 2, and 4 by assuming a single origin
stop and a single destination stop. The shape of the algorithm already
supports multiple seeded stops, which is what we will use when access/egress
are real.

## Walking strategies

We treat walking as a separable concern.

- **MVP (today).** Haversine distance plus a constant speed. Cheap, no extra
  data, good enough for short transfers in a city centre. Transfers between
  named stops still come from `transfers.txt` rather than being recomputed.
- **Phase 2.** Bundle a pedestrian graph derived from OSM ways tagged
  `highway=footway`, `highway=path`, sidewalks, and crossings. Run Dijkstra
  with a contraction hierarchy or A* with a great-circle heuristic. Cache
  stop-to-stop walks under a configurable radius (300 m? 800 m?).
- **Phase 3.** Time-of-day-aware walking: avoid closed pedestrian zones,
  prefer lit paths at night, optionally honour accessibility constraints
  (stairs, kerbs).

Walking is also where the McRAPTOR transition gets interesting: walking time
becomes a second optimisation criterion alongside arrival time, so itineraries
that arrive a minute later but walk five minutes less can survive into the
Pareto set.

## Go <-> Flutter integration

The Dart side defines `LocalTransitRouter` in `lib/src/local_router.dart`. The
only implementation today is `MockTransitRouter`, which fabricates two
itineraries for any request. The plan:

1. **cgo FFI.** Build `router/` as a C-archive (`go build -buildmode=c-archive`
   for iOS, `-buildmode=c-shared` for Android), expose a small C ABI, and call
   it from Dart with `dart:ffi`.
2. **JSON over FFI (now).** The first wire format is JSON. It is slow but
   trivial to debug and matches the existing Dart models. Request and response
   are both null-terminated UTF-8 strings allocated by Go and freed via an
   explicit `Free` symbol.
3. **Schema'd binary later.** Once the API stabilises, switch to FlatBuffers
   or protobuf. The same Dart models stay in place; only the codec changes.
4. **No goroutines across the boundary.** The exported entrypoint runs a
   single search and returns. Concurrency lives entirely on the Dart side.

We also need to decide where the engine lives at process scope. Reloading the
GTFS feed on every call is fine for the sample feed (a few hundred kilobytes)
but unacceptable for real ones. The plan is to expose
`router_init(handle, gtfs_dir)` and `router_route(handle, request_json)` so
the Go state outlives a single call.

## Risks and open questions

- **GTFS edge cases.** Times past `24:00:00`, blocks, frequencies, and
  service exceptions are not modelled yet. The sample feed deliberately
  avoids these so they fail loudly when we add them.
- **Memory pressure on phones.** A full national feed can be hundreds of
  megabytes uncompressed. We will need compact indexes and probably a
  region-limited deployment per device.
- **FFI startup cost.** Loading the Go runtime adds noticeable bytes and
  warm-up time. Worth measuring before committing.
- **Cold start.** Reading a multi-megabyte SQLite database before the first
  route query is the obvious latency sink. Mmap'd binary indexes are the
  escape hatch.
- **Battery.** Routing is bursty but doing it on a hot UI thread will be felt.
  All FFI calls must be on a Dart isolate or scheduled off the platform
  thread.
- **Testing real feeds.** The synthetic Bern fixture catches API regressions
  but not scale regressions. The vendored Toei Train feed
  (`assets/sample_toei_train/`) closes part of that gap — it is real Tokyo
  data with ~150 stops and ~5 600 trips — but full-network scale (Toei Bus
  is ~47 000 trips) is still only validated by running `tool/fetch_gtfs`
  manually.

## Staged build plan

The plan is intentionally staged so each milestone is shippable on its own.

### Milestone 1 - Wired mock (done)

- Flutter UI with map, inputs, itinerary cards.
- `MockTransitRouter` returns plausible-looking results.
- Go RAPTOR core exists and is tested in isolation against synthetic data.
- CI runs Flutter analyse/test and Go test.

### Milestone 2 - Real GTFS feed (in progress)

- Ship `assets/sample_toei_train/` — a vendored real Tokyo Toei subway feed
  from ODPT — as the CI fixture.
- Add `tool/fetch_gtfs/` to pull live Toei Bus / Toei Train feeds from the
  no-API-key public ODPT bucket into the gitignored `assets/real_gtfs/`.
- Keep `assets/sample_gtfs/` as the small synthetic unit-test fixture (its
  expected itineraries are hand-written and small enough to reason about).
- Add architecture documentation (this file) and third-party attribution
  (`LICENSES_THIRD_PARTY.md`).

### Milestone 3 - Real router from Flutter

- Build the Go core as a static/dynamic library.
- Implement `GoFfiRouter` against `LocalTransitRouter`.
- Replace `MockTransitRouter` in production code, keep it for tests.
- Use stop IDs from the bundled feed in the UI input layer.

### Milestone 4 - Indexes and persistence

- Define a binary index format.
- Add a one-shot ingest tool that converts GTFS into the binary format.
- Switch the Go engine to read indexes instead of raw CSV at runtime.
- Persist indexes under the platform-appropriate app directory.

### Milestone 5 - Real access/egress

- Implement haversine-based stop snapping.
- Add origin/destination geocoding (probably platform-side initially).
- Extend the engine API to accept a set of seeded stops with offsets.

### Milestone 6 - OSM pedestrian routing

- Bundle (or download) an OSM extract for the active region.
- Build a contracted pedestrian graph.
- Replace haversine access/egress with real walking shortest paths.

### Milestone 7 - McRAPTOR

- Add walking time as a second criterion.
- Return Pareto-optimal alternatives.
- Surface "fastest" vs "fewest transfers" vs "least walking" in the UI.

### Milestone 8 - Range RAPTOR

- Support "show me everything that leaves in the next hour" without N
  independent searches.
- Useful for showing connection bands rather than single trips.

## Where to look next

- Start a new feature by reading `router/router_test.go` to understand the
  contract.
- Treat `assets/sample_toei_train/README.md` as the canonical description of
  the real fixture and `assets/sample_gtfs/README.md` for the synthetic one;
  if you extend either feed, update its README in the same change.
- Keep `lib/src/models.dart` and `router/router.go` in lockstep: any new field
  on `Itinerary` or `Leg` must round-trip through the (eventual) FFI codec.
