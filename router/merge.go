package router

import "sort"

// Merge combines several feeds into one, namespacing each feed's IDs with a
// stable prefix so callers can plan trips across operators that publish their
// schedules independently (e.g. Toei subway + Kobe community bus).
//
// IDs become "<prefix>:<original_id>". Stop names are preserved unchanged so
// that the name-based transfer fallback in appendSameStationTransfers can
// stitch together same-named stations across feeds — that is what makes a
// Tokyo→Osaka query routable when each leg lives in its own GTFS bundle.
//
// The input map is keyed by the prefix to use for that feed. Prefixes must be
// non-empty and unique; the function panics if two prefixes collide because
// that would silently lose data.
func Merge(feeds map[string]*Feed) *Feed {
	if len(feeds) == 0 {
		return &Feed{
			Stops:  map[string]Stop{},
			Routes: map[string]Route{},
			Trips:  map[string]Trip{},
		}
	}

	prefixes := make([]string, 0, len(feeds))
	for prefix := range feeds {
		if prefix == "" {
			panic("router.Merge: empty feed prefix")
		}
		prefixes = append(prefixes, prefix)
	}
	sort.Strings(prefixes)

	merged := &Feed{
		Stops:  map[string]Stop{},
		Routes: map[string]Route{},
		Trips:  map[string]Trip{},
	}

	for _, prefix := range prefixes {
		feed := feeds[prefix]
		if feed == nil {
			continue
		}
		ns := prefix + ":"

		for id, stop := range feed.Stops {
			nsID := ns + id
			if _, dup := merged.Stops[nsID]; dup {
				panic("router.Merge: stop id collision after prefixing: " + nsID)
			}
			stop.ID = nsID
			merged.Stops[nsID] = stop
		}

		for id, route := range feed.Routes {
			nsID := ns + id
			route.ID = nsID
			merged.Routes[nsID] = route
		}

		for id, trip := range feed.Trips {
			nsID := ns + id
			trip.ID = nsID
			trip.RouteID = ns + trip.RouteID
			// ServiceID stays per-feed; calendars are loaded per-feed when
			// callers want service filtering. The router itself ignores it.
			times := make([]StopTime, len(trip.StopTimes))
			for i, st := range trip.StopTimes {
				st.StopID = ns + st.StopID
				times[i] = st
			}
			trip.StopTimes = times
			merged.Trips[nsID] = trip
		}

		for _, t := range feed.Transfers {
			merged.Transfers = append(merged.Transfers, Transfer{
				FromStopID: ns + t.FromStopID,
				ToStopID:   ns + t.ToStopID,
				Duration:   t.Duration,
			})
		}
	}

	merged.Transfers = appendSameStationTransfers(merged.Stops, merged.Transfers)
	return merged
}
