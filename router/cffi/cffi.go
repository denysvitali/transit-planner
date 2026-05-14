//go:build cgo

// Package cffi exposes a C-ABI surface around the transit-planner router so
// that the Flutter app can drive it through dart:ffi. The package is intended
// to be built with `go build -buildmode=c-shared` to produce a dynamic library
// (.so / .dylib / .dll) plus the matching C header.
//
// All exported functions exchange JSON encoded payloads to keep the FFI
// boundary trivial: the caller is responsible for releasing every C string
// returned by this package via TP_Free.
package cffi

/*
#include <stdlib.h>
*/
import "C"

import (
	"encoding/json"
	"unsafe"

	"github.com/denysvitali/transit-planner/router"
)

// routeRequest is the JSON payload accepted by TP_Route.
type routeRequest struct {
	FeedDir      string `json:"feedDir"`
	From         string `json:"from"`
	To           string `json:"to"`
	Departure    int    `json:"departure"`
	MaxTransfers int    `json:"maxTransfers"`
}

// legPayload mirrors router.Leg using only JSON-friendly types.
type legPayload struct {
	Mode      string      `json:"mode"`
	RouteID   string      `json:"routeId,omitempty"`
	TripID    string      `json:"tripId,omitempty"`
	FromStop  router.Stop `json:"fromStop"`
	ToStop    router.Stop `json:"toStop"`
	Departure int         `json:"departure"`
	Arrival   int         `json:"arrival"`
}

// routeResponse is the success payload returned by TP_Route.
type routeResponse struct {
	Arrival   int          `json:"arrival"`
	Transfers int          `json:"transfers"`
	Legs      []legPayload `json:"legs"`
}

// errorResponse is returned whenever TP_Route cannot produce an itinerary.
type errorResponse struct {
	Error string `json:"error"`
}

// TP_Route computes a single itinerary between two stops in the GTFS feed
// stored at the directory referenced by the request. Both the input and the
// output are JSON strings; the returned C string must be released by the
// caller via TP_Free.
//
//export TP_Route
func TP_Route(reqJSON *C.char) *C.char {
	if reqJSON == nil {
		return C.CString(RouteJSON(""))
	}
	return C.CString(RouteJSON(C.GoString(reqJSON)))
}

// RouteJSON is the pure-Go core of TP_Route. It is exported so that the
// c-shared wrapper in cmd/libtransitplanner can reuse it, and so that tests
// can exercise the routing logic without having to import "C" (the Go
// toolchain forbids cgo imports inside _test.go files that share their
// package with cgo code).
func RouteJSON(raw string) string {
	if raw == "" {
		return errorJSON("request is null")
	}

	var req routeRequest
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		return errorJSON("invalid request JSON: " + err.Error())
	}

	feed, err := router.LoadGTFS(req.FeedDir)
	if err != nil {
		return errorJSON("load feed: " + err.Error())
	}

	engine := router.NewEngine(feed)
	itinerary, err := engine.Route(req.From, req.To, req.Departure, router.Options{
		MaxTransfers: req.MaxTransfers,
	})
	if err != nil {
		return errorJSON(err.Error())
	}

	legs := make([]legPayload, 0, len(itinerary.Legs))
	for _, leg := range itinerary.Legs {
		legs = append(legs, legPayload{
			Mode:      leg.Mode,
			RouteID:   leg.RouteID,
			TripID:    leg.TripID,
			FromStop:  leg.FromStop,
			ToStop:    leg.ToStop,
			Departure: leg.Departure,
			Arrival:   leg.Arrival,
		})
	}
	resp := routeResponse{
		Arrival:   itinerary.Arrival,
		Transfers: itinerary.Transfers,
		Legs:      legs,
	}
	data, err := json.Marshal(resp)
	if err != nil {
		return errorJSON("encode response: " + err.Error())
	}
	return string(data)
}

// TP_Free releases a C string previously returned by this package. It is safe
// to invoke with a nil pointer.
//
//export TP_Free
func TP_Free(p *C.char) {
	if p == nil {
		return
	}
	C.free(unsafe.Pointer(p))
}

func errorJSON(message string) string {
	data, err := json.Marshal(errorResponse{Error: message})
	if err != nil {
		// Fall back to a hard-coded payload that is guaranteed to be valid
		// JSON when even the error encoder fails.
		return `{"error":"unknown error"}`
	}
	return string(data)
}
