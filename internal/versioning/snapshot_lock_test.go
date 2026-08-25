package versioning_test

import (
	"path/filepath"
	"testing"
	"time"

	"task216-aftershock/internal/model"
	"task216-aftershock/internal/review"
	"task216-aftershock/internal/store"
	"task216-aftershock/internal/versioning"
)

// TestLockMainshockChangesSnapshotHash 验证：研究人员把簇内一个余震重新锁定为主震后，
// 成员角色发生变更，目录快照哈希必须随之改变且可追溯。
func TestLockMainshockChangesSnapshotHash(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "lock_snapshot.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	cat := &model.Catalog{Name: "lock", Region: "r", Status: model.CatalogAnalyzing}
	catalogID, err := db.InsertCatalog(cat)
	if err != nil {
		t.Fatal(err)
	}

	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	origMain := &model.Event{CatalogID: catalogID, OriginTime: base, Latitude: 36, Longitude: 140, DepthKm: 12, Magnitude: 5.0, Fingerprint: "orig", Status: model.EventValid}
	aftershock := &model.Event{CatalogID: catalogID, OriginTime: base.Add(24 * time.Hour), Latitude: 36.05, Longitude: 140.05, DepthKm: 10, Magnitude: 4.5, Fingerprint: "aft", Status: model.EventValid}
	if _, err := db.InsertEvent(origMain); err != nil {
		t.Fatal(err)
	}
	if _, err := db.InsertEvent(aftershock); err != nil {
		t.Fatal(err)
	}

	cluster := &model.Cluster{CatalogID: catalogID, MainshockID: origMain.ID, Status: model.ClusterCandidate, Confidence: 0.8}
	clusterID, err := db.InsertCluster(cluster)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AddMembership(clusterID, origMain.ID, model.RoleMainshock); err != nil {
		t.Fatal(err)
	}
	if err := db.AddMembership(clusterID, aftershock.ID, model.RoleAftershock); err != nil {
		t.Fatal(err)
	}

	// 锁定前的快照哈希。
	hashBefore, err := versioning.Snapshot(db, catalogID)
	if err != nil {
		t.Fatal(err)
	}

	// 重新锁定余震为主震（合法成员，角色由 aftershock 升为 mainshock）。
	if _, err := review.NewManager(db).LockMainshock(clusterID, aftershock.ID); err != nil {
		t.Fatalf("locking a legitimate aftershock as mainshock should succeed: %v", err)
	}

	// 角色变更必须落库可见。
	members, err := db.ListMembersByCluster(clusterID)
	if err != nil {
		t.Fatal(err)
	}
	roles := make(map[int64]string, len(members))
	for _, mem := range members {
		roles[mem.EventID] = mem.Role
	}
	if roles[aftershock.ID] != model.RoleMainshock || roles[origMain.ID] != model.RoleAftershock {
		t.Fatalf("membership roles not rewritten after lock: %#v", roles)
	}

	// 锁定后的快照哈希必须不同——角色变更须在目录快照中可追溯。
	hashAfter, err := versioning.Snapshot(db, catalogID)
	if err != nil {
		t.Fatal(err)
	}
	if hashAfter == hashBefore {
		t.Fatalf("snapshot hash unchanged after role rewrite: before=%s after=%s", hashBefore, hashAfter)
	}

	// 快照哈希必须对角色敏感：在锁定后状态下仅交换两个事件的角色，
	// 哈希应再次改变（证明哈希计入 role 字段，而非仅依赖成员集合）。
	swapBefore := hashAfter
	if err := db.DeleteMembership(clusterID, aftershock.ID); err != nil {
		t.Fatal(err)
	}
	if err := db.AddMembership(clusterID, aftershock.ID, model.RoleAftershock); err != nil {
		t.Fatal(err)
	}
	if err := db.DeleteMembership(clusterID, origMain.ID); err != nil {
		t.Fatal(err)
	}
	if err := db.AddMembership(clusterID, origMain.ID, model.RoleMainshock); err != nil {
		t.Fatal(err)
	}
	swapAfter, err := versioning.Snapshot(db, catalogID)
	if err != nil {
		t.Fatal(err)
	}
	if swapAfter == swapBefore {
		t.Fatalf("snapshot hash not role-sensitive: unchanged after role swap: %s", swapAfter)
	}
}
