// fetch_gtfs downloads real-world GTFS feeds from no-key public endpoints.
//
// The catalog is organised by ISO 3166-1 alpha-2 country code so that callers
// can pull every feed for a country in one shot:
//
//	go run ./tool/fetch_gtfs -list                       # show known feeds
//	go run ./tool/fetch_gtfs -list -country JP           # show only Japan
//	go run ./tool/fetch_gtfs -feed toei-train            # one feed
//	go run ./tool/fetch_gtfs -country IT                 # curated Italian feeds
//	go run ./tool/fetch_gtfs -country JP -complete       # active no-key JP feeds from Mobility Database
//	go run ./tool/fetch_gtfs -country JP -complete -complete-source transitland
//	go run ./tool/fetch_gtfs -feed toei-bus -out my/dir  # custom output
//
// Downloaded zips land in assets/real_gtfs/<country>/<feed>/<feed>.zip with a
// MANIFEST.json next to each one (source URL, fetch timestamp, SHA-256).
// assets/real_gtfs/ is gitignored; commit only vendored fixtures under
// assets/sample_*.
//
// Sources we draw from:
//
//   - Mobility Database (https://mobilitydatabase.org, CSV at
//     https://files.mobilitydatabase.org/feeds_v2.csv) — global catalog used
//     by -complete for active no-key GTFS rows.
//   - Transitland REST API (https://transit.land/api/v2/rest) — authenticated
//     catalog used by -complete -complete-source transitland. Set
//     TRANSITLAND_API_KEY in the environment; the key is sent as a header and
//     is never written to manifests.
//   - ODPT public bucket (api-public.odpt.org) — Tokyo Metropolitan Bureau of
//     Transportation feeds (CC-BY 4.0).
//   - Official Swiss and Italian regional/city portals for the curated CH and
//     IT catalog entries.
//
// Major Japanese rail operators (JR East/West/Central, Tokyo Metro, Hankyu,
// Hanshin, Nankai, Keihan, Kintetsu, Shinkansen) only publish through ODPT's
// authenticated developer API. They are intentionally absent from this
// no-key catalog. Plan inter-city Tokyo-Osaka trips by combining the
// no-key feeds below with router.Merge, accepting that long-distance rail
// will be missing until those keys are wired in.
package main

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/denysvitali/transit-planner/router/catalog"
)

const mobilityDatabaseFeedsURL = "https://files.mobilitydatabase.org/feeds_v2.csv"
const transitlandBaseURL = "https://transit.land/api/v2/rest"
const transitlandAPIKeyEnv = "TRANSITLAND_API_KEY"

type completeSource string

const (
	completeSourceMobilityDB  completeSource = "mobilitydb"
	completeSourceTransitland completeSource = "transitland"
)

type manifest struct {
	Feed        string    `json:"feed"`
	Country     string    `json:"country"`
	Region      string    `json:"region"`
	SourceURL   string    `json:"source_url"`
	Publisher   string    `json:"publisher"`
	License     string    `json:"license"`
	Description string    `json:"description"`
	FetchedAt   time.Time `json:"fetched_at"`
	SHA256      string    `json:"sha256"`
	Bytes       int64     `json:"bytes"`
}

func main() {
	var (
		feedName = flag.String("feed", "", "feed name to fetch (see -list)")
		country  = flag.String("country", "", "ISO 3166-1 alpha-2 country code (e.g. JP) — fetch every feed for that country")
		outDir   = flag.String("out", "", "output directory (default: assets/real_gtfs/<country>/<feed>)")
		list     = flag.Bool("list", false, "list known feeds and exit")
		complete = flag.Bool("complete", false, "with -country, use Mobility Database's active no-key feed catalog for fuller country coverage")
		source   = flag.String("complete-source", string(completeSourceMobilityDB), "complete catalog source: mobilitydb or transitland")
	)
	flag.Parse()

	completeSource, err := parseCompleteSource(*source)
	if err != nil {
		fail(err)
	}

	if *list {
		if err := listFeeds(strings.ToUpper(*country), *complete, completeSource); err != nil {
			fail(err)
		}
		return
	}

	switch {
	case *country != "" && *feedName != "":
		fail(fmt.Errorf("specify -feed or -country, not both"))
	case *country != "":
		if err := fetchCountry(strings.ToUpper(*country), *outDir, *complete, completeSource); err != nil {
			fail(err)
		}
	case *feedName != "":
		spec, ok := catalog.Feeds[*feedName]
		if !ok {
			fail(fmt.Errorf("unknown feed %q; try -list", *feedName))
		}
		if _, err := fetchOne(spec, *outDir); err != nil {
			fail(err)
		}
	default:
		fail(fmt.Errorf("nothing to do: pass -feed <name>, -country <CC>, or -list"))
	}
}

func parseCompleteSource(value string) (completeSource, error) {
	source := completeSource(strings.ToLower(strings.TrimSpace(value)))
	switch source {
	case completeSourceMobilityDB, completeSourceTransitland:
		return source, nil
	default:
		return "", fmt.Errorf("unknown -complete-source %q (want mobilitydb or transitland)", value)
	}
}

func listFeeds(country string, complete bool, source completeSource) error {
	feeds := catalog.SortedFeeds()
	if complete {
		if country == "" {
			return fmt.Errorf("-complete requires -country when listing feeds")
		}
		var err error
		feeds, err = fetchCompleteFeedSpecs(country, source)
		if err != nil {
			return err
		}
	}
	var currentCountry string
	for _, f := range feeds {
		if country != "" && f.Country != country {
			continue
		}
		if f.Country != currentCountry {
			fmt.Printf("\n[%s]\n", f.Country)
			currentCountry = f.Country
		}
		fmt.Printf("  %-30s  %s, %s\n", f.ID, f.Region, f.Publisher)
		fmt.Printf("    %s  (%s)\n", f.SourceURL, f.License)
		fmt.Printf("    %s\n", f.Description)
	}
	return nil
}

func fetchCountry(country, outDir string, complete bool, source completeSource) error {
	feeds := catalog.SortedFeeds()
	if complete {
		var err error
		feeds, err = fetchCompleteFeedSpecs(country, source)
		if err != nil {
			return err
		}
	}
	var any bool
	for _, spec := range feeds {
		if spec.Country != country {
			continue
		}
		any = true
		target := outDir
		if target != "" {
			target = filepath.Join(target, normaliseFeedDir(spec.ID))
		}
		fmt.Printf("==> %s (%s, %s)\n", spec.ID, spec.Country, spec.Region)
		if _, err := fetchOne(spec, target); err != nil {
			return fmt.Errorf("%s: %w", spec.ID, err)
		}
	}
	if !any {
		return fmt.Errorf("no feeds for country %q", country)
	}
	return nil
}

func fetchCompleteFeedSpecs(country string, source completeSource) ([]catalog.FeedSpec, error) {
	switch source {
	case completeSourceMobilityDB:
		return fetchMobilityDatabaseFeedSpecs(country)
	case completeSourceTransitland:
		return fetchTransitlandFeedSpecs(country, os.Getenv(transitlandAPIKeyEnv), transitlandBaseURL)
	default:
		return nil, fmt.Errorf("unknown complete source %q", source)
	}
}

func fetchMobilityDatabaseFeedSpecs(country string) ([]catalog.FeedSpec, error) {
	req, err := http.NewRequest(http.MethodGet, mobilityDatabaseFeedsURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "transit-planner-fetch/0.1 (+github.com/denysvitali/transit-planner)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<14))
		return nil, fmt.Errorf("download %s: unexpected status %d: %s", mobilityDatabaseFeedsURL, resp.StatusCode, string(body))
	}
	return mobilityDatabaseFeedSpecs(country, resp.Body)
}

func mobilityDatabaseFeedSpecs(country string, r io.Reader) ([]catalog.FeedSpec, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	header, err := reader.Read()
	if err != nil {
		return nil, err
	}
	cols := map[string]int{}
	for i, name := range header {
		cols[name] = i
	}
	required := []string{
		"id", "data_type", "location.country_code", "location.subdivision_name",
		"location.municipality", "provider", "name", "urls.direct_download",
		"urls.authentication_type", "urls.latest", "urls.license", "status",
	}
	for _, name := range required {
		if _, ok := cols[name]; !ok {
			return nil, fmt.Errorf("Mobility Database CSV missing column %q", name)
		}
	}

	var out []catalog.FeedSpec
	seen := map[string]bool{}
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		value := func(name string) string {
			idx := cols[name]
			if idx >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[idx])
		}
		if value("data_type") != "gtfs" || value("status") != "active" || value("location.country_code") != country {
			continue
		}
		if auth := value("urls.authentication_type"); auth != "" && auth != "0" {
			continue
		}
		sourceURL := value("urls.latest")
		if sourceURL == "" {
			sourceURL = value("urls.direct_download")
		}
		if sourceURL == "" {
			continue
		}
		id := value("id")
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		region := value("location.subdivision_name")
		if region == "" {
			region = value("location.municipality")
		}
		if region == "" {
			region = "Nationwide"
		}
		publisher := value("provider")
		if publisher == "" {
			publisher = "Mobility Database"
		}
		name := value("name")
		if name == "" {
			name = publisher
		}
		license := mobilityDatabaseLicenseName(value("urls.license"))
		description := name + " GTFS feed from Mobility Database active no-key catalog."
		out = append(out, catalog.FeedSpec{
			ID: id, Name: name, Country: country, Region: region,
			Publisher: publisher, License: license, SourceURL: sourceURL,
			LocalFileName: id + ".zip", Description: description,
			Attribution: "Transit data © " + publisher + ", " + license + "; mirrored by Mobility Database.",
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Region != out[j].Region {
			return out[i].Region < out[j].Region
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

func mobilityDatabaseLicenseName(url string) string {
	switch {
	case strings.Contains(url, "creativecommons.org/licenses/by/4.0"):
		return "CC-BY-4.0"
	case strings.Contains(url, "creativecommons.org/licenses/by/2.1"):
		return "CC-BY-2.1-JP"
	case strings.Contains(url, "creativecommons.org/publicdomain/zero"):
		return "CC0-1.0"
	case strings.Contains(url, "creativecommons.org/licenses/by-sa"):
		return "CC-BY-SA"
	case url != "":
		return url
	default:
		return "Licence unspecified by Mobility Database"
	}
}

type countryBBox struct {
	minLon float64
	minLat float64
	maxLon float64
	maxLat float64
}

var transitlandCountryBBoxes = map[string]countryBBox{
	"CH": {minLon: 5.95, minLat: 45.82, maxLon: 10.49, maxLat: 47.81},
	"IT": {minLon: 6.62, minLat: 35.49, maxLon: 18.52, maxLat: 47.09},
	"JP": {minLon: 122.93, minLat: 24.04, maxLon: 153.99, maxLat: 45.56},
}

type transitlandFeedsResponse struct {
	Feeds []transitlandFeed `json:"feeds"`
	Meta  struct {
		After int `json:"after"`
	} `json:"meta"`
}

type transitlandFeed struct {
	ID        int    `json:"id"`
	OnestopID string `json:"onestop_id"`
	Name      string `json:"name"`
	FeedState struct {
		FeedVersion struct {
			Geometry transitlandGeometry `json:"geometry"`
		} `json:"feed_version"`
	} `json:"feed_state"`
	AssociatedOperators []struct {
		Name string `json:"name"`
	} `json:"associated_operators"`
	License struct {
		SPDXIdentifier     string `json:"spdx_identifier"`
		URL                string `json:"url"`
		AttributionText    string `json:"attribution_text"`
		Redistribution     string `json:"redistribution_allowed"`
		CommercialUse      string `json:"commercial_use_allowed"`
		CreateDerived      string `json:"create_derived_product"`
		ShareAlikeOptional string `json:"share_alike_optional"`
	} `json:"license"`
}

type transitlandGeometry struct {
	Type        string          `json:"type"`
	Coordinates json.RawMessage `json:"coordinates"`
}

func fetchTransitlandFeedSpecs(country, apiKey, baseURL string) ([]catalog.FeedSpec, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("%s is required for -complete-source transitland", transitlandAPIKeyEnv)
	}
	bbox, ok := transitlandCountryBBoxes[country]
	if !ok {
		return nil, fmt.Errorf("Transitland complete coverage is not configured for country %q", country)
	}

	var out []catalog.FeedSpec
	seen := map[string]bool{}
	after := 0
	for {
		req, err := transitlandFeedsRequest(baseURL, apiKey, bbox, after)
		if err != nil {
			return nil, err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		var parsed transitlandFeedsResponse
		err = decodeJSONResponse(resp, &parsed)
		if err != nil {
			return nil, err
		}
		for _, feed := range parsed.Feeds {
			if !transitlandFeedMatchesCountryBBox(feed, bbox) {
				continue
			}
			spec, ok := transitlandFeedSpec(country, baseURL, feed)
			if !ok || seen[spec.ID] {
				continue
			}
			seen[spec.ID] = true
			out = append(out, spec)
		}
		if parsed.Meta.After == 0 || parsed.Meta.After == after {
			break
		}
		after = parsed.Meta.After
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func transitlandFeedMatchesCountryBBox(feed transitlandFeed, country countryBBox) bool {
	bounds, ok := transitlandGeometryBounds(feed.FeedState.FeedVersion.Geometry)
	if !ok || !bounds.intersects(country) {
		return false
	}
	width := country.maxLon - country.minLon
	height := country.maxLat - country.minLat
	if bounds.maxLon-bounds.minLon > width*2 || bounds.maxLat-bounds.minLat > height*2 {
		return false
	}
	centerLon := (bounds.minLon + bounds.maxLon) / 2
	centerLat := (bounds.minLat + bounds.maxLat) / 2
	return centerLon >= country.minLon && centerLon <= country.maxLon &&
		centerLat >= country.minLat && centerLat <= country.maxLat
}

func transitlandGeometryBounds(geometry transitlandGeometry) (countryBBox, bool) {
	if len(geometry.Coordinates) == 0 {
		return countryBBox{}, false
	}
	var coordinates any
	if err := json.Unmarshal(geometry.Coordinates, &coordinates); err != nil {
		return countryBBox{}, false
	}
	bounds := countryBBox{minLon: 180, minLat: 90, maxLon: -180, maxLat: -90}
	var anyPoint bool
	visitCoordinatePairs(coordinates, func(lon, lat float64) {
		anyPoint = true
		if lon < bounds.minLon {
			bounds.minLon = lon
		}
		if lon > bounds.maxLon {
			bounds.maxLon = lon
		}
		if lat < bounds.minLat {
			bounds.minLat = lat
		}
		if lat > bounds.maxLat {
			bounds.maxLat = lat
		}
	})
	return bounds, anyPoint
}

func visitCoordinatePairs(value any, visit func(lon, lat float64)) {
	items, ok := value.([]any)
	if !ok {
		return
	}
	if len(items) >= 2 {
		lon, lonOK := items[0].(float64)
		lat, latOK := items[1].(float64)
		if lonOK && latOK {
			visit(lon, lat)
			return
		}
	}
	for _, item := range items {
		visitCoordinatePairs(item, visit)
	}
}

func (bbox countryBBox) intersects(other countryBBox) bool {
	return bbox.minLon <= other.maxLon && bbox.maxLon >= other.minLon &&
		bbox.minLat <= other.maxLat && bbox.maxLat >= other.minLat
}

func transitlandFeedsRequest(baseURL, apiKey string, bbox countryBBox, after int) (*http.Request, error) {
	endpoint, err := url.Parse(strings.TrimRight(baseURL, "/") + "/feeds")
	if err != nil {
		return nil, err
	}
	q := endpoint.Query()
	q.Set("spec", "gtfs")
	q.Set("fetch_error", "false")
	q.Set("limit", "100")
	q.Set("bbox", fmt.Sprintf("%g,%g,%g,%g", bbox.minLon, bbox.minLat, bbox.maxLon, bbox.maxLat))
	q.Set("license_redistribution_allowed", "exclude_no")
	q.Set("license_create_derived_product", "exclude_no")
	q.Set("license_commercial_use_allowed", "exclude_no")
	if after > 0 {
		q.Set("after", strconv.Itoa(after))
	}
	endpoint.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "transit-planner-fetch/0.1 (+github.com/denysvitali/transit-planner)")
	req.Header.Set("apikey", apiKey)
	return req, nil
}

func decodeJSONResponse(resp *http.Response, target any) error {
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<14))
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

func transitlandFeedSpec(country, baseURL string, feed transitlandFeed) (catalog.FeedSpec, bool) {
	key := feed.OnestopID
	if key == "" && feed.ID > 0 {
		key = strconv.Itoa(feed.ID)
	}
	if key == "" {
		return catalog.FeedSpec{}, false
	}
	idSuffix := key
	if feed.OnestopID == "" {
		idSuffix = "id-" + idSuffix
	}
	id := "transitland-" + sanitizeFeedID(idSuffix)
	publisher := transitlandPublisher(feed)
	name := strings.TrimSpace(feed.Name)
	if name == "" {
		name = publisher
	}
	license := transitlandLicenseName(feed)
	attribution := strings.TrimSpace(feed.License.AttributionText)
	if attribution == "" {
		attribution = "Transit data © " + publisher + ", " + license + "; discovered through Transitland."
	}
	downloadURL := strings.TrimRight(baseURL, "/") + "/feeds/" + url.PathEscape(key) + "/download_latest_feed_version"
	return catalog.FeedSpec{
		ID: id, Name: name, Country: country, Region: "Transitland " + country,
		Publisher: publisher, License: license, SourceURL: downloadURL,
		LocalFileName: id + ".zip",
		Description:   name + " GTFS feed discovered through Transitland.",
		Attribution:   attribution,
	}, true
}

func transitlandPublisher(feed transitlandFeed) string {
	for _, operator := range feed.AssociatedOperators {
		if name := strings.TrimSpace(operator.Name); name != "" {
			return name
		}
	}
	if name := strings.TrimSpace(feed.Name); name != "" {
		return name
	}
	return "Transitland"
}

func transitlandLicenseName(feed transitlandFeed) string {
	switch {
	case feed.License.SPDXIdentifier != "":
		return feed.License.SPDXIdentifier
	case feed.License.URL != "":
		return feed.License.URL
	default:
		return "Transitland license metadata"
	}
}

func sanitizeFeedID(value string) string {
	value = strings.ToLower(value)
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	out = strings.Join(strings.FieldsFunc(out, func(r rune) bool { return r == '-' }), "-")
	if out == "" {
		return "feed"
	}
	return out
}

func fetchOne(spec catalog.FeedSpec, outDir string) (string, error) {
	dir := outDir
	if dir == "" {
		dir = filepath.Join("assets", "real_gtfs", strings.ToLower(spec.Country), normaliseFeedDir(spec.ID))
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	zipPath := filepath.Join(dir, spec.ID+".zip")
	if err := download(spec.SourceURL, zipPath); err != nil {
		return "", fmt.Errorf("download %s: %w", spec.SourceURL, err)
	}

	sum, size, err := hashFile(zipPath)
	if err != nil {
		return "", err
	}

	m := manifest{
		Feed:        spec.ID,
		Country:     spec.Country,
		Region:      spec.Region,
		SourceURL:   spec.SourceURL,
		Publisher:   spec.Publisher,
		License:     spec.License,
		Description: spec.Description,
		FetchedAt:   time.Now().UTC(),
		SHA256:      sum,
		Bytes:       size,
	}
	if err := writeManifest(filepath.Join(dir, "MANIFEST.json"), m); err != nil {
		return "", err
	}

	fmt.Printf("wrote %s (%d bytes, sha256=%s)\n", zipPath, size, sum)
	return zipPath, nil
}

func download(url, dst string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "transit-planner-fetch/0.1 (+github.com/denysvitali/transit-planner)")
	if req.URL.Host == "transit.land" {
		if apiKey := strings.TrimSpace(os.Getenv(transitlandAPIKeyEnv)); apiKey != "" {
			req.Header.Set("apikey", apiKey)
		} else {
			return fmt.Errorf("%s is required to download Transitland feeds", transitlandAPIKeyEnv)
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<14))
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	tmp := dst + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dst)
}

func hashFile(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

func writeManifest(path string, m manifest) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(m)
}

func normaliseFeedDir(name string) string {
	return strings.ReplaceAll(name, "-", "_")
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
