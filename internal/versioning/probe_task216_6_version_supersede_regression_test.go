package versioning_test

import (
	"path/filepath"
	"testing"

	"task216-aftershock/internal/catalog"
	"task216-aftershock/internal/model"
	"task216-aftershock/internal/store"
	"task216-aftershock/internal/versioning"
)

func TestLatestPublishedVersionSupersedesPrevious(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "versions.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	c, err := catalog.NewManager(db).Create("catalog", "region")
	if err != nil {
		t.Fatal(err)
	}
	m := versioning.NewManager(db)
	v1, err := m.CreateDraft(c.ID, "one")
	if err != nil {
		t.Fatal(err)
	}
	if len(v1.SnapshotHash) != 64 {
		t.Fatalf("snapshot hash length = %d, want 64", len(v1.SnapshotHash))
	}
	if _, err := m.Publish(v1.ID); err != nil {
		t.Fatal(err)
	}
	v2, err := m.CreateDraft(c.ID, "two")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Publish(v2.ID); err != nil {
		t.Fatal(err)
	}
	old, err := m.Get(v1.ID)
	if err != nil {
		t.Fatal(err)
	}
	active, err := m.ActivePublished(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if old.Status != model.VersionSuperseded || active.ID != v2.ID || active.Status != model.VersionPublished {
		t.Fatalf("version lifecycle mismatch: old=%+v active=%+v", old, active)
	}
}
