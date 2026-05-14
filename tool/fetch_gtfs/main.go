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
//	go run ./tool/fetch_gtfs -feed toei-bus -out my/dir  # custom output
//
// Downloaded zips land in assets/real_gtfs/<country>/<feed>/<feed>.zip with a
// MANIFEST.json next to each one (source URL, fetch timestamp, SHA-256).
// assets/real_gtfs/ is gitignored; commit only vendored fixtures under
// assets/sample_*.
//
// Sources we draw from — all open, no API key required:
//
//   - Mobility Database (https://mobilitydatabase.org, CSV at
//     https://files.mobilitydatabase.org/feeds_v2.csv) — global catalog used
//     by -complete for active no-key GTFS rows.
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
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/denysvitali/transit-planner/router/catalog"
)

const mobilityDatabaseFeedsURL = "https://files.mobilitydatabase.org/feeds_v2.csv"

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
	)
	flag.Parse()

	if *list {
		if err := listFeeds(strings.ToUpper(*country), *complete); err != nil {
			fail(err)
		}
		return
	}

	switch {
	case *country != "" && *feedName != "":
		fail(fmt.Errorf("specify -feed or -country, not both"))
	case *country != "":
		if err := fetchCountry(strings.ToUpper(*country), *outDir, *complete); err != nil {
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

func listFeeds(country string, complete bool) error {
	feeds := catalog.SortedFeeds()
	if complete {
		if country == "" {
			return fmt.Errorf("-complete requires -country when listing feeds")
		}
		var err error
		feeds, err = fetchMobilityDatabaseFeedSpecs(country)
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

func fetchCountry(country, outDir string, complete bool) error {
	feeds := catalog.SortedFeeds()
	if complete {
		var err error
		feeds, err = fetchMobilityDatabaseFeedSpecs(country)
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
