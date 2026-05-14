package router

import (
	"math"
	"sort"
)

// earthRadiusMeters is the mean Earth radius used by the haversine formula.
const earthRadiusMeters = 6371000.0

// NearbyStop pairs a stop with its great-circle distance from a query point.
type NearbyStop struct {
	Stop           Stop
	DistanceMeters float64
}

// HaversineMeters returns the great-circle distance in meters between two
// (latitude, longitude) coordinates expressed in decimal degrees.
func HaversineMeters(lat1, lon1, lat2, lon2 float64) float64 {
	phi1 := lat1 * math.Pi / 180
	phi2 := lat2 * math.Pi / 180
	dPhi := (lat2 - lat1) * math.Pi / 180
	dLambda := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(dPhi/2)*math.Sin(dPhi/2) +
		math.Cos(phi1)*math.Cos(phi2)*math.Sin(dLambda/2)*math.Sin(dLambda/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusMeters * c
}

// NearbyStops returns all stops within radiusMeters of the given point,
// sorted by ascending distance. The slice is empty when no stops match
// or when the feed has no stops.
func (f *Feed) NearbyStops(lat, lon, radiusMeters float64) []NearbyStop {
	if f == nil || len(f.Stops) == 0 {
		return nil
	}
	results := make([]NearbyStop, 0)
	for _, stop := range f.Stops {
		d := HaversineMeters(lat, lon, stop.Lat, stop.Lon)
		if d <= radiusMeters {
			results = append(results, NearbyStop{Stop: stop, DistanceMeters: d})
		}
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].DistanceMeters == results[j].DistanceMeters {
			return results[i].Stop.ID < results[j].Stop.ID
		}
		return results[i].DistanceMeters < results[j].DistanceMeters
	})
	return results
}

// NearestStop returns the single stop closest to the given coordinates along
// with its distance in meters. The boolean is false if the feed contains no
// stops.
func (f *Feed) NearestStop(lat, lon float64) (Stop, float64, bool) {
	if f == nil || len(f.Stops) == 0 {
		return Stop{}, 0, false
	}
	var (
		best    Stop
		bestDst = math.Inf(1)
		bestID  string
	)
	for _, stop := range f.Stops {
		d := HaversineMeters(lat, lon, stop.Lat, stop.Lon)
		if d < bestDst || (d == bestDst && stop.ID < bestID) {
			best = stop
			bestDst = d
			bestID = stop.ID
		}
	}
	return best, bestDst, true
}
