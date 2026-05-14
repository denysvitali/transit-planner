// fetch_gtfs downloads real-world GTFS feeds from no-key public endpoints.
//
// Currently wired to the Tokyo Public Transportation Open Data Center (ODPT),
// which exposes a public-bucket mirror of selected feeds under
// api-public.odpt.org — no API key, no registration. Feeds are CC-BY 4.0;
// attribution is required (see LICENSES_THIRD_PARTY.md).
//
// Usage:
//
//	go run ./tool/fetch_gtfs                       # default: toei-bus
//	go run ./tool/fetch_gtfs -feed toei-train
//	go run ./tool/fetch_gtfs -feed kanazawa-flatbus
//	go run ./tool/fetch_gtfs -feed toei-bus -out assets/real_gtfs/toei_bus
//	go run ./tool/fetch_gtfs -feed kanazawa-flatbus -out assets/real_gtfs/kanazawa_flatbus
//
// The downloaded zip is written to <out>/<feed>.zip and a MANIFEST.json with
// the source URL, fetch timestamp, and SHA-256 sits alongside it for
// reproducibility. The default output directory is under assets/real_gtfs/,
// which is gitignored; commit a vendored snapshot only under assets/sample_*.
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
)

type feedSpec struct {
	name        string
	url         string
	publisher   string
	license     string
	description string
}

var feeds = map[string]feedSpec{
	"toei-bus": {
		name:        "toei-bus",
		url:         "https://api-public.odpt.org/api/v4/files/Toei/data/ToeiBus-GTFS.zip",
		publisher:   "Tokyo Metropolitan Bureau of Transportation (東京都交通局)",
		license:     "CC-BY-4.0",
		description: "Toei municipal bus network (Tokyo), via ODPT public bucket",
	},
	"toei-train": {
		name:        "toei-train",
		url:         "https://api-public.odpt.org/api/v4/files/Toei/data/Toei-Train-GTFS.zip",
		publisher:   "Tokyo Metropolitan Bureau of Transportation (東京都交通局)",
		license:     "CC-BY-4.0",
		description: "Toei subway lines (浅草線, 三田線, 新宿線, 大江戸線, 日暮里舎人, 都電荒川), via ODPT public bucket",
	},
	"kanazawa-flatbus": {
		name:        "kanazawa-flatbus",
		url:         "https://catalog-data.city.kanazawa.ishikawa.jp/dataset/1196beb4-f9f9-463c-9723-5b38d8127425/resource/9636cac5-1449-4656-893b-ec98d834eb23/download/flatbus20260401.zip",
		publisher:   "Kanazawa City, Ishikawa",
		license:     "CC-BY-4.0",
		description: "Kanazawa municipal bus network GTFS, via Kanazawa Open Data Catalog",
	},
	"kanazawa-hakusan-meguru": {
		name:        "kanazawa-hakusan-meguru",
		url:         "https://catalog-data.city.kanazawa.ishikawa.jp/dataset/89d93f28-38b4-4971-9988-2ff2d3227f56/resource/50049b19-fe9f-4ca1-9ea9-9d0a24141644/download/172103_bus.zip",
		publisher:   "Hakusan City, Ishikawa",
		license:     "CC-BY-4.0",
		description: "Hakusan city bus network GTFS, via Kanazawa Open Data Catalog",
	},
	"kanazawa-tsubata-bus": {
		name:        "kanazawa-tsubata-bus",
		url:         "https://catalog-data.city.kanazawa.ishikawa.jp/dataset/8cd7f0dc-aab0-4bf4-a09d-c1d79faf4512/resource/9565f9b7-3bf7-4937-bee5-789d2aa4bf8a/download/gtfs-jp_tsubata.zip",
		publisher:   "Tsubata Town, Ishikawa",
		license:     "CC-BY-4.0",
		description: "Tsubata bus GTFS (GTFS-JP), via Kanazawa Open Data Catalog",
	},
}

type manifest struct {
	Feed        string    `json:"feed"`
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
		feedName = flag.String("feed", "toei-bus", "feed to fetch (toei-bus, toei-train, kanazawa-flatbus, kanazawa-hakusan-meguru, kanazawa-tsubata-bus)")
		outDir   = flag.String("out", "", "output directory (default: assets/real_gtfs/<feed>)")
		list     = flag.Bool("list", false, "list known feeds and exit")
	)
	flag.Parse()

	if *list {
		for _, key := range []string{
			"toei-bus",
			"toei-train",
			"kanazawa-flatbus",
			"kanazawa-hakusan-meguru",
			"kanazawa-tsubata-bus",
		} {
			f := feeds[key]
			fmt.Printf("%s  %s  (%s, %s)\n    %s\n", f.name, f.url, f.license, f.publisher, f.description)
		}
		return
	}

	spec, ok := feeds[*feedName]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown feed %q; try -list\n", *feedName)
		os.Exit(2)
	}

	dir := *outDir
	if dir == "" {
		dir = filepath.Join("assets", "real_gtfs", normaliseFeedDir(spec.name))
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fail(err)
	}

	zipPath := filepath.Join(dir, spec.name+".zip")
	if err := download(spec.url, zipPath); err != nil {
		fail(fmt.Errorf("download %s: %w", spec.url, err))
	}

	sum, size, err := hashFile(zipPath)
	if err != nil {
		fail(err)
	}

	m := manifest{
		Feed:        spec.name,
		SourceURL:   spec.url,
		Publisher:   spec.publisher,
		License:     spec.license,
		Description: spec.description,
		FetchedAt:   time.Now().UTC(),
		SHA256:      sum,
		Bytes:       size,
	}
	if err := writeManifest(filepath.Join(dir, "MANIFEST.json"), m); err != nil {
		fail(err)
	}

	fmt.Printf("wrote %s (%d bytes, sha256=%s)\n", zipPath, size, sum)
	fmt.Printf("manifest: %s\n", filepath.Join(dir, "MANIFEST.json"))
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
