package catalog_test

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"task216-aftershock/internal/catalog"
	"task216-aftershock/internal/event"
	"task216-aftershock/internal/model"
	"task216-aftershock/internal/store"
)

// newManagerWithCatalog 打开临时数据库并创建一个处于 importing 状态的目录。
func newManagerWithCatalog(t *testing.T) (*catalog.Manager, int64, func()) {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "catalog.db"))
	if err != nil {
		t.Fatal(err)
	}
	c, err := catalog.NewManager(db).Create("cat", "region")
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	return catalog.NewManager(db), c.ID, func() { db.Close() }
}

// 导入事件不应推进目录状态：目录仍须停留在 importing，等待显式 MarkAnalyzing。
func TestIngestKeepsCatalogImporting(t *testing.T) {
	mgr, id, cleanup := newManagerWithCatalog(t)
	defer cleanup()

	if _, _, err := mgr.IngestEvent(id, event.EventInput{
		OriginTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Latitude:   36.0, Longitude: 140.0, DepthKm: 10, Magnitude: 4.5,
		LocErrorH: 1, LocErrorZ: 1,
	}); err != nil {
		t.Fatalf("ingest failed: %v", err)
	}

	c, err := mgr.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	if c.Status != model.CatalogImporting {
		t.Fatalf("ingest advanced status to %q, want importing", c.Status)
	}
}

// 处于 importing 的目录必须被拒绝发布并保留导入状态。
func TestPublishRejectsImportingCatalog(t *testing.T) {
	mgr, id, cleanup := newManagerWithCatalog(t)
	defer cleanup()

	err := mgr.Publish(id)
	if !errors.Is(err, model.ErrConflict) {
		t.Fatalf("expected ErrConflict publishing importing catalog, got %v", err)
	}

	c, err := mgr.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	if c.Status != model.CatalogImporting {
		t.Fatalf("publish mutated importing catalog to %q, want importing", c.Status)
	}
}

// 目录经显式 MarkAnalyzing 后才能发布。
func TestPublishAllowedAfterMarkAnalyzing(t *testing.T) {
	mgr, id, cleanup := newManagerWithCatalog(t)
	defer cleanup()

	if err := mgr.MarkAnalyzing(id); err != nil {
		t.Fatalf("mark analyzing: %v", err)
	}
	if err := mgr.Publish(id); err != nil {
		t.Fatalf("publish after analyzing: %v", err)
	}

	c, err := mgr.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	if c.Status != model.CatalogPublished {
		t.Fatalf("status = %q, want published", c.Status)
	}
}
