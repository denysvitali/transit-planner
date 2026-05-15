// build_gtfs_db imports one or more GTFS feeds into a single SQLite database.
//
// Example:
//
//	go run ./tool/build_gtfs_db -db /tmp/gtfs.sqlite \
//	  -feed sample=assets/sample_gtfs
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/denysvitali/transit-planner/router/catalog"
	"github.com/denysvitali/transit-planner/router/gtfsdb"
)

type feedFlags []string

func (f *feedFlags) String() string {
	return strings.Join(*f, ",")
}

func (f *feedFlags) Set(value string) error {
	*f = append(*f, value)
	return nil
}

func main() {
	var feeds feedFlags
	dbPath := flag.String("db", "", "SQLite output path")
	flag.Var(&feeds, "feed", "feed import in the form <feed-id>=<gtfs-dir-or-zip> (repeatable)")
	flag.Parse()

	if *dbPath == "" || len(feeds) == 0 {
		fmt.Fprintln(os.Stderr, "usage: build_gtfs_db -db <path> -feed <feed-id>=<gtfs-dir-or-zip> [-feed ...]")
		os.Exit(2)
	}

	for _, value := range feeds {
		id, sourcePath, ok := strings.Cut(value, "=")
		if !ok || strings.TrimSpace(id) == "" || strings.TrimSpace(sourcePath) == "" {
			fmt.Fprintf(os.Stderr, "invalid -feed %q, want <feed-id>=<gtfs-dir-or-zip>\n", value)
			os.Exit(2)
		}

		meta := metadataForFeed(strings.TrimSpace(id))
		result, err := gtfsdb.ImportFeed(context.Background(), gtfsdb.ImportOptions{
			DBPath:     *dbPath,
			SourcePath: strings.TrimSpace(sourcePath),
			Feed:       meta,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "import %s: %v\n", id, err)
			os.Exit(1)
		}
		fmt.Printf("imported %s: feed_id=%d feed_version_id=%d files=%d rows=%d sha256=%s\n",
			meta.Code, result.FeedID, result.FeedVersionID, result.Files, result.Rows, result.SHA256)
	}
}

func metadataForFeed(id string) gtfsdb.FeedMetadata {
	if spec, ok := catalog.Feeds[id]; ok {
		return gtfsdb.FeedMetadata{
			Code:        spec.ID,
			Name:        spec.Name,
			CountryCode: spec.Country,
			Region:      spec.Region,
			Publisher:   spec.Publisher,
			License:     spec.License,
			SourceURL:   spec.SourceURL,
			Attribution: spec.Attribution,
		}
	}
	return gtfsdb.FeedMetadata{Code: id, Name: id}
}
