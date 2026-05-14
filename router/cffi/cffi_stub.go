//go:build !cgo

// Package cffi is a no-op stub when cgo is disabled. The real C-ABI surface
// lives in cffi.go and is gated on the `cgo` build tag so that builds of the
// rest of the module continue to work without a C toolchain.
package cffi
