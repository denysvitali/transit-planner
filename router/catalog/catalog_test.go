package catalog

import "testing"

func TestStaticCatalogIsEmpty(t *testing.T) {
	if len(Feeds) != 0 {
		t.Fatalf("static feed catalog has %d entries, want 0", len(Feeds))
	}
	if len(Networks) != 0 {
		t.Fatalf("static network catalog has %d entries, want 0", len(Networks))
	}
}
