// Package index provides a compact, serializable in-memory representation of
// a parsed GTFS feed. It is optimized for fast load times on mobile devices
// by replacing string-keyed lookups with dense int32 indexes and by using
// encoding/gob for a single-pass binary format.
package index

import (
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"sort"

	"github.com/denysvitali/transit-planner/router"
)

// Magic and version identify the binary stream produced by CompiledFeed.WriteTo.
var magic = [4]byte{'T', 'P', 'F', 'D'}

// Version is the current binary format version. Bump on incompatible changes.
const Version uint32 = 1

// CompiledStop is a stop reduced to its identity plus coordinates.
// The string ID is retained for round-tripping back to the source feed.
type CompiledStop struct {
	ID   string
	Name string
	Lat  float64
	Lon  float64
}

// CompiledRoute mirrors router.Route with a stable identity.
type CompiledRoute struct {
	ID        string
	ShortName string
	LongName  string
	Type      int32
}

// CompiledStopTime is a single stop visit on a trip, referencing the dense
// stop index rather than the original GTFS stop_id string.
type CompiledStopTime struct {
	StopIdx   int32
	Sequence  int32
	Arrival   int32
	Departure int32
}

// CompiledTrip groups stop times for a single trip and references the dense
// route index.
type CompiledTrip struct {
	ID        string
	RouteIdx  int32
	ServiceID string
	StopTimes []CompiledStopTime
}

// CompiledTransfer references stops by dense index. Duration is in seconds.
type CompiledTransfer struct {
	FromStopIdx int32
	ToStopIdx   int32
	Duration    int32
}

// CompiledFeed is the dense, serializable form of router.Feed.
//
// The slices are the source of truth; the StopIndex / RouteIndex maps allow
// callers to translate a GTFS string id back into the dense slot. They are
// rebuilt on ReadFrom so they do not need to be transmitted on the wire.
type CompiledFeed struct {
	Stops     []CompiledStop
	Routes    []CompiledRoute
	Trips     []CompiledTrip
	Transfers []CompiledTransfer

	// stopIndex / routeIndex are derived lookup tables built from Stops and
	// Routes. They are unexported so encoding/gob does not include them in the
	// serialized payload; ReadFrom rebuilds them on load.
	stopIndex  map[string]int32
	routeIndex map[string]int32
}

// StopIndex returns the offset of the stop with the given GTFS id, or -1 if
// it is not present.
func (c *CompiledFeed) StopIndex(id string) int32 {
	if i, ok := c.stopIndex[id]; ok {
		return i
	}
	return -1
}

// RouteIndex returns the offset of the route with the given GTFS id, or -1 if
// it is not present.
func (c *CompiledFeed) RouteIndex(id string) int32 {
	if i, ok := c.routeIndex[id]; ok {
		return i
	}
	return -1
}

// Compile turns a parsed GTFS feed into a CompiledFeed with dense indexes.
// The input feed is not modified. Stops, routes and trips are emitted in a
// deterministic order (sorted by their GTFS id) so the binary output is
// reproducible regardless of map iteration order.
func Compile(feed *router.Feed) *CompiledFeed {
	if feed == nil {
		return &CompiledFeed{
			stopIndex:  map[string]int32{},
			routeIndex: map[string]int32{},
		}
	}

	stopIDs := sortedKeys(feed.Stops)
	stops := make([]CompiledStop, len(stopIDs))
	stopIndex := make(map[string]int32, len(stopIDs))
	for i, id := range stopIDs {
		s := feed.Stops[id]
		stops[i] = CompiledStop{
			ID:   s.ID,
			Name: s.Name,
			Lat:  s.Lat,
			Lon:  s.Lon,
		}
		stopIndex[id] = int32(i)
	}

	routeIDs := sortedKeys(feed.Routes)
	routes := make([]CompiledRoute, len(routeIDs))
	routeIndex := make(map[string]int32, len(routeIDs))
	for i, id := range routeIDs {
		r := feed.Routes[id]
		routes[i] = CompiledRoute{
			ID:        r.ID,
			ShortName: r.ShortName,
			LongName:  r.LongName,
			Type:      int32(r.Type),
		}
		routeIndex[id] = int32(i)
	}

	tripIDs := sortedKeys(feed.Trips)
	trips := make([]CompiledTrip, 0, len(tripIDs))
	for _, id := range tripIDs {
		t := feed.Trips[id]
		ridx, ok := routeIndex[t.RouteID]
		if !ok {
			// Skip trips referencing unknown routes rather than crashing; this
			// mirrors the lenient behavior of the existing CSV loader.
			ridx = -1
		}
		sts := make([]CompiledStopTime, 0, len(t.StopTimes))
		for _, st := range t.StopTimes {
			sidx, ok := stopIndex[st.StopID]
			if !ok {
				sidx = -1
			}
			sts = append(sts, CompiledStopTime{
				StopIdx:   sidx,
				Sequence:  int32(st.Sequence),
				Arrival:   int32(st.Arrival),
				Departure: int32(st.Departure),
			})
		}
		trips = append(trips, CompiledTrip{
			ID:        t.ID,
			RouteIdx:  ridx,
			ServiceID: t.ServiceID,
			StopTimes: sts,
		})
	}

	transfers := make([]CompiledTransfer, 0, len(feed.Transfers))
	for _, tr := range feed.Transfers {
		from, ok := stopIndex[tr.FromStopID]
		if !ok {
			from = -1
		}
		to, ok := stopIndex[tr.ToStopID]
		if !ok {
			to = -1
		}
		transfers = append(transfers, CompiledTransfer{
			FromStopIdx: from,
			ToStopIdx:   to,
			Duration:    int32(tr.Duration),
		})
	}

	return &CompiledFeed{
		Stops:      stops,
		Routes:     routes,
		Trips:      trips,
		Transfers:  transfers,
		stopIndex:  stopIndex,
		routeIndex: routeIndex,
	}
}

// WriteTo serializes the compiled feed to w as: magic("TPFD") || u32(version)
// || gob(payload). It returns the total number of bytes written.
func (c *CompiledFeed) WriteTo(w io.Writer) (int64, error) {
	cw := &countingWriter{w: w}
	if _, err := cw.Write(magic[:]); err != nil {
		return cw.n, err
	}
	var verBuf [4]byte
	binary.BigEndian.PutUint32(verBuf[:], Version)
	if _, err := cw.Write(verBuf[:]); err != nil {
		return cw.n, err
	}
	enc := gob.NewEncoder(cw)
	if err := enc.Encode(c); err != nil {
		return cw.n, fmt.Errorf("encoding compiled feed: %w", err)
	}
	return cw.n, nil
}

// ReadFrom decodes a CompiledFeed previously produced by WriteTo. It verifies
// the magic bytes and the version, then rebuilds the in-memory indexes.
func ReadFrom(r io.Reader) (*CompiledFeed, error) {
	var header [8]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, fmt.Errorf("reading header: %w", err)
	}
	if [4]byte{header[0], header[1], header[2], header[3]} != magic {
		return nil, errors.New("index: bad magic, not a compiled feed")
	}
	version := binary.BigEndian.Uint32(header[4:])
	if version != Version {
		return nil, fmt.Errorf("index: unsupported version %d (want %d)", version, Version)
	}

	var c CompiledFeed
	dec := gob.NewDecoder(r)
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("decoding compiled feed: %w", err)
	}

	c.stopIndex = make(map[string]int32, len(c.Stops))
	for i, s := range c.Stops {
		c.stopIndex[s.ID] = int32(i)
	}
	c.routeIndex = make(map[string]int32, len(c.Routes))
	for i, r := range c.Routes {
		c.routeIndex[r.ID] = int32(i)
	}
	return &c, nil
}

// sortedKeys returns the keys of m sorted in ascending order. It works for
// any map keyed by string.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// countingWriter tracks how many bytes have been written through it.
type countingWriter struct {
	w io.Writer
	n int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	return n, err
}
