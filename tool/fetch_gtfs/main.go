// fetch_gtfs downloads real-world GTFS feeds from no-key public endpoints.
//
// The catalog is organised by ISO 3166-1 alpha-2 country code so that callers
// can pull every feed for a country in one shot:
//
//	go run ./tool/fetch_gtfs -list                       # show known feeds
//	go run ./tool/fetch_gtfs -list -country JP           # show only Japan
//	go run ./tool/fetch_gtfs -feed toei-train            # one feed
//	go run ./tool/fetch_gtfs -country JP                 # every JP feed
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
//     https://files.mobilitydatabase.org/feeds_v2.csv) — global catalog of
//     6000+ GTFS feeds across 99+ countries, ISO-coded by country and
//     subdivision. Each Japan feed below points at the same direct download
//     URL listed in that catalog.
//   - ODPT public bucket (api-public.odpt.org) — Tokyo Metropolitan Bureau of
//     Transportation feeds (CC-BY 4.0).
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
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/denysvitali/transit-planner/router/catalog"
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
	)
	flag.Parse()

	if *list {
		listFeeds(strings.ToUpper(*country))
		return
	}

	switch {
	case *country != "" && *feedName != "":
		fail(fmt.Errorf("specify -feed or -country, not both"))
	case *country != "":
		if err := fetchCountry(strings.ToUpper(*country), *outDir); err != nil {
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

func listFeeds(country string) {
	var currentCountry string
	for _, f := range catalog.SortedFeeds() {
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
}

func fetchCountry(country, outDir string) error {
	var any bool
	for _, spec := range catalog.SortedFeeds() {
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
