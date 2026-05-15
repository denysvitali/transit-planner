package catalog

import "sort"

type FeedSpec struct {
	ID                   string
	Name                 string
	Description          string
	Country              string
	Region               string
	Publisher            string
	License              string
	SourceURL            string
	LocalFileName        string
	Attribution          string
	CenterLatitude       float64
	CenterLongitude      float64
	DefaultDepartureHour *int
	BundledAssetPath     string
}

type NetworkSpec struct {
	ID                   string
	Name                 string
	Description          string
	Country              string
	Region               string
	Publisher            string
	License              string
	SourceURL            string
	Attribution          string
	CenterLatitude       float64
	CenterLongitude      float64
	DefaultDepartureHour *int
	ComponentFeedIDs     []string
}

// Runtime apps discover Transitland feeds through the API. These maps remain
// empty so tooling can still share FeedSpec/NetworkSpec without carrying a
// manually-maintained feed inventory.
var Feeds = map[string]FeedSpec{}
var Networks = []NetworkSpec{}

func SortedFeeds() []FeedSpec {
	out := make([]FeedSpec, 0, len(Feeds))
	for _, feed := range Feeds {
		out = append(out, feed)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Country != out[j].Country {
			return out[i].Country < out[j].Country
		}
		return out[i].ID < out[j].ID
	})
	return out
}
