# router/cffi

Pure-Go core of the C-ABI / `dart:ffi` surface that the Flutter app uses to
drive the transit-planner router. The package exposes a single entrypoint:

- `RouteJSON(req string) string` — accepts a JSON request, returns a JSON
  response. All errors are encoded as `{"error": "..."}`; the function never
  panics.

The package is **pure Go** — no `import "C"`, no cgo build tag — so it is
exercised by the default `go test ./...` flow on CI without needing a C
toolchain. The thin cgo wrapper that produces the actual shared library
lives in [`../../cmd/libtransitplanner`](../../cmd/libtransitplanner).

## JSON contract

Request — set exactly one of `feedDir` or `feedZip`:

```json
{
  "feedZip": "/path/to/Toei-Train-GTFS.zip",
  "from": "101",
  "to": "108",
  "departure": 18000,
  "maxTransfers": 0
}
```

```json
{
  "feedDir": "/path/to/extracted/gtfs",
  "from": "A",
  "to": "B",
  "departure": 28800,
  "maxTransfers": 2
}
```

Successful response:

```json
{
  "arrival": 32400,
  "transfers": 1,
  "legs": [
    {
      "mode": "transit",
      "routeId": "R1",
      "tripId": "T1",
      "fromStop": { "ID": "A", "Name": "Alpha", "Lat": 46.0, "Lon": 7.0 },
      "toStop":   { "ID": "B", "Name": "Beta",  "Lat": 46.1, "Lon": 7.1 },
      "departure": 28800,
      "arrival": 29400
    }
  ]
}
```

Failure response:

```json
{ "error": "destination unreachable" }
```

The current implementation reloads the GTFS feed on every call. A future
revision will introduce caching once we know how Flutter wants to manage the
feed lifecycle.

## Building the shared library

The cgo wrapper in [`cmd/libtransitplanner`](../../cmd/libtransitplanner)
re-exports `TP_Route` and `TP_Free` as C symbols. Build it for the host:

```sh
# Linux
CGO_ENABLED=1 go build \
  -buildmode=c-shared \
  -o build/libtransit_planner.so \
  ./cmd/libtransitplanner

# macOS
CGO_ENABLED=1 go build \
  -buildmode=c-shared \
  -o build/libtransit_planner.dylib \
  ./cmd/libtransitplanner

# Windows (from a MinGW shell)
CGO_ENABLED=1 go build \
  -buildmode=c-shared \
  -o build/transit_planner.dll \
  ./cmd/libtransitplanner
```

The build also produces a `libtransit_planner.h` header next to the library;
the Flutter side can use it as a reference when wiring `dart:ffi` bindings.

For Android / iOS, run the same command under the NDK / Xcode toolchains
with the appropriate `CC`, `CGO_ENABLED=1`, and per-ABI `GOOS` / `GOARCH`.

## Loading from Flutter

On the Dart side use `DynamicLibrary.open` with the platform-specific path,
look up the `TP_Route` and `TP_Free` symbols, and remember to call `TP_Free`
for every pointer returned by `TP_Route` to avoid leaking memory.

## Running the tests

```sh
go test ./router/cffi          # the routing surface (no cgo needed)
go build ./cmd/libtransitplanner  # smoke test that the wrapper still compiles
```

The integration test `TestRouteJSONFeedZipToei` loads the vendored Toei
subway feed from `assets/sample_toei_train/Toei-Train-GTFS.zip` and routes
through it via the JSON surface — i.e. the exact code path the Flutter FFI
will eventually exercise.
