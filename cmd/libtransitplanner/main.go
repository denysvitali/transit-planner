// Package main is a thin cgo wrapper around router/cffi that produces the
// dynamic library consumed by the Flutter app over dart:ffi.
//
// All routing logic lives in router/cffi.*JSON (pure Go) so it is fully
// tested by the default `go test ./...` flow without a C toolchain. This
// wrapper only marshals C strings in and out.
//
// Build for the host platform:
//
//	CGO_ENABLED=1 go build -buildmode=c-shared \
//	  -o build/libtransit_planner.so ./cmd/libtransitplanner
//
// For Android, drive the same command with the NDK toolchain wrapper as
// CC (see .github/workflows/ci.yml for the per-ABI matrix).
package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"unsafe"

	"github.com/denysvitali/transit-planner/router/cffi"
)

// TP_Open accepts a JSON request with feedZip or feedDir, loads the feed,
// and returns a JSON response containing a handle for subsequent calls.
//
//export TP_Open
func TP_Open(reqJSON *C.char) *C.char {
	return C.CString(cffi.OpenJSON(goString(reqJSON)))
}

// TP_Close releases a feed previously returned by TP_Open. Safe to call
// repeatedly with the same handle.
//
//export TP_Close
func TP_Close(reqJSON *C.char) *C.char {
	return C.CString(cffi.CloseJSON(goString(reqJSON)))
}

// TP_Stops returns the stop list for a previously-opened feed as JSON.
//
//export TP_Stops
func TP_Stops(reqJSON *C.char) *C.char {
	return C.CString(cffi.StopsJSON(goString(reqJSON)))
}

// TP_Route computes a single itinerary. The request can carry either a
// handle (the fast path, feed reused across calls) or an inline
// feedZip/feedDir (the legacy one-shot path).
//
//export TP_Route
func TP_Route(reqJSON *C.char) *C.char {
	return C.CString(cffi.RouteJSON(goString(reqJSON)))
}

// TP_Free releases a C string previously returned by any of the entry
// points above. Safe to call with a nil pointer.
//
//export TP_Free
func TP_Free(p *C.char) {
	if p == nil {
		return
	}
	C.free(unsafe.Pointer(p))
}

func goString(p *C.char) string {
	if p == nil {
		return ""
	}
	return C.GoString(p)
}

// main is required by -buildmode=c-shared but is never invoked at runtime.
func main() {}
