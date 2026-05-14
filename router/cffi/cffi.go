// Package cffi is the pure-Go core that powers the C-ABI / dart:ffi surface
// of the transit-planner router. It accepts JSON requests and returns JSON
// responses, so the actual cgo wrapper (in cmd/libtransitplanner) stays
// tiny — it only forwards C strings to and from RouteJSON.
//
// Keeping this package pure-Go means it is exercised by the default
// `go test ./...` flow without needing a C toolchain in CI, and it is
// importable from the rest of the Go codebase (e.g. the CLI router) without
// pulling cgo in transitively.
package cffi

import (
	"encoding/json"
	"errors"

	"github.com/denysvitali/transit-planner/router"
)

// routeRequest is the JSON payload accepted by the FFI entry point.
//
// Exactly one of FeedDir and FeedZip must be set:
//   - FeedDir points at a directory of extracted GTFS CSV files.
//   - FeedZip points at a zip archive in the shape published by agencies
//     (e.g. the ODPT Toei feeds). The mobile app ships zips as assets so
//     this is the production path.
type routeRequest struct {
	FeedDir      string `json:"feedDir,omitempty"`
	FeedZip      string `json:"feedZip,omitempty"`
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

// routeResponse is the success payload returned by RouteJSON.
type routeResponse struct {
	Arrival   int          `json:"arrival"`
	Transfers int          `json:"transfers"`
	Legs      []legPayload `json:"legs"`
}

// errorResponse is returned whenever RouteJSON cannot produce an itinerary.
type errorResponse struct {
	Error string `json:"error"`
}

// RouteJSON is the single entrypoint of the FFI surface. It loads the
// requested GTFS feed and runs a route query against it. Both input and
// output are JSON strings; on any failure the response is shaped like
// {"error": "..."}, never an empty string.
func RouteJSON(raw string) string {
	if raw == "" {
		return errorJSON("request is null")
	}

	var req routeRequest
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		return errorJSON("invalid request JSON: " + err.Error())
	}

	feed, err := loadFeed(req)
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

func loadFeed(req routeRequest) (*router.Feed, error) {
	switch {
	case req.FeedZip != "" && req.FeedDir != "":
		return nil, errors.New("specify exactly one of feedZip or feedDir")
	case req.FeedZip != "":
		return router.LoadGTFSZip(req.FeedZip)
	case req.FeedDir != "":
		return router.LoadGTFS(req.FeedDir)
	default:
		return nil, errors.New("feedDir or feedZip is required")
	}
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
