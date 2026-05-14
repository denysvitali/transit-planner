# router/cffi

C-ABI surface around the transit-planner router. The package is built with
cgo and exposes two `//export`-annotated functions so that the Flutter app
can drive the Go router through `dart:ffi`:

- `TP_Route(reqJSON *C.char) *C.char` — accepts a JSON request and returns a
  JSON response. The returned pointer is allocated by cgo and **must** be
  freed by the caller via `TP_Free`.
- `TP_Free(p *C.char)` — releases a string previously returned by `TP_Route`.

The package is gated behind the `cgo` build tag. When cgo is disabled
(`CGO_ENABLED=0`) a stub file keeps the package importable but exports
nothing, so `go build ./...` and `go test ./...` keep working on CI images
that do not ship a C toolchain.

## JSON contract

Request:

```json
{
  "feedDir": "/path/to/gtfs",
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

`go build -buildmode=c-shared` requires a `package main`. The cffi package
itself is intentionally a normal library package so it stays importable from
ordinary Go code (and from the test suite). To produce the actual `.so` /
`.dylib` / `.dll` consumed by Flutter, create a thin main wrapper that
re-exports the two cgo entrypoints. A minimal wrapper looks like:

```go
// build/libtransitplanner/main.go
package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"unsafe"

	"github.com/denysvitali/transit-planner/router/cffi"
)

//export TP_Route
func TP_Route(reqJSON *C.char) *C.char {
	var raw string
	if reqJSON != nil {
		raw = C.GoString(reqJSON)
	}
	return C.CString(cffi.RouteJSON(raw))
}

//export TP_Free
func TP_Free(p *C.char) {
	if p == nil {
		return
	}
	C.free(unsafe.Pointer(p))
}

func main() {}
```

Then build:

```sh
# Linux
CGO_ENABLED=1 go build \
  -buildmode=c-shared \
  -o build/libtransit_planner.so \
  ./build/libtransitplanner

# macOS
CGO_ENABLED=1 go build \
  -buildmode=c-shared \
  -o build/libtransit_planner.dylib \
  ./build/libtransitplanner

# Windows (from a MinGW shell)
CGO_ENABLED=1 go build \
  -buildmode=c-shared \
  -o build/transit_planner.dll \
  ./build/libtransitplanner
```

The build also produces a `libtransit_planner.h` header next to the library
which the Flutter side can use as a reference when wiring the `dart:ffi`
bindings.

The `//export TP_Route` and `//export TP_Free` directives also live on the
matching functions inside this package. They serve as the canonical
declaration of the FFI surface; the main wrapper above simply forwards to
them through the package's pure-Go `RouteJSON` helper to avoid duplicate C
symbols at link time.

## Loading from Flutter

On the Dart side use `DynamicLibrary.open` with the platform-specific path,
look up the `TP_Route` and `TP_Free` symbols, and remember to call `TP_Free`
for every pointer returned by `TP_Route` to avoid leaking memory.

## Running the tests

```sh
# Default CI flow — cgo disabled, the stub is compiled.
CGO_ENABLED=0 go test ./...

# Full FFI test (requires a C toolchain such as gcc or clang).
CGO_ENABLED=1 go test ./router/cffi
```
