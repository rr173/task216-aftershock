package versioning_test

import (
	"path/filepath"
	"testing"

	"task216-aftershock/internal/catalog"
	"task216-aftershock/internal/model"
	"task216-aftershock/internal/store"
	"task216-aftershock/internal/versioning"
)

func TestPublishedVersionSurvivesReopenAndSupersedesPrevious(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "versions.db")
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	c, err := catalog.NewManager(db).Create("versioned-catalog", "region-a")
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	manager := versioning.NewManager(db)
	v1, err := manager.CreateDraft(c.ID, "release-1")
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	if v1.Number != 1 || v1.SnapshotHash == "" {
		db.Close()
		t.Fatalf("unexpected first draft: %+v", v1)
	}
	if _, err := manager.Publish(v1.ID); err != nil {
		db.Close()
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	db, err = store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	manager = versioning.NewManager(db)
	recovered, err := manager.Get(v1.ID)
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	if recovered.Status != model.VersionPublished {
		db.Close()
		t.Fatalf("published status was not recovered: %+v", recovered)
	}

	v2, err := manager.CreateDraft(c.ID, "release-2")
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	if _, err := manager.Publish(v2.ID); err != nil {
		db.Close()
		t.Fatal(err)
	}
	old, err := manager.Get(v1.ID)
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	active, err := manager.ActivePublished(c.ID)
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	if old.Status != model.VersionSuperseded || active.ID != v2.ID || active.Status != model.VersionPublished {
		db.Close()
		t.Fatalf("version replacement invariant failed: old=%+v active=%+v", old, active)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
}
