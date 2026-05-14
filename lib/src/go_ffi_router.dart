// Conditional export for the Go-FFI router. dart:ffi is unavailable on the
// web target, so the import below resolves to go_ffi_router_stub.dart there
// (which returns a MockTransitRouter) and to go_ffi_router_io.dart on every
// platform that exposes dart:io.
export 'go_ffi_router_stub.dart'
    if (dart.library.io) 'go_ffi_router_io.dart';
