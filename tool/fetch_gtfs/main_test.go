package main

import (
	"strings"
	"testing"
)

func TestMobilityDatabaseFeedSpecsFiltersActiveNoKeyGTFS(t *testing.T) {
	const csvData = `id,data_type,entity_type,location.country_code,location.subdivision_name,location.municipality,provider,is_official,name,note,feed_contact_email,static_reference,urls.direct_download,urls.authentication_type,urls.authentication_info,urls.api_key_parameter_name,urls.latest,urls.license,location.bounding_box.minimum_latitude,location.bounding_box.maximum_latitude,location.bounding_box.minimum_longitude,location.bounding_box.maximum_longitude,location.bounding_box.extracted_on,status,features,redirect.id,redirect.comment
jp-open,gtfs,,JP,Tokyo,Tokyo,Open Publisher,True,Open Tokyo,,,,https://example.com/direct.zip,0,,,https://files.mobilitydatabase.org/jp-open/latest.zip,https://creativecommons.org/licenses/by/4.0/deed.ja,,,,,,active,,,
jp-auth,gtfs,,JP,Tokyo,Tokyo,Auth Publisher,True,Auth Tokyo,,,,https://example.com/auth.zip,1,,,,https://creativecommons.org/licenses/by/4.0/deed.ja,,,,,,active,,,
jp-rt,gtfs_rt,tu,JP,Tokyo,Tokyo,Realtime Publisher,True,RT,,,,https://example.com/rt.pb,0,,,,https://creativecommons.org/licenses/by/4.0/deed.ja,,,,,,active,,,
it-open,gtfs,,IT,Lazio,Rome,Rome Publisher,True,Rome,,,,https://example.com/rome.zip,0,,,https://files.mobilitydatabase.org/it-open/latest.zip,https://creativecommons.org/licenses/by-sa/4.0/,,,,,,active,,,
jp-old,gtfs,,JP,Tokyo,Tokyo,Old Publisher,True,Old,,,,https://example.com/old.zip,0,,,https://files.mobilitydatabase.org/jp-old/latest.zip,https://creativecommons.org/licenses/by/4.0/deed.ja,,,,,,deprecated,,,
`

	feeds, err := mobilityDatabaseFeedSpecs("JP", strings.NewReader(csvData))
	if err != nil {
		t.Fatal(err)
	}
	if len(feeds) != 1 {
		t.Fatalf("len(feeds) = %d, want 1: %+v", len(feeds), feeds)
	}
	feed := feeds[0]
	if feed.ID != "jp-open" {
		t.Fatalf("feed ID = %q, want jp-open", feed.ID)
	}
	if feed.SourceURL != "https://files.mobilitydatabase.org/jp-open/latest.zip" {
		t.Fatalf("SourceURL = %q", feed.SourceURL)
	}
	if feed.License != "CC-BY-4.0" {
		t.Fatalf("License = %q", feed.License)
	}
}
