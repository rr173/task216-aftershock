package review_test

import (
	"path/filepath"
	"testing"
	"time"

	"task216-aftershock/internal/model"
	"task216-aftershock/internal/review"
	"task216-aftershock/internal/store"
)

func TestMergeKeepsTheLargerMainshock(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "merge.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	c := &model.Catalog{Name: "catalog", Status: model.CatalogAnalyzing}
	catalogID, err := db.InsertCatalog(c)
	if err != nil {
		t.Fatal(err)
	}
	aEvent := &model.Event{CatalogID: catalogID, OriginTime: time.Now().UTC(), Latitude: 36, Longitude: 140, Magnitude: 4.0, Fingerprint: "a", Status: model.EventValid}
	bEvent := &model.Event{CatalogID: catalogID, OriginTime: time.Now().UTC(), Latitude: 36.1, Longitude: 140.1, Magnitude: 6.0, Fingerprint: "b", Status: model.EventValid}
	if _, err := db.InsertEvent(aEvent); err != nil {
		t.Fatal(err)
	}
	if _, err := db.InsertEvent(bEvent); err != nil {
		t.Fatal(err)
	}
	a := &model.Cluster{CatalogID: catalogID, MainshockID: aEvent.ID, Status: model.ClusterCandidate}
	b := &model.Cluster{CatalogID: catalogID, MainshockID: bEvent.ID, Status: model.ClusterCandidate}
	if _, err := db.InsertCluster(a); err != nil {
		t.Fatal(err)
	}
	if _, err := db.InsertCluster(b); err != nil {
		t.Fatal(err)
	}
	if err := db.AddMembership(a.ID, aEvent.ID, model.RoleMainshock); err != nil {
		t.Fatal(err)
	}
	if err := db.AddMembership(b.ID, bEvent.ID, model.RoleMainshock); err != nil {
		t.Fatal(err)
	}
	merged, err := review.NewManager(db).MergeClusters(a.ID, b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if merged.MainshockID != bEvent.ID {
		t.Fatalf("merged mainshock = %d, want %d", merged.MainshockID, bEvent.ID)
	}
}
