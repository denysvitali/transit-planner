package gtfsdb

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestImportFeedStoresMetadataAndFeedScopedTables(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "gtfs.sqlite")

	result, err := ImportFeed(context.Background(), ImportOptions{
		DBPath:     dbPath,
		SourcePath: filepath.Join("..", "..", "assets", "sample_gtfs"),
		Feed: FeedMetadata{
			Code:        "sample-bern",
			Name:        "Sample Bern",
			CountryCode: "CH",
			Publisher:   "Transit Planner tests",
			License:     "test fixture",
			SourceURL:   "https://example.invalid/sample_gtfs.zip",
			Attribution: "Transit Planner sample fixture.",
		},
	})
	if err != nil {
		t.Fatalf("ImportFeed: %v", err)
	}
	if result.FeedID == 0 {
		t.Fatal("FeedID is zero")
	}
	if result.FeedVersionID == 0 {
		t.Fatal("FeedVersionID is zero")
	}
	if result.Files < 7 {
		t.Fatalf("Files = %d, want at least fixture GTFS files", result.Files)
	}
	if result.Rows == 0 {
		t.Fatal("Rows is zero")
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	assertScalar(t, db, `select count(*) from feeds where code = 'sample-bern'`, 1)
	assertScalar(t, db, `select count(*) from feed_versions where feed_id = ? and active = 1`, 1, result.FeedID)
	assertScalar(t, db, `select count(*) from stops where feed_id = ?`, 10, result.FeedID)
	assertScalar(t, db, `select count(*) from routes where feed_id = ?`, 3, result.FeedID)
	assertScalar(t, db, `select count(*) from trips where feed_id = ?`, 9, result.FeedID)
	assertScalar(t, db, `select count(*) from gtfs_files where feed_id = ? and filename = 'stops.txt'`, 1, result.FeedID)
	assertScalar(t, db, `select count(*) from gtfs_rows where feed_id = ? and filename = 'stops.txt'`, 10, result.FeedID)

	var attribution string
	if err := db.QueryRow(`select attribution_text from feeds where id = ?`, result.FeedID).Scan(&attribution); err != nil {
		t.Fatal(err)
	}
	if attribution != "Transit Planner sample fixture." {
		t.Fatalf("attribution = %q", attribution)
	}
}

func TestImportFeedKeepsMultipleFeedsSeparate(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "gtfs.sqlite")
	sourcePath := filepath.Join("..", "..", "assets", "sample_gtfs")

	first, err := ImportFeed(context.Background(), ImportOptions{
		DBPath:     dbPath,
		SourcePath: sourcePath,
		Feed:       FeedMetadata{Code: "first", Name: "First"},
	})
	if err != nil {
		t.Fatalf("ImportFeed first: %v", err)
	}
	second, err := ImportFeed(context.Background(), ImportOptions{
		DBPath:     dbPath,
		SourcePath: sourcePath,
		Feed:       FeedMetadata{Code: "second", Name: "Second"},
	})
	if err != nil {
		t.Fatalf("ImportFeed second: %v", err)
	}
	if first.FeedID == second.FeedID {
		t.Fatal("separate feed codes reused the same feed_id")
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	assertScalar(t, db, `select count(*) from feeds`, 2)
	assertScalar(t, db, `select count(*) from stops`, 20)
	assertScalar(t, db, `select count(distinct feed_id) from stops`, 2)
}

func TestImportFeedKeepsOnlyLatestVersionActive(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "gtfs.sqlite")
	sourcePath := filepath.Join("..", "..", "assets", "sample_gtfs")

	first, err := ImportFeed(context.Background(), ImportOptions{
		DBPath:     dbPath,
		SourcePath: sourcePath,
		Feed:       FeedMetadata{Code: "sample", Name: "Sample"},
	})
	if err != nil {
		t.Fatalf("ImportFeed first: %v", err)
	}
	second, err := ImportFeed(context.Background(), ImportOptions{
		DBPath:     dbPath,
		SourcePath: sourcePath,
		Feed:       FeedMetadata{Code: "sample", Name: "Sample"},
	})
	if err != nil {
		t.Fatalf("ImportFeed second: %v", err)
	}
	if first.FeedID != second.FeedID {
		t.Fatal("same feed code did not reuse feed_id")
	}
	if first.FeedVersionID == second.FeedVersionID {
		t.Fatal("new import did not create a new feed_version_id")
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	assertScalar(t, db, `select count(*) from feed_versions where feed_id = ?`, 2, first.FeedID)
	assertScalar(t, db, `select count(*) from feed_versions where feed_id = ? and active = 1`, 1, first.FeedID)
	assertScalar(t, db, `select count(*) from stops where feed_id = ?`, 20, first.FeedID)
	assertScalar(t, db, `select count(*) from active_stops where feed_id = ?`, 10, first.FeedID)
}

func assertScalar(t *testing.T, db *sql.DB, query string, want int, args ...any) {
	t.Helper()
	var got int
	if err := db.QueryRow(query, args...).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("%s = %d, want %d", query, got, want)
	}
}
