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
	"sort"
	"strings"
	"time"
)

type feedSpec struct {
	name        string
	country     string // ISO 3166-1 alpha-2
	region      string // ISO 3166-2 subdivision or free-form area
	url         string
	publisher   string
	license     string
	description string
}

// feeds is the catalog of no-key GTFS endpoints. Keep entries sorted by
// country then name for stable -list output.
var feeds = map[string]feedSpec{
	// Japan — Tokyo (ODPT public bucket)
	"toei-bus": {
		name:        "toei-bus",
		country:     "JP",
		region:      "Tokyo",
		url:         "https://api-public.odpt.org/api/v4/files/Toei/data/ToeiBus-GTFS.zip",
		publisher:   "Tokyo Metropolitan Bureau of Transportation (東京都交通局)",
		license:     "CC-BY-4.0",
		description: "Toei municipal bus network (Tokyo), via ODPT public bucket",
	},
	"toei-train": {
		name:        "toei-train",
		country:     "JP",
		region:      "Tokyo",
		url:         "https://api-public.odpt.org/api/v4/files/Toei/data/Toei-Train-GTFS.zip",
		publisher:   "Tokyo Metropolitan Bureau of Transportation (東京都交通局)",
		license:     "CC-BY-4.0",
		description: "Toei subway lines (浅草線, 三田線, 新宿線, 大江戸線, 日暮里舎人, 都電荒川), via ODPT public bucket",
	},
	// Japan — Ishikawa (Kanazawa Open Data Catalog)
	"kanazawa-flatbus": {
		name:        "kanazawa-flatbus",
		country:     "JP",
		region:      "Ishikawa",
		url:         "https://catalog-data.city.kanazawa.ishikawa.jp/dataset/1196beb4-f9f9-463c-9723-5b38d8127425/resource/9636cac5-1449-4656-893b-ec98d834eb23/download/flatbus20260401.zip",
		publisher:   "Kanazawa City, Ishikawa",
		license:     "CC-BY-4.0",
		description: "Kanazawa municipal bus network, via Kanazawa Open Data Catalog",
	},
	"kanazawa-hakusan-meguru": {
		name:        "kanazawa-hakusan-meguru",
		country:     "JP",
		region:      "Ishikawa",
		url:         "https://catalog-data.city.kanazawa.ishikawa.jp/dataset/89d93f28-38b4-4971-9988-2ff2d3227f56/resource/50049b19-fe9f-4ca1-9ea9-9d0a24141644/download/172103_bus.zip",
		publisher:   "Hakusan City, Ishikawa",
		license:     "CC-BY-4.0",
		description: "Hakusan city bus network, via Kanazawa Open Data Catalog",
	},
	"kanazawa-tsubata-bus": {
		name:        "kanazawa-tsubata-bus",
		country:     "JP",
		region:      "Ishikawa",
		url:         "https://catalog-data.city.kanazawa.ishikawa.jp/dataset/8cd7f0dc-aab0-4bf4-a09d-c1d79faf4512/resource/9565f9b7-3bf7-4937-bee5-789d2aa4bf8a/download/gtfs-jp_tsubata.zip",
		publisher:   "Tsubata Town, Ishikawa",
		license:     "CC-BY-4.0",
		description: "Tsubata bus network (GTFS-JP), via Kanazawa Open Data Catalog",
	},
	// Japan — Kansai (Mobility Database / gtfs-data.jp)
	"kobe-shiokaze": {
		name:        "kobe-shiokaze",
		country:     "JP",
		region:      "Hyogo",
		url:         "https://api.gtfs-data.jp/v2/organizations/kobecity/feeds/kobe-shiokaze/files/feed.zip?rid=current",
		publisher:   "Kobe City (神戸市)",
		license:     "CC-BY-2.1-JP",
		description: "Kobe Shiokaze community bus, via gtfs-data.jp",
	},
	"kobe-satoyama": {
		name:        "kobe-satoyama",
		country:     "JP",
		region:      "Hyogo",
		url:         "https://api.gtfs-data.jp/v2/organizations/kobecity/feeds/kobe-satoyama/files/feed.zip?rid=current",
		publisher:   "Kobe City (神戸市)",
		license:     "CC-BY-4.0",
		description: "Kobe Satoyama community bus, via gtfs-data.jp",
	},
	"himeji-ieshima": {
		name:        "himeji-ieshima",
		country:     "JP",
		region:      "Hyogo",
		url:         "https://api.gtfs-data.jp/v2/organizations/himejicity/feeds/ieshima-boze-yukihiko/files/feed.zip?rid=current",
		publisher:   "Himeji City (姫路市)",
		license:     "CC-BY-2.1-JP",
		description: "Himeji Ieshima / Boze / Yukihiko routes, via gtfs-data.jp",
	},
	"takarazuka-runrunbus": {
		name:        "takarazuka-runrunbus",
		country:     "JP",
		region:      "Hyogo",
		url:         "https://api.gtfs-data.jp/v2/organizations/takarazukacity/feeds/runrunbus/files/feed.zip?rid=current",
		publisher:   "Takarazuka City (宝塚市)",
		license:     "CC-BY-2.1-JP",
		description: "Takarazuka runrun community bus, via gtfs-data.jp",
	},
	"nishinomiya-sakurayamanami": {
		name:        "nishinomiya-sakurayamanami",
		country:     "JP",
		region:      "Hyogo",
		url:         "https://api.gtfs-data.jp/v2/organizations/nishinomiyacity/feeds/sakurayamanami/files/feed.zip?rid=current",
		publisher:   "Nishinomiya City (西宮市)",
		license:     "CC-BY-2.1-JP",
		description: "Nishinomiya Sakurayamanami community bus, via gtfs-data.jp",
	},
	"yamatokoriyama-kingyobus": {
		name:        "yamatokoriyama-kingyobus",
		country:     "JP",
		region:      "Nara",
		url:         "https://api.gtfs-data.jp/v2/organizations/yamatokoriyamacity/feeds/kingyobus/files/feed.zip?rid=current",
		publisher:   "Yamatokoriyama City (大和郡山市)",
		license:     "CC-BY-4.0",
		description: "Yamatokoriyama Kingyo community bus, via gtfs-data.jp",
	},
	"rinkan-koyasan": {
		name:        "rinkan-koyasan",
		country:     "JP",
		region:      "Wakayama",
		url:         "https://api.gtfs-data.jp/v2/organizations/rinkan/feeds/koyasan/files/feed.zip?rid=current",
		publisher:   "Nankai Rinkan Bus (南海りんかんバス)",
		license:     "CC-BY-4.0",
		description: "Mt. Koya / Koyasan bus network, via gtfs-data.jp",
	},
}

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
		spec, ok := feeds[*feedName]
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
	keys := sortedFeedKeys()
	var currentCountry string
	for _, key := range keys {
		f := feeds[key]
		if country != "" && f.country != country {
			continue
		}
		if f.country != currentCountry {
			fmt.Printf("\n[%s]\n", f.country)
			currentCountry = f.country
		}
		fmt.Printf("  %-30s  %s, %s\n", f.name, f.region, f.publisher)
		fmt.Printf("    %s  (%s)\n", f.url, f.license)
		fmt.Printf("    %s\n", f.description)
	}
}

func fetchCountry(country, outDir string) error {
	keys := sortedFeedKeys()
	var any bool
	for _, key := range keys {
		spec := feeds[key]
		if spec.country != country {
			continue
		}
		any = true
		target := outDir
		if target != "" {
			target = filepath.Join(target, normaliseFeedDir(spec.name))
		}
		fmt.Printf("==> %s (%s, %s)\n", spec.name, spec.country, spec.region)
		if _, err := fetchOne(spec, target); err != nil {
			return fmt.Errorf("%s: %w", spec.name, err)
		}
	}
	if !any {
		return fmt.Errorf("no feeds for country %q", country)
	}
	return nil
}

func fetchOne(spec feedSpec, outDir string) (string, error) {
	dir := outDir
	if dir == "" {
		dir = filepath.Join("assets", "real_gtfs", strings.ToLower(spec.country), normaliseFeedDir(spec.name))
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	zipPath := filepath.Join(dir, spec.name+".zip")
	if err := download(spec.url, zipPath); err != nil {
		return "", fmt.Errorf("download %s: %w", spec.url, err)
	}

	sum, size, err := hashFile(zipPath)
	if err != nil {
		return "", err
	}

	m := manifest{
		Feed:        spec.name,
		Country:     spec.country,
		Region:      spec.region,
		SourceURL:   spec.url,
		Publisher:   spec.publisher,
		License:     spec.license,
		Description: spec.description,
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

func sortedFeedKeys() []string {
	keys := make([]string, 0, len(feeds))
	for k := range feeds {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if feeds[keys[i]].country != feeds[keys[j]].country {
			return feeds[keys[i]].country < feeds[keys[j]].country
		}
		return keys[i] < keys[j]
	})
	return keys
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
