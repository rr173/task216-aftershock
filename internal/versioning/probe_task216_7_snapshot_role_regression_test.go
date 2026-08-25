package versioning_test

import (
	"path/filepath"
	"testing"
	"time"

	"task216-aftershock/internal/catalog"
	"task216-aftershock/internal/model"
	"task216-aftershock/internal/review"
	"task216-aftershock/internal/store"
	"task216-aftershock/internal/versioning"
)

func TestSnapshotChangesWhenMembershipRoleChanges(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "snapshot.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	c, err := catalog.NewManager(db).Create("catalog", "region")
	if err != nil {
		t.Fatal(err)
	}
	mainshock := &model.Event{CatalogID: c.ID, OriginTime: time.Now().UTC(), Latitude: 36, Longitude: 140, Magnitude: 5.5, Fingerprint: "main", Status: model.EventValid}
	member := &model.Event{CatalogID: c.ID, OriginTime: time.Now().UTC(), Latitude: 36.1, Longitude: 140.1, Magnitude: 4.5, Fingerprint: "member", Status: model.EventValid}
	if _, err := db.InsertEvent(mainshock); err != nil {
		t.Fatal(err)
	}
	if _, err := db.InsertEvent(member); err != nil {
		t.Fatal(err)
	}
	cluster := &model.Cluster{CatalogID: c.ID, MainshockID: mainshock.ID, Status: model.ClusterCandidate}
	if _, err := db.InsertCluster(cluster); err != nil {
		t.Fatal(err)
	}
	if err := db.AddMembership(cluster.ID, mainshock.ID, model.RoleMainshock); err != nil {
		t.Fatal(err)
	}
	if err := db.AddMembership(cluster.ID, member.ID, model.RoleAftershock); err != nil {
		t.Fatal(err)
	}
	before, err := versioning.Snapshot(db, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := review.NewManager(db).LockMainshock(cluster.ID, member.ID); err != nil {
		t.Fatal(err)
	}
	after, err := versioning.Snapshot(db, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Fatalf("snapshot hash did not change after role reassignment: %s", before)
	}
}
