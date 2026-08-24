package versioning

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"task216-aftershock/internal/store"
)

// Snapshot 计算目录当前归属状态的不可变哈希。
// 涵盖：全部有效事件（指纹+状态）、全部簇（主震+状态+置信度）、簇成员关系。
// 已封存目录的迟到写入因目录只读而无法改变该哈希，保证版本可追溯。
func Snapshot(db *store.DB, catalogID int64) (string, error) {
	events, err := db.ListEventsByCatalog(catalogID)
	if err != nil {
		return "", err
	}
	clusters, err := db.ListClustersByCatalog(catalogID)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	for _, e := range events {
		fmt.Fprintf(&b, "E|%d|%s|%s\n", e.ID, e.Fingerprint, e.Status)
	}

	// 收集成员关系。
	type mem struct {
		clusterID, eventID int64
		role               string
	}
	var members []mem
	for _, c := range clusters {
		ms, err := db.ListMembersByCluster(c.ID)
		if err != nil {
			return "", err
		}
		for _, m := range ms {
			members = append(members, mem{m.ClusterID, m.EventID, m.Role})
		}
	}
	sort.Slice(members, func(i, j int) bool {
		if members[i].clusterID != members[j].clusterID {
			return members[i].clusterID < members[j].clusterID
		}
		return members[i].eventID < members[j].eventID
	})
	for _, m := range members {
		fmt.Fprintf(&b, "M|%d|%d|%s\n", m.clusterID, m.eventID, m.role)
	}

	for _, c := range clusters {
		fmt.Fprintf(&b, "C|%d|%d|%s|%.4f\n", c.ID, c.MainshockID, c.Status, c.Confidence)
	}

	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:16]), nil
}
