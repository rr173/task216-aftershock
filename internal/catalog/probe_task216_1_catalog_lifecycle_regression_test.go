package catalog_test

import (
	"path/filepath"
	"testing"
	"time"

	"task216-aftershock/internal/catalog"
	"task216-aftershock/internal/event"
	"task216-aftershock/internal/model"
	"task216-aftershock/internal/store"
)

func TestPrematureCatalogPublishRejected(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "catalog.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	m := catalog.NewManager(db)
	c, err := m.Create("catalog", "region")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := m.IngestEvent(c.ID, event.EventInput{OriginTime: timeValue(), Latitude: 1, Longitude: 1, Magnitude: 4}); err != nil {
		t.Fatal(err)
	}
	current, err := m.Get(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Status != model.CatalogImporting {
		t.Fatalf("ingest advanced catalog to %q", current.Status)
	}
	if err := m.Publish(c.ID); err == nil {
		t.Fatal("publishing an importing catalog should be rejected")
	}
}

func timeValue() time.Time { return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC) }
