package catalog

import "testing"

func TestNetworksReferenceKnownFeeds(t *testing.T) {
	for _, network := range Networks {
		if len(network.ComponentFeedIDs) == 0 {
			t.Fatalf("%s has no component feeds", network.ID)
		}
		for _, id := range network.ComponentFeedIDs {
			if _, ok := Feeds[id]; !ok {
				t.Fatalf("%s references unknown feed %s", network.ID, id)
			}
		}
	}
}

func TestDownloadableFeedsUseTransitlandEndpoints(t *testing.T) {
	for id, feed := range Feeds {
		if feed.LocalFileName == "" {
			t.Fatalf("%s has empty local file name", id)
		}
		if got, want := feed.SourceURL, "https://transit.land/api/v2/rest/feeds/"; len(got) < len(want) || got[:len(want)] != want {
			t.Fatalf("%s source URL = %q, want Transitland REST endpoint", id, feed.SourceURL)
		}
	}
}
