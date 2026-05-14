package catalog

import (
	"strings"
	"testing"
)

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

func TestMobilityDatabaseFeedsUseMirroredLatestZip(t *testing.T) {
	for id, feed := range Feeds {
		if !strings.HasPrefix(id, "jbda-") {
			continue
		}
		if feed.LocalFileName != id+".zip" {
			t.Fatalf("%s local file name = %q, want %q", id, feed.LocalFileName, id+".zip")
		}
		if feed.SourceURL == "" {
			t.Fatalf("%s has empty source URL", id)
		}
	}
}
