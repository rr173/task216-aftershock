package review_test

import (
	"path/filepath"
	"testing"
	"time"

	"task216-aftershock/internal/model"
	"task216-aftershock/internal/review"
	"task216-aftershock/internal/store"
)

func TestLockMainshockRewritesMembershipRoles(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "review.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	catalog := &model.Catalog{Name: "review", Status: model.CatalogAnalyzing}
	catalogID, err := db.InsertCatalog(catalog)
	if err != nil {
		t.Fatal(err)
	}

	mainshock := &model.Event{CatalogID: catalogID, OriginTime: time.Now().UTC(), Latitude: 36, Longitude: 140, Magnitude: 5.5, Fingerprint: "main", Status: model.EventValid}
	member := &model.Event{CatalogID: catalogID, OriginTime: time.Now().UTC(), Latitude: 36.1, Longitude: 140.1, Magnitude: 4.5, Fingerprint: "member", Status: model.EventValid}
	if _, err := db.InsertEvent(mainshock); err != nil {
		t.Fatal(err)
	}
	if _, err := db.InsertEvent(member); err != nil {
		t.Fatal(err)
	}

	cluster := &model.Cluster{CatalogID: catalogID, MainshockID: mainshock.ID, Status: model.ClusterCandidate, Confidence: 0.8}
	clusterID, err := db.InsertCluster(cluster)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AddMembership(clusterID, mainshock.ID, model.RoleMainshock); err != nil {
		t.Fatal(err)
	}
	if err := db.AddMembership(clusterID, member.ID, model.RoleAftershock); err != nil {
		t.Fatal(err)
	}

	locked, err := review.NewManager(db).LockMainshock(clusterID, member.ID)
	if err != nil {
		t.Fatal(err)
	}
	if locked.MainshockID != member.ID || locked.Status != model.ClusterConfirmed {
		t.Fatalf("unexpected locked cluster: %+v", locked)
	}

	members, err := db.ListMembersByCluster(clusterID)
	if err != nil {
		t.Fatal(err)
	}
	roles := make(map[int64]string, len(members))
	for _, current := range members {
		roles[current.EventID] = current.Role
	}
	if roles[member.ID] != model.RoleMainshock || roles[mainshock.ID] != model.RoleAftershock {
		t.Fatalf("membership roles were not rewritten: %#v", roles)
	}
}

// TestMergeClustersPicksLargerMainshock 验证合并后保留两个簇中震级更大的主震，
// 并同步簇状态与成员角色。
func TestMergeClustersPicksLargerMainshock(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "review.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	catalogID, err := db.InsertCatalog(&model.Catalog{Name: "merge", Status: model.CatalogAnalyzing})
	if err != nil {
		t.Fatal(err)
	}

	// 簇 A 的主震震级 5.5，簇 B 的主震震级 6.2 —— 合并后主震应为 B 的主震。
	mainA := &model.Event{CatalogID: catalogID, OriginTime: time.Now().UTC(), Latitude: 36, Longitude: 140, Magnitude: 5.5, Fingerprint: "mainA", Status: model.EventValid}
	mainB := &model.Event{CatalogID: catalogID, OriginTime: time.Now().UTC(), Latitude: 36.1, Longitude: 140.1, Magnitude: 6.2, Fingerprint: "mainB", Status: model.EventValid}
	aftA := &model.Event{CatalogID: catalogID, OriginTime: time.Now().UTC(), Latitude: 36.2, Longitude: 140.2, Magnitude: 4.0, Fingerprint: "aftA", Status: model.EventValid}
	aftB := &model.Event{CatalogID: catalogID, OriginTime: time.Now().UTC(), Latitude: 36.3, Longitude: 140.3, Magnitude: 4.2, Fingerprint: "aftB", Status: model.EventValid}
	for _, ev := range []*model.Event{mainA, mainB, aftA, aftB} {
		if _, err := db.InsertEvent(ev); err != nil {
			t.Fatal(err)
		}
	}

	clusterAID, err := db.InsertCluster(&model.Cluster{CatalogID: catalogID, MainshockID: mainA.ID, Status: model.ClusterCandidate, Confidence: 0.7})
	if err != nil {
		t.Fatal(err)
	}
	clusterBID, err := db.InsertCluster(&model.Cluster{CatalogID: catalogID, MainshockID: mainB.ID, Status: model.ClusterCandidate, Confidence: 0.9})
	if err != nil {
		t.Fatal(err)
	}
	for _, mem := range []struct {
		cluster int64
		event   int64
		role    string
	}{
		{clusterAID, mainA.ID, model.RoleMainshock},
		{clusterAID, aftA.ID, model.RoleAftershock},
		{clusterBID, mainB.ID, model.RoleMainshock},
		{clusterBID, aftB.ID, model.RoleAftershock},
	} {
		if err := db.AddMembership(mem.cluster, mem.event, mem.role); err != nil {
			t.Fatal(err)
		}
	}

	merged, err := review.NewManager(db).MergeClusters(clusterAID, clusterBID)
	if err != nil {
		t.Fatal(err)
	}
	if merged.MainshockID != mainB.ID {
		t.Fatalf("expected merged mainshock %d (larger magnitude), got %d", mainB.ID, merged.MainshockID)
	}
	if merged.Status != model.ClusterConfirmed {
		t.Fatalf("expected merged cluster A confirmed, got %q", merged.Status)
	}

	b, err := db.GetCluster(clusterBID)
	if err != nil {
		t.Fatal(err)
	}
	if b.Status != model.ClusterMerged {
		t.Fatalf("expected cluster B merged, got %q", b.Status)
	}

	members, err := db.ListMembersByCluster(clusterAID)
	if err != nil {
		t.Fatal(err)
	}
	roles := make(map[int64]string, len(members))
	for _, mem := range members {
		roles[mem.EventID] = mem.Role
	}
	if roles[mainB.ID] != model.RoleMainshock {
		t.Fatalf("new mainshock %d should be mainshock, got %q", mainB.ID, roles[mainB.ID])
	}
	if roles[mainA.ID] != model.RoleAftershock {
		t.Fatalf("demoted old mainshock %d should be aftershock, got %q", mainA.ID, roles[mainA.ID])
	}
	for _, ev := range []int64{aftA.ID, aftB.ID} {
		if roles[ev] != model.RoleAftershock {
			t.Fatalf("aftershock %d should be aftershock, got %q", ev, roles[ev])
		}
	}
}

// TestMergeClustersKeepsLargerWhenAIsLarger 验证当簇 A 的主震震级更大时，合并后主震保持为 A 的主震。
func TestMergeClustersKeepsLargerWhenAIsLarger(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "review.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	catalogID, err := db.InsertCatalog(&model.Catalog{Name: "merge2", Status: model.CatalogAnalyzing})
	if err != nil {
		t.Fatal(err)
	}

	mainA := &model.Event{CatalogID: catalogID, OriginTime: time.Now().UTC(), Latitude: 36, Longitude: 140, Magnitude: 6.5, Fingerprint: "mainA", Status: model.EventValid}
	mainB := &model.Event{CatalogID: catalogID, OriginTime: time.Now().UTC(), Latitude: 36.1, Longitude: 140.1, Magnitude: 5.0, Fingerprint: "mainB", Status: model.EventValid}
	for _, ev := range []*model.Event{mainA, mainB} {
		if _, err := db.InsertEvent(ev); err != nil {
			t.Fatal(err)
		}
	}

	clusterAID, err := db.InsertCluster(&model.Cluster{CatalogID: catalogID, MainshockID: mainA.ID, Status: model.ClusterCandidate, Confidence: 0.7})
	if err != nil {
		t.Fatal(err)
	}
	clusterBID, err := db.InsertCluster(&model.Cluster{CatalogID: catalogID, MainshockID: mainB.ID, Status: model.ClusterCandidate, Confidence: 0.7})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AddMembership(clusterAID, mainA.ID, model.RoleMainshock); err != nil {
		t.Fatal(err)
	}
	if err := db.AddMembership(clusterBID, mainB.ID, model.RoleMainshock); err != nil {
		t.Fatal(err)
	}

	merged, err := review.NewManager(db).MergeClusters(clusterAID, clusterBID)
	if err != nil {
		t.Fatal(err)
	}
	if merged.MainshockID != mainA.ID {
		t.Fatalf("expected merged mainshock %d (A is larger), got %d", mainA.ID, merged.MainshockID)
	}

	members, err := db.ListMembersByCluster(clusterAID)
	if err != nil {
		t.Fatal(err)
	}
	roles := make(map[int64]string, len(members))
	for _, mem := range members {
		roles[mem.EventID] = mem.Role
	}
	if roles[mainA.ID] != model.RoleMainshock {
		t.Fatalf("A's mainshock %d should remain mainshock, got %q", mainA.ID, roles[mainA.ID])
	}
	if roles[mainB.ID] != model.RoleAftershock {
		t.Fatalf("B's demoted mainshock %d should be aftershock, got %q", mainB.ID, roles[mainB.ID])
	}
}
