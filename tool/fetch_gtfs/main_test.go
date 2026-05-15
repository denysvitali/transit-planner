package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
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

func TestTransitlandLiveFeedsWhenAPIKeyPresent(t *testing.T) {
	apiKey := os.Getenv(transitlandAPIKeyEnv)
	if apiKey == "" {
		t.Skipf("%s is not set", transitlandAPIKeyEnv)
	}
	req, err := transitlandFeedsRequest(
		transitlandBaseURL,
		apiKey,
		transitlandCountryBBoxes["CH"],
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var parsed transitlandFeedsResponse
	if err := decodeJSONResponse(resp, &parsed); err != nil {
		t.Fatal(err)
	}
	if len(parsed.Feeds) == 0 {
		t.Fatal("Transitland returned no Swiss GTFS feeds")
	}
	if _, ok := transitlandFeedSpec("CH", transitlandBaseURL, parsed.Feeds[0]); !ok {
		t.Fatal("Transitland feed could not be converted to a feed spec")
	}
}

func TestTransitlandFeedSpecBuildsAuthenticatedDownloadURLWithoutKey(t *testing.T) {
	spec, ok := transitlandFeedSpec("JP", "https://transit.land/api/v2/rest", transitlandFeed{
		ID:        123,
		OnestopID: "f-test~feed",
		Name:      "Example Feed",
		AssociatedOperators: []struct {
			Name string `json:"name"`
		}{
			{Name: "Example Operator"},
		},
		License: struct {
			SPDXIdentifier     string `json:"spdx_identifier"`
			URL                string `json:"url"`
			AttributionText    string `json:"attribution_text"`
			Redistribution     string `json:"redistribution_allowed"`
			CommercialUse      string `json:"commercial_use_allowed"`
			CreateDerived      string `json:"create_derived_product"`
			ShareAlikeOptional string `json:"share_alike_optional"`
		}{
			SPDXIdentifier:  "CC-BY-4.0",
			AttributionText: "Use Example attribution.",
		},
	})
	if !ok {
		t.Fatal("transitlandFeedSpec returned false")
	}
	if spec.ID != "transitland-f-test-feed" {
		t.Fatalf("ID = %q", spec.ID)
	}
	if spec.SourceURL != "https://transit.land/api/v2/rest/feeds/f-test~feed/download_latest_feed_version" {
		t.Fatalf("SourceURL = %q", spec.SourceURL)
	}
	if strings.Contains(spec.SourceURL, "apikey") {
		t.Fatalf("SourceURL must not contain API key material: %q", spec.SourceURL)
	}
	if spec.Attribution != "Use Example attribution." {
		t.Fatalf("Attribution = %q", spec.Attribution)
	}
}

func TestTransitlandFeedsRequestUsesHeaderAuthAndLicenseFilters(t *testing.T) {
	req, err := transitlandFeedsRequest(
		"https://transit.land/api/v2/rest",
		"test-key",
		countryBBox{minLon: 1, minLat: 2, maxLon: 3, maxLat: 4},
		42,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := req.Header.Get("apikey"); got != "test-key" {
		t.Fatalf("apikey header = %q", got)
	}
	if strings.Contains(req.URL.String(), "test-key") {
		t.Fatalf("request URL must not contain API key material: %q", req.URL.String())
	}
	values, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{
		"spec":                           "gtfs",
		"fetch_error":                    "false",
		"bbox":                           "1,2,3,4",
		"after":                          "42",
		"license_redistribution_allowed": "exclude_no",
		"license_create_derived_product": "exclude_no",
		"license_commercial_use_allowed": "exclude_no",
	} {
		if got := values.Get(key); got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestTransitlandFeedMatchesCountryBBoxRejectsOversizedFeeds(t *testing.T) {
	ch := transitlandCountryBBoxes["CH"]
	local := transitlandFeedWithPolygon([][]float64{
		{7.0, 46.0},
		{8.0, 46.0},
		{8.0, 47.0},
		{7.0, 47.0},
		{7.0, 46.0},
	})
	europeWide := transitlandFeedWithPolygon([][]float64{
		{-9.0, 38.0},
		{23.0, 38.0},
		{23.0, 54.0},
		{-9.0, 54.0},
		{-9.0, 38.0},
	})
	outside := transitlandFeedWithPolygon([][]float64{
		{11.0, 45.7},
		{12.0, 45.7},
		{12.0, 46.4},
		{11.0, 46.4},
		{11.0, 45.7},
	})

	if !transitlandFeedMatchesCountryBBox(local, ch) {
		t.Fatal("expected local Swiss-like feed to match CH bbox")
	}
	if transitlandFeedMatchesCountryBBox(europeWide, ch) {
		t.Fatal("expected oversized European feed to be rejected")
	}
	if transitlandFeedMatchesCountryBBox(outside, ch) {
		t.Fatal("expected feed centered outside CH bbox to be rejected")
	}
}

func transitlandFeedWithPolygon(points [][]float64) transitlandFeed {
	coordinates, err := json.Marshal([][][]float64{points})
	if err != nil {
		panic(err)
	}
	var feed transitlandFeed
	feed.FeedState.FeedVersion.Geometry = transitlandGeometry{
		Type:        "Polygon",
		Coordinates: coordinates,
	}
	return feed
}
