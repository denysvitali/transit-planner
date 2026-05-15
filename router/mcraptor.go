package router

import (
	"container/list"
	"errors"
	"sort"
)

type MultiOptions struct {
	MaxTransfers int
}

type mcLabel struct {
	stopID    string
	arrival   int
	transfers int
	walk      int
	prev      *mcLabel
	leg       Leg
}

func (e *Engine) RouteMulti(originStopID, destinationStopID string, departure int, opts MultiOptions) ([]Itinerary, error) {
	if _, ok := e.feed.Stops[originStopID]; !ok {
		return nil, errors.New("origin stop not found")
	}
	if _, ok := e.feed.Stops[destinationStopID]; !ok {
		return nil, errors.New("destination stop not found")
	}
	maxRounds := opts.MaxTransfers + 1
	if maxRounds < 1 {
		maxRounds = 1
	}

	// labels[stopID] holds the current Pareto set for the stop across all rounds.
	labels := map[string][]*mcLabel{}
	start := &mcLabel{stopID: originStopID, arrival: departure}
	labels[originStopID] = []*mcLabel{start}

	// per-round frontier: labels created in the previous round used to extend in this round.
	frontier := map[string][]*mcLabel{originStopID: {start}}
	applyMcTransfers(e, labels, frontier)

	for round := 0; round < maxRounds; round++ {
		next := map[string][]*mcLabel{}
		for stopID, current := range frontier {
			for _, src := range current {
				for _, boarding := range e.tripsByStop[stopID] {
					trip := boarding.trip
					if trip.StopTimes[boarding.stopIndex].Departure < src.arrival {
						continue
					}
					board := trip.StopTimes[boarding.stopIndex]
					addedTransfer := 0
					// Boarding a transit leg after an existing transit leg counts as a transfer.
					if hasTransitAncestor(src) {
						addedTransfer = 1
					}
					for i := boarding.stopIndex + 1; i < len(trip.StopTimes); i++ {
						alight := trip.StopTimes[i]
						candidate := &mcLabel{
							stopID:    alight.StopID,
							arrival:   alight.Arrival,
							transfers: src.transfers + addedTransfer,
							walk:      src.walk,
							prev:      src,
							leg: Leg{
								Mode:      "transit",
								RouteID:   trip.RouteID,
								TripID:    trip.ID,
								FromStop:  e.feed.Stops[board.StopID],
								ToStop:    e.feed.Stops[alight.StopID],
								Departure: board.Departure,
								Arrival:   alight.Arrival,
							},
						}
						if candidate.transfers > opts.MaxTransfers {
							continue
						}
						if addLabel(labels, candidate) {
							next[alight.StopID] = append(next[alight.StopID], candidate)
						}
					}
				}
			}
		}
		if len(next) == 0 {
			break
		}
		applyMcTransfers(e, labels, next)
		frontier = next
	}

	dest := labels[destinationStopID]
	if len(dest) == 0 {
		return nil, errors.New("destination unreachable")
	}
	// Final Pareto filter then sort by arrival.
	sorted := make([]*mcLabel, len(dest))
	copy(sorted, dest)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].arrival != sorted[j].arrival {
			return sorted[i].arrival < sorted[j].arrival
		}
		if sorted[i].transfers != sorted[j].transfers {
			return sorted[i].transfers < sorted[j].transfers
		}
		return sorted[i].walk < sorted[j].walk
	})
	result := make([]Itinerary, 0, len(sorted))
	for _, lbl := range sorted {
		result = append(result, buildMcItinerary(lbl))
	}
	return result, nil
}

func applyMcTransfers(e *Engine, labels map[string][]*mcLabel, frontier map[string][]*mcLabel) {
	queue := list.New()
	for _, items := range frontier {
		for _, l := range items {
			queue.PushBack(l)
		}
	}
	for queue.Len() > 0 {
		front := queue.Front()
		queue.Remove(front)
		current := front.Value.(*mcLabel)
		for _, transfer := range e.transfersByStop[current.stopID] {
			// Disallow chained walks: a transfer must follow either the origin or a transit leg.
			if current.prev != nil && current.leg.Mode == "walk" {
				continue
			}
			nextArrival := current.arrival + transfer.Duration
			candidate := &mcLabel{
				stopID:    transfer.ToStopID,
				arrival:   nextArrival,
				transfers: current.transfers,
				walk:      current.walk + transfer.Duration,
				prev:      current,
				leg: Leg{
					Mode:      "walk",
					FromStop:  e.feed.Stops[transfer.FromStopID],
					ToStop:    e.feed.Stops[transfer.ToStopID],
					Departure: current.arrival,
					Arrival:   nextArrival,
				},
			}
			if addLabel(labels, candidate) {
				frontier[transfer.ToStopID] = append(frontier[transfer.ToStopID], candidate)
				queue.PushBack(candidate)
			}
		}
	}
}

// addLabel inserts l into labels[stopID] if it isn't dominated by an existing label.
// Returns true when the label was added (and dominated existing labels were removed).
func addLabel(labels map[string][]*mcLabel, l *mcLabel) bool {
	existing := labels[l.stopID]
	kept := existing[:0]
	for _, e := range existing {
		if dominates(e, l) {
			return false
		}
		if dominates(l, e) {
			continue
		}
		kept = append(kept, e)
	}
	kept = append(kept, l)
	labels[l.stopID] = kept
	return true
}

func dominates(a, b *mcLabel) bool {
	if a.arrival > b.arrival || a.transfers > b.transfers || a.walk > b.walk {
		return false
	}
	return a.arrival < b.arrival || a.transfers < b.transfers || a.walk < b.walk
}

func hasTransitAncestor(l *mcLabel) bool {
	for cursor := l; cursor != nil && cursor.prev != nil; cursor = cursor.prev {
		if cursor.leg.Mode == "transit" {
			return true
		}
	}
	return false
}

func buildMcItinerary(destination *mcLabel) Itinerary {
	var legs []Leg
	for cursor := destination; cursor != nil && cursor.prev != nil; cursor = cursor.prev {
		legs = append(legs, cursor.leg)
	}
	// Reverse legs.
	for i, j := 0, len(legs)-1; i < j; i, j = i+1, j-1 {
		legs[i], legs[j] = legs[j], legs[i]
	}
	return Itinerary{
		Legs:      legs,
		Arrival:   destination.arrival,
		Transfers: destination.transfers,
	}
}
