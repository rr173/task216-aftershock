package review_test

import (
	"path/filepath"
	"testing"
	"time"

	"task216-aftershock/internal/model"
	"task216-aftershock/internal/review"
	"task216-aftershock/internal/service"
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

// TestMergeClustersOfFreshlyIdentifiedClusters 合并由识别流程产生的两个余震簇。
// 回归：识别流程若未持久化主震 ID（mainshock_id=0），MergeClusters 在比较震级时
// 会解引用 nil 指针而 panic（merge.go:35）。此处验证合并后主震始终有效且返回正常结果。
func TestMergeClustersOfFreshlyIdentifiedClusters(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "merge.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	catalogID, err := db.InsertCatalog(&model.Catalog{Name: "merge", Status: model.CatalogAnalyzing})
	if err != nil {
		t.Fatal(err)
	}

	// 两个主震（A 的震级大于 B），各自余震，均落入同一时空窗以形成两个候选簇。
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	day := 24 * time.Hour
	mainA := &model.Event{CatalogID: catalogID, OriginTime: base, Latitude: 36.00, Longitude: 140.00, DepthKm: 12, Magnitude: 5.5, LocErrorH: 2, LocErrorZ: 3, Fingerprint: "mainA", Status: model.EventValid}
	aftA := &model.Event{CatalogID: catalogID, OriginTime: base.Add(1 * day), Latitude: 35.90, Longitude: 140.05, DepthKm: 10, Magnitude: 4.5, LocErrorH: 3, LocErrorZ: 4, Fingerprint: "aftA", Status: model.EventValid}
	mainB := &model.Event{CatalogID: catalogID, OriginTime: base.Add(3 * day), Latitude: 36.10, Longitude: 140.10, DepthKm: 11, Magnitude: 5.0, LocErrorH: 2, LocErrorZ: 2, Fingerprint: "mainB", Status: model.EventValid}
	aftB := &model.Event{CatalogID: catalogID, OriginTime: base.Add(4 * day), Latitude: 36.15, Longitude: 140.15, DepthKm: 9, Magnitude: 4.0, LocErrorH: 3, LocErrorZ: 3, Fingerprint: "aftB", Status: model.EventValid}
	for _, e := range []*model.Event{mainA, aftA, mainB, aftB} {
		if _, err := db.InsertEvent(e); err != nil {
			t.Fatal(err)
		}
	}

	// 用服务层执行识别流程，确保主震 ID 被持久化写入簇行。
	svc := service.New(db)
	res, err := svc.IdentifyClusters(catalogID, model.ClusterInput{
		MinMainshockMag:  3.0,
		TimeWindowDays:   30,
		DeltaMagnitude:   1.2,
		UseUtsuDistance:  false,
		DistanceWindowKm: 50,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.ClusterCount < 2 {
		t.Fatalf("expected >=2 clusters, got %d", res.ClusterCount)
	}

	ca, err := db.GetCluster(res.ClusterIDs[0])
	if err != nil {
		t.Fatal(err)
	}
	cb, err := db.GetCluster(res.ClusterIDs[1])
	if err != nil {
		t.Fatal(err)
	}
	// 识别流程必须持久化真实主震 ID，而非遗留 0。
	if ca.MainshockID == 0 || cb.MainshockID == 0 {
		t.Fatalf("identified clusters must persist mainshock id, got A=%d B=%d", ca.MainshockID, cb.MainshockID)
	}

	// 合并两个由识别产生的簇：不应 panic，且返回正常的合并结果。
	merged, err := review.NewManager(db).MergeClusters(ca.ID, cb.ID)
	if err != nil {
		t.Fatalf("MergeClusters failed: %v", err)
	}
	if merged.Status != model.ClusterConfirmed {
		t.Fatalf("expected merged cluster confirmed, got %s", merged.Status)
	}
	// 合并后主震应为震级更大者（mainA, M5.5 > mainB, M5.0）。
	if merged.MainshockID != mainA.ID {
		t.Fatalf("expected merged mainshock %d, got %d", mainA.ID, merged.MainshockID)
	}

	// B 应被标记 merged，A 仍承载全部成员（mainA, aftA, mainB, aftB）。
	cbAfter, err := db.GetCluster(cb.ID)
	if err != nil {
		t.Fatal(err)
	}
	if cbAfter.Status != model.ClusterMerged {
		t.Fatalf("expected cluster B merged, got %s", cbAfter.Status)
	}
	members, err := db.ListMembersByCluster(ca.ID)
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[int64]string, len(members))
	for _, mem := range members {
		seen[mem.EventID] = mem.Role
	}
	if seen[mainA.ID] != model.RoleMainshock {
		t.Fatalf("mainshock A role not preserved: %#v", seen)
	}
	if len(seen) != 4 {
		t.Fatalf("expected 4 members in merged cluster, got %d (%#v)", len(seen), seen)
	}

	// 合并后的簇再次合并应拒绝（B 已 merged）。
	if _, err := review.NewManager(db).MergeClusters(ca.ID, cb.ID); err == nil {
		t.Fatal("expected error merging an already-merged cluster")
	}
}
