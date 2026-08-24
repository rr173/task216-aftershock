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
