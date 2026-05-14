// Package main is a thin cgo wrapper around router/cffi that produces the
// dynamic library consumed by the Flutter app over dart:ffi.
//
// All routing logic lives in router/cffi.RouteJSON (pure Go) so it is fully
// tested by the default `go test ./...` flow without a C toolchain. This
// wrapper only marshals C strings in and out.
//
// Build for the host platform:
//
//	CGO_ENABLED=1 go build -buildmode=c-shared \
//	  -o build/libtransit_planner.so ./cmd/libtransitplanner
//
// For Android / iOS, drive the same command with the NDK / Xcode toolchain
// in CC plus per-ABI GOOS/GOARCH.
package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"unsafe"

	"github.com/denysvitali/transit-planner/router/cffi"
)

// TP_Route accepts a JSON request and returns a JSON response. The returned
// pointer is allocated by cgo and must be released by the caller via
// TP_Free; failing to do so leaks memory on every call.
//
//export TP_Route
func TP_Route(reqJSON *C.char) *C.char {
	var raw string
	if reqJSON != nil {
		raw = C.GoString(reqJSON)
	}
	return C.CString(cffi.RouteJSON(raw))
}

// TP_Free releases a C string previously returned by TP_Route. Safe to call
// with a nil pointer.
//
//export TP_Free
func TP_Free(p *C.char) {
	if p == nil {
		return
	}
	C.free(unsafe.Pointer(p))
}

// main is required by -buildmode=c-shared but is never invoked at runtime.
func main() {}
