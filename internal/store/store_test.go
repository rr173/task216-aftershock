package store

import (
	"path/filepath"
	"testing"
	"time"

	"task216-aftershock/internal/model"
)

func TestOpenAndMigrate(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
}

func TestCatalogAndEventPersistence(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	c := &model.Catalog{Name: "test", Region: "r", Status: model.CatalogImporting}
	id, err := db.InsertCatalog(c)
	if err != nil {
		t.Fatal(err)
	}

	e := &model.Event{
		CatalogID: id, OriginTime: time.Now().UTC(), Latitude: 36, Longitude: 140,
		DepthKm: 12, Magnitude: 5.5, Fingerprint: "fp1", Status: model.EventValid,
	}
	if _, err := db.InsertEvent(e); err != nil {
		t.Fatal(err)
	}

	// 重复指纹应返回 ErrDuplicate。
	if _, err := db.InsertEvent(e); err != model.ErrDuplicate {
		t.Fatalf("expected ErrDuplicate, got %v", err)
	}

	got, err := db.GetCatalog(id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "test" {
		t.Fatalf("unexpected catalog name %q", got.Name)
	}
}

func TestClusterAndVersionLifecycle(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	c := &model.Catalog{Name: "c", Status: model.CatalogAnalyzing}
	cid, _ := db.InsertCatalog(c)

	cl := &model.Cluster{CatalogID: cid, MainshockID: 1, Status: model.ClusterCandidate, Confidence: 0.8}
	clid, err := db.InsertCluster(cl)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AddMembership(clid, 1, model.RoleMainshock); err != nil {
		t.Fatal(err)
	}
	if err := db.UpdateCluster(clid, 1, model.ClusterConfirmed, 0.9); err != nil {
		t.Fatal(err)
	}

	v := &model.Version{CatalogID: cid, Number: 1, Label: "v1", Status: model.VersionDraft, SnapshotHash: "h"}
	if _, err := db.InsertVersion(v); err != nil {
		t.Fatal(err)
	}
	num, err := db.NextVersionNumber(cid)
	if err != nil {
		t.Fatal(err)
	}
	if num != 2 {
		t.Fatalf("expected next number 2, got %d", num)
	}
}
