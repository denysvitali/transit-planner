// Package cffi is the pure-Go core that powers the C-ABI / dart:ffi surface
// of the transit-planner router. It accepts JSON requests and returns JSON
// responses, so the actual cgo wrapper (in cmd/libtransitplanner) stays
// tiny — it only forwards C strings to and from these functions.
//
// Keeping this package pure-Go means it is exercised by the default
// `go test ./...` flow without needing a C toolchain in CI, and it is
// importable from the rest of the Go codebase without pulling cgo in
// transitively.
package cffi

import (
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/denysvitali/transit-planner/router"
)

// feedHandle bundles a parsed GTFS feed with its prebuilt engine so that
// subsequent route queries can reuse the same indexes. The engine is the
// expensive piece — for the Toei subway feed it owns ~5 600 trip records —
// so reloading per call wastes 300-500 ms each time.
type feedHandle struct {
	feed   *router.Feed
	engine *router.Engine
}

var (
	handles    sync.Map // int64 → *feedHandle
	nextHandle atomic.Int64
)

// openRequest opens a GTFS feed and returns a handle that subsequent calls
// reference. Exactly one of FeedZip and FeedDir must be set.
type openRequest struct {
	FeedDir string `json:"feedDir,omitempty"`
	FeedZip string `json:"feedZip,omitempty"`
}

type openResponse struct {
	Handle int64 `json:"handle"`
}

// closeRequest releases a previously-opened feed. It is safe to call with a
// handle that no longer exists.
type closeRequest struct {
	Handle int64 `json:"handle"`
}

// stopsRequest asks for the stops belonging to a feed.
type stopsRequest struct {
	Handle int64 `json:"handle"`
}

type stopPayload struct {
	ID   string  `json:"id"`
	Name string  `json:"name"`
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
}

type stopsResponse struct {
	Stops []stopPayload `json:"stops"`
}

// routeRequest is the routing payload. Provide either Handle (the fast path
// — feed reused across calls) or FeedZip/FeedDir (the legacy one-shot path
// that loads the feed for this call only and discards it).
type routeRequest struct {
	Handle       int64  `json:"handle,omitempty"`
	FeedDir      string `json:"feedDir,omitempty"`
	FeedZip      string `json:"feedZip,omitempty"`
	From         string `json:"from"`
	To           string `json:"to"`
	Departure    int    `json:"departure"`
	MaxTransfers int    `json:"maxTransfers"`
	RouteTypes   []int  `json:"routeTypes,omitempty"`
}

type legPayload struct {
	Mode      string      `json:"mode"`
	RouteID   string      `json:"routeId,omitempty"`
	TripID    string      `json:"tripId,omitempty"`
	RouteType int         `json:"routeType,omitempty"`
	FromStop  router.Stop `json:"fromStop"`
	ToStop    router.Stop `json:"toStop"`
	Departure int         `json:"departure"`
	Arrival   int         `json:"arrival"`
}

type routeResponse struct {
	Arrival   int          `json:"arrival"`
	Transfers int          `json:"transfers"`
	Legs      []legPayload `json:"legs"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// OpenJSON loads a GTFS feed (zip or directory) and returns a handle the
// caller can pass to StopsJSON / RouteJSON / CloseJSON.
func OpenJSON(raw string) string {
	if raw == "" {
		return errorJSON("request is null")
	}
	var req openRequest
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		return errorJSON("invalid request JSON: " + err.Error())
	}
	feed, err := loadFeed(req.FeedDir, req.FeedZip)
	if err != nil {
		return errorJSON("load feed: " + err.Error())
	}
	h := nextHandle.Add(1)
	handles.Store(h, &feedHandle{feed: feed, engine: router.NewEngine(feed)})
	return mustMarshal(openResponse{Handle: h})
}

// CloseJSON releases the feed previously returned by OpenJSON. Calling with
// an unknown handle is a no-op so the caller can be idempotent.
func CloseJSON(raw string) string {
	if raw == "" {
		return errorJSON("request is null")
	}
	var req closeRequest
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		return errorJSON("invalid request JSON: " + err.Error())
	}
	handles.Delete(req.Handle)
	return `{}`
}

// StopsJSON returns every stop in a previously-opened feed.
func StopsJSON(raw string) string {
	if raw == "" {
		return errorJSON("request is null")
	}
	var req stopsRequest
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		return errorJSON("invalid request JSON: " + err.Error())
	}
	h, ok := lookupHandle(req.Handle)
	if !ok {
		return errorJSON("unknown handle")
	}
	stops := make([]stopPayload, 0, len(h.feed.Stops))
	for _, s := range h.feed.Stops {
		stops = append(stops, stopPayload{
			ID:   s.ID,
			Name: s.Name,
			Lat:  s.Lat,
			Lon:  s.Lon,
		})
	}
	return mustMarshal(stopsResponse{Stops: stops})
}

// RouteJSON computes a single itinerary. It accepts either a handle (fast)
// or an inline feedZip/feedDir (slow, one-shot — useful for tests and the
// non-interactive CLI).
func RouteJSON(raw string) string {
	if raw == "" {
		return errorJSON("request is null")
	}
	var req routeRequest
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		return errorJSON("invalid request JSON: " + err.Error())
	}

	engine, err := engineFor(req)
	if err != nil {
		return errorJSON(err.Error())
	}

	itinerary, err := engine.Route(req.From, req.To, req.Departure, router.Options{
		MaxTransfers:      req.MaxTransfers,
		AllowedRouteTypes: allowedRouteTypes(req.RouteTypes),
	})
	if err != nil {
		return errorJSON(err.Error())
	}

	legs := make([]legPayload, 0, len(itinerary.Legs))
	for _, leg := range itinerary.Legs {
		routeType := -1
		if gtfsRouteType, ok := engine.RouteType(leg.RouteID); ok {
			routeType = gtfsRouteType
		}
		legs = append(legs, legPayload{
			Mode:      leg.Mode,
			RouteID:   leg.RouteID,
			TripID:    leg.TripID,
			RouteType: routeType,
			FromStop:  leg.FromStop,
			ToStop:    leg.ToStop,
			Departure: leg.Departure,
			Arrival:   leg.Arrival,
		})
	}
	return mustMarshal(routeResponse{
		Arrival:   itinerary.Arrival,
		Transfers: itinerary.Transfers,
		Legs:      legs,
	})
}

func allowedRouteTypes(routeTypes []int) map[int]bool {
	if routeTypes == nil {
		return nil
	}
	allowed := make(map[int]bool, len(routeTypes))
	for _, routeType := range routeTypes {
		allowed[routeType] = true
	}
	return allowed
}

func engineFor(req routeRequest) (*router.Engine, error) {
	if req.Handle != 0 {
		h, ok := lookupHandle(req.Handle)
		if !ok {
			return nil, errors.New("unknown handle")
		}
		return h.engine, nil
	}
	feed, err := loadFeed(req.FeedDir, req.FeedZip)
	if err != nil {
		return nil, errors.New("load feed: " + err.Error())
	}
	return router.NewEngine(feed), nil
}

func loadFeed(dir, zip string) (*router.Feed, error) {
	switch {
	case zip != "" && dir != "":
		return nil, errors.New("specify exactly one of feedZip or feedDir")
	case zip != "":
		return router.LoadGTFSZip(zip)
	case dir != "":
		return router.LoadGTFS(dir)
	default:
		return nil, errors.New("feedDir or feedZip is required")
	}
}

func lookupHandle(h int64) (*feedHandle, bool) {
	v, ok := handles.Load(h)
	if !ok {
		return nil, false
	}
	return v.(*feedHandle), true
}

func mustMarshal(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return errorJSON("encode response: " + err.Error())
	}
	return string(data)
}

func errorJSON(message string) string {
	data, err := json.Marshal(errorResponse{Error: message})
	if err != nil {
		return `{"error":"unknown error"}`
	}
	return string(data)
}
