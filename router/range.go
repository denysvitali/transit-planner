package router

import (
	"errors"
	"sort"
)

// RouteRange computes earliest-arrival itineraries for every reasonable
// departure time in the [fromTime, toTime] window. It returns the set of
// Pareto-optimal {departure, arrival} itineraries: no result is dominated by
// another with a strictly later departure and an earlier-or-equal arrival.
//
// Candidate departure times are derived from the GTFS StopTime.Departure
// values observed at the origin stop that fall inside the window. For each
// candidate the engine reuses the underlying single-departure search.
func (e *Engine) RouteRange(originStopID, destinationStopID string, fromTime, toTime int, options Options) ([]Itinerary, error) {
	if _, ok := e.feed.Stops[originStopID]; !ok {
		return nil, errors.New("origin stop not found")
	}
	if _, ok := e.feed.Stops[destinationStopID]; !ok {
		return nil, errors.New("destination stop not found")
	}
	if toTime < fromTime {
		return nil, errors.New("invalid window: toTime < fromTime")
	}

	departures := e.collectOriginDepartures(originStopID, fromTime, toTime)
	if len(departures) == 0 {
		return nil, nil
	}

	type candidate struct {
		departure int
		itin      Itinerary
	}
	candidates := make([]candidate, 0, len(departures))
	for _, t := range departures {
		itin, err := e.Route(originStopID, destinationStopID, t, options)
		if err != nil {
			// Unreachable for this departure; just skip.
			continue
		}
		dep := itineraryDeparture(itin, t)
		candidates = append(candidates, candidate{departure: dep, itin: itin})
	}
	if len(candidates) == 0 {
		return nil, errors.New("destination unreachable in window")
	}

	// Deduplicate exact (departure, arrival) pairs.
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].departure != candidates[j].departure {
			return candidates[i].departure < candidates[j].departure
		}
		return candidates[i].itin.Arrival < candidates[j].itin.Arrival
	})
	deduped := candidates[:0]
	seen := map[[2]int]struct{}{}
	for _, c := range candidates {
		key := [2]int{c.departure, c.itin.Arrival}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		deduped = append(deduped, c)
	}

	// Drop dominated entries: an itinerary is dominated if some other
	// itinerary has a strictly later departure AND an earlier-or-equal arrival.
	results := make([]Itinerary, 0, len(deduped))
	for i, ci := range deduped {
		dominated := false
		for j, cj := range deduped {
			if i == j {
				continue
			}
			if cj.departure > ci.departure && cj.itin.Arrival <= ci.itin.Arrival {
				dominated = true
				break
			}
		}
		if !dominated {
			results = append(results, ci.itin)
		}
	}

	sort.SliceStable(results, func(i, j int) bool {
		di := itineraryDeparture(results[i], 0)
		dj := itineraryDeparture(results[j], 0)
		if di != dj {
			return di < dj
		}
		return results[i].Arrival < results[j].Arrival
	})
	return results, nil
}

// collectOriginDepartures gathers all StopTime.Departure values at the origin
// stop that fall within [fromTime, toTime]. The origin stop must not be the
// final stop of the trip in question (passengers can only board at non-final
// stops). Duplicate timestamps across trips are returned once each.
func (e *Engine) collectOriginDepartures(originStopID string, fromTime, toTime int) []int {
	seen := map[int]struct{}{}
	for _, boarding := range e.tripsByStop[originStopID] {
		trip := boarding.trip
		stopTime := trip.StopTimes[boarding.stopIndex]
		if boarding.stopIndex == len(trip.StopTimes)-1 {
			// Cannot board at the final stop of a trip.
			continue
		}
		if stopTime.Departure < fromTime || stopTime.Departure > toTime {
			continue
		}
		seen[stopTime.Departure] = struct{}{}
	}
	out := make([]int, 0, len(seen))
	for t := range seen {
		out = append(out, t)
	}
	sort.Ints(out)
	return out
}

// itineraryDeparture returns the departure timestamp of the first transit or
// walk leg in the itinerary. When the itinerary has no legs (origin equals
// destination), the provided fallback is returned.
func itineraryDeparture(itin Itinerary, fallback int) int {
	if len(itin.Legs) == 0 {
		return fallback
	}
	return itin.Legs[0].Departure
}
