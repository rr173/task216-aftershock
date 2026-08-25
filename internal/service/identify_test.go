package service

import (
	"path/filepath"
	"testing"
	"time"

	"task216-aftershock/internal/event"
	"task216-aftershock/internal/model"
	"task216-aftershock/internal/store"
)

// sampleConflictEvents 构造两个主震与一个跨窗口重叠事件，触发边界冲突。
func sampleConflictEvents() []event.EventInput {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	day := 24 * time.Hour
	return []event.EventInput{
		{OriginTime: base, Latitude: 36.00, Longitude: 140.00, DepthKm: 12, Magnitude: 5.5, LocErrorH: 2, LocErrorZ: 3},
		{OriginTime: base.Add(1 * day), Latitude: 35.90, Longitude: 140.05, DepthKm: 10, Magnitude: 4.5, LocErrorH: 3, LocErrorZ: 4},
		{OriginTime: base.Add(3 * day), Latitude: 36.10, Longitude: 140.10, DepthKm: 11, Magnitude: 5.0, LocErrorH: 2, LocErrorZ: 2},
		{OriginTime: base.Add(4 * day), Latitude: 36.15, Longitude: 140.15, DepthKm: 9, Magnitude: 4.0, LocErrorH: 3, LocErrorZ: 3},
		{OriginTime: base.Add(5 * day), Latitude: 36.05, Longitude: 140.05, DepthKm: 13, Magnitude: 4.5, LocErrorH: 4, LocErrorZ: 5},
	}
}

func clusterInput() model.ClusterInput {
	return model.ClusterInput{
		MinMainshockMag:  3.0,
		TimeWindowDays:   30,
		DeltaMagnitude:   1.2,
		UseUtsuDistance:   false,
		DistanceWindowKm: 50,
	}
}

// TestIdentifyClustersRebuildsConflictsOnRepeat 验证同一目录用相同参数重复识别余震簇时，
// 旧冲突被清空重建，冲突数量与内容保持一致，不产生重复或悬空记录。
func TestIdentifyClustersRebuildsConflictsOnRepeat(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "repeat.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app := New(db)

	c, err := app.Catalog.Create("repeat", "r")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.Catalog.IngestBatch(c.ID, sampleConflictEvents()); err != nil {
		t.Fatal(err)
	}

	in := clusterInput()

	// 第一次识别。
	first, err := app.IdentifyClusters(c.ID, in)
	if err != nil {
		t.Fatal(err)
	}
	if first.ConflictCount == 0 {
		t.Fatal("expected at least one boundary conflict on first run")
	}
	firstRows, err := db.ListConflictsByCatalog(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(firstRows) != first.ConflictCount {
		t.Fatalf("first run persisted %d conflict rows, want %d", len(firstRows), first.ConflictCount)
	}

	// 第二次以相同参数重复识别：应重建当前结果，而非追加。
	second, err := app.IdentifyClusters(c.ID, in)
	if err != nil {
		t.Fatal(err)
	}
	if second.ConflictCount != first.ConflictCount {
		t.Fatalf("conflict count drifted across runs: first=%d second=%d", first.ConflictCount, second.ConflictCount)
	}
	secondRows, err := db.ListConflictsByCatalog(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(secondRows) != first.ConflictCount {
		t.Fatalf("persisted conflict rows after repeat run: %d, want %d (no accumulation)", len(secondRows), first.ConflictCount)
	}

	// 重建后的冲突不应残留指向已删除簇的悬空引用。
	clusters, err := db.ListClustersByCatalog(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	live := make(map[int64]bool, len(clusters))
	for _, cl := range clusters {
		live[cl.ID] = true
	}
	for _, cf := range secondRows {
		if !live[cf.ClusterAID] || !live[cf.ClusterBID] {
			t.Fatalf("conflict %d references a non-existent cluster: a=%d b=%d", cf.ID, cf.ClusterAID, cf.ClusterBID)
		}
		if cf.Status != model.ConflictOpen {
			t.Fatalf("conflict %d status = %q, want %q", cf.ID, cf.Status, model.ConflictOpen)
		}
	}

	// 第三次重复：幂等性再确认（数量与内容一致）。
	third, err := app.IdentifyClusters(c.ID, in)
	if err != nil {
		t.Fatal(err)
	}
	thirdRows, err := db.ListConflictsByCatalog(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if third.ConflictCount != first.ConflictCount || len(thirdRows) != first.ConflictCount {
		t.Fatalf("conflict count not stable on third run: got %d (count) / %d (rows), want %d",
			third.ConflictCount, len(thirdRows), first.ConflictCount)
	}
}
