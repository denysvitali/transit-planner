package router

import (
	"container/list"
	"errors"
	"math"
	"sort"
)

const unreachable = math.MaxInt / 4

type Engine struct {
	feed            *Feed
	tripsByStop     map[string][]Trip
	transfersByStop map[string][]Transfer
}

type Options struct {
	MaxTransfers int
}

type Leg struct {
	Mode      string
	RouteID   string
	TripID    string
	FromStop  Stop
	ToStop    Stop
	Departure int
	Arrival   int
}

type Itinerary struct {
	Legs      []Leg
	Arrival   int
	Transfers int
}

type label struct {
	stopID string
	time   int
	prev   *label
	leg    Leg
	round  int
}

func NewEngine(feed *Feed) *Engine {
	tripsByStop := map[string][]Trip{}
	for _, trip := range feed.Trips {
		for _, stopTime := range trip.StopTimes {
			tripsByStop[stopTime.StopID] = append(tripsByStop[stopTime.StopID], trip)
		}
	}
	transfersByStop := map[string][]Transfer{}
	for _, transfer := range feed.Transfers {
		transfersByStop[transfer.FromStopID] = append(transfersByStop[transfer.FromStopID], transfer)
	}
	return &Engine{
		feed:            feed,
		tripsByStop:     tripsByStop,
		transfersByStop: transfersByStop,
	}
}

func (e *Engine) Route(originStopID, destinationStopID string, departure int, options Options) (Itinerary, error) {
	if _, ok := e.feed.Stops[originStopID]; !ok {
		return Itinerary{}, errors.New("origin stop not found")
	}
	if _, ok := e.feed.Stops[destinationStopID]; !ok {
		return Itinerary{}, errors.New("destination stop not found")
	}
	maxRounds := options.MaxTransfers + 1
	if maxRounds < 1 {
		maxRounds = 1
	}

	best := make([]map[string]*label, maxRounds+1)
	for i := range best {
		best[i] = map[string]*label{}
	}
	start := &label{stopID: originStopID, time: departure}
	best[0][originStopID] = start

	for round := 0; round < maxRounds; round++ {
		e.applyTransfers(best[round])
		for stopID, current := range best[round] {
			for _, trip := range e.tripsByStop[stopID] {
				boardIndex := firstBoardableIndex(trip.StopTimes, stopID, current.time)
				if boardIndex < 0 {
					continue
				}
				board := trip.StopTimes[boardIndex]
				for i := boardIndex + 1; i < len(trip.StopTimes); i++ {
					alight := trip.StopTimes[i]
					existing := best[round+1][alight.StopID]
					if existing != nil && existing.time <= alight.Arrival {
						continue
					}
					best[round+1][alight.StopID] = &label{
						stopID: alight.StopID,
						time:   alight.Arrival,
						prev:   current,
						round:  round + 1,
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
				}
			}
		}
	}
	e.applyTransfers(best[maxRounds])

	var destination *label
	for _, round := range best {
		if candidate := round[destinationStopID]; candidate != nil {
			if destination == nil || candidate.time < destination.time {
				destination = candidate
			}
		}
	}
	if destination == nil {
		return Itinerary{}, errors.New("destination unreachable")
	}
	return e.buildItinerary(destination), nil
}

func (e *Engine) applyTransfers(round map[string]*label) {
	queue := list.New()
	for _, item := range round {
		queue.PushBack(item)
	}
	for queue.Len() > 0 {
		front := queue.Front()
		queue.Remove(front)
		current := front.Value.(*label)
		for _, transfer := range e.transfersByStop[current.stopID] {
			nextTime := current.time + transfer.Duration
			existing := round[transfer.ToStopID]
			if existing != nil && existing.time <= nextTime {
				continue
			}
			next := &label{
				stopID: transfer.ToStopID,
				time:   nextTime,
				prev:   current,
				round:  current.round,
				leg: Leg{
					Mode:      "walk",
					FromStop:  e.feed.Stops[transfer.FromStopID],
					ToStop:    e.feed.Stops[transfer.ToStopID],
					Departure: current.time,
					Arrival:   nextTime,
				},
			}
			round[transfer.ToStopID] = next
			queue.PushBack(next)
		}
	}
}

func (e *Engine) buildItinerary(destination *label) Itinerary {
	var legs []Leg
	for cursor := destination; cursor != nil && cursor.prev != nil; cursor = cursor.prev {
		legs = append(legs, cursor.leg)
	}
	sort.SliceStable(legs, func(i, j int) bool { return i > j })
	transfers := 0
	transitLegs := 0
	for _, leg := range legs {
		if leg.Mode == "transit" {
			transitLegs++
		}
	}
	if transitLegs > 0 {
		transfers = transitLegs - 1
	}
	return Itinerary{
		Legs:      legs,
		Arrival:   destination.time,
		Transfers: transfers,
	}
}

func firstBoardableIndex(times []StopTime, stopID string, earliest int) int {
	for i, stopTime := range times {
		if stopTime.StopID == stopID && stopTime.Departure >= earliest {
			return i
		}
	}
	return -1
}
