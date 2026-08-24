// Package smoke 实现 --smoke-test 端到端自检：真实导入带定位误差的地震事件、
// 按时空窗识别余震簇、检测重叠边界冲突、锁定主震、拆分冲突事件、发布目录版本，
// 关闭并重新打开同一数据库验证持久化与重启恢复，最后以 0 退出码结束。
package smoke

import (
	"fmt"
	"os"
	"time"

	"task216-aftershock/internal/event"
	"task216-aftershock/internal/model"
	"task216-aftershock/internal/service"
	"task216-aftershock/internal/store"
)

// Main 自检入口：args[0] 为数据库路径。
func Main(args []string) {
	dbPath := "aftershock.db"
	if len(args) > 0 && args[0] != "" {
		dbPath = args[0]
	}
	if err := Run(dbPath); err != nil {
		fmt.Fprintf(os.Stderr, "smoke test FAILED: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("smoke test PASSED")
}

// sampleEvents 构造样例事件：两个主震、各自余震，以及一个跨窗口重叠事件。
func sampleEvents() []event.EventInput {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	day := 24 * time.Hour
	return []event.EventInput{
		// 主震 A（M5.5）及其余震。
		{OriginTime: base, Latitude: 36.00, Longitude: 140.00, DepthKm: 12, Magnitude: 5.5, LocErrorH: 2, LocErrorZ: 3},
		{OriginTime: base.Add(1 * day), Latitude: 35.90, Longitude: 140.05, DepthKm: 10, Magnitude: 4.5, LocErrorH: 3, LocErrorZ: 4},
		// 主震 B（M5.0，位于 A 的窗口内，自身也是主震候选）。
		{OriginTime: base.Add(3 * day), Latitude: 36.10, Longitude: 140.10, DepthKm: 11, Magnitude: 5.0, LocErrorH: 2, LocErrorZ: 2},
		// B 的余震。
		{OriginTime: base.Add(4 * day), Latitude: 36.15, Longitude: 140.15, DepthKm: 9, Magnitude: 4.0, LocErrorH: 3, LocErrorZ: 3},
		// 重叠事件 X：同时落入 A 与 B 的窗口，触发边界冲突。
		{OriginTime: base.Add(5 * day), Latitude: 36.05, Longitude: 140.05, DepthKm: 13, Magnitude: 4.5, LocErrorH: 4, LocErrorZ: 5},
	}
}

// Run 执行端到端自检，返回 nil 表示全部通过。
func Run(dbPath string) error {
	var catalogID int64
	var clusterA, clusterB int64
	var conflictID int64
	var versionID int64

	step := func(n int, name string, fn func() error) error {
		fmt.Printf("[smoke %d/8] %s ...\n", n, name)
		if err := fn(); err != nil {
			return fmt.Errorf("smoke step %d (%s): %w", n, name, err)
		}
		return nil
	}

	// 第一步：创建目录并导入样例事件。
	if err := step(1, "创建目录并导入样例事件", func() error {
		db, err := store.Open(dbPath)
		if err != nil {
			return err
		}
		defer db.Close()
		app := service.New(db)

		c, err := app.Catalog.Create("smoke-catalog", "示例震区")
		if err != nil {
			return err
		}
		catalogID = c.ID

		res, err := app.Catalog.IngestBatch(catalogID, sampleEvents())
		if err != nil {
			return err
		}
		if res.Accepted < 5 {
			return fmt.Errorf("expected >=5 accepted events, got %d", res.Accepted)
		}
		return nil
	}); err != nil {
		return err
	}

	// 第二步：按时空窗识别余震簇（显式 50km/30 天窗口）。
	if err := step(2, "识别余震簇", func() error {
		db, err := store.Open(dbPath)
		if err != nil {
			return err
		}
		defer db.Close()
		app := service.New(db)

		res, err := app.IdentifyClusters(catalogID, model.ClusterInput{
			MinMainshockMag:  3.0,
			TimeWindowDays:   30,
			DeltaMagnitude:   1.2,
			UseUtsuDistance:  false,
			DistanceWindowKm: 50,
		})
		if err != nil {
			return err
		}
		if res.ClusterCount < 2 {
			return fmt.Errorf("expected >=2 clusters, got %d", res.ClusterCount)
		}
		if res.ConflictCount < 1 {
			return fmt.Errorf("expected >=1 boundary conflict, got %d", res.ConflictCount)
		}
		if len(res.ClusterIDs) >= 2 {
			clusterA = res.ClusterIDs[0]
			clusterB = res.ClusterIDs[1]
		}
		return nil
	}); err != nil {
		return err
	}

	// 第三步：读取冲突记录，确认重叠事件被标记。
	if err := step(3, "读取边界冲突", func() error {
		db, err := store.Open(dbPath)
		if err != nil {
			return err
		}
		defer db.Close()
		confs, err := db.ListConflictsByCatalog(catalogID)
		if err != nil {
			return err
		}
		if len(confs) == 0 {
			return fmt.Errorf("no conflicts persisted")
		}
		conflictID = confs[0].ID
		return nil
	}); err != nil {
		return err
	}

	// 第四步：锁定簇 A 的主震。
	if err := step(4, "锁定簇 A 主震", func() error {
		db, err := store.Open(dbPath)
		if err != nil {
			return err
		}
		defer db.Close()
		app := service.New(db)

		ca, err := app.Review.GetCluster(clusterA)
		if err != nil {
			return err
		}
		locked, err := app.Review.LockMainshock(clusterA, ca.MainshockID)
		if err != nil {
			return err
		}
		if locked.Status != model.ClusterConfirmed {
			return fmt.Errorf("expected cluster confirmed, got %s", locked.Status)
		}
		return nil
	}); err != nil {
		return err
	}

	// 第五步：拆分冲突事件（把重叠事件拆出到新簇）。
	if err := step(5, "拆分冲突事件", func() error {
		db, err := store.Open(dbPath)
		if err != nil {
			return err
		}
		defer db.Close()
		app := service.New(db)

		conf, err := db.GetConflict(conflictID)
		if err != nil {
			return err
		}
		if _, err := app.Review.SplitCluster(clusterB, []int64{conf.EventID}); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	// 第六步：裁决冲突并发布目录版本。
	if err := step(6, "裁决冲突并发布版本", func() error {
		db, err := store.Open(dbPath)
		if err != nil {
			return err
		}
		defer db.Close()
		app := service.New(db)

		if err := app.ResolveConflict(conflictID, model.ResolutionAssignA); err != nil {
			return err
		}
		v, err := app.Versioning.CreateDraft(catalogID, "smoke-release")
		if err != nil {
			return err
		}
		pub, err := app.Versioning.Publish(v.ID)
		if err != nil {
			return err
		}
		if pub.Status != model.VersionPublished {
			return fmt.Errorf("expected version published, got %s", pub.Status)
		}
		versionID = pub.ID
		return nil
	}); err != nil {
		return err
	}

	// 第七步：关闭并重新打开数据库，验证持久化与重启恢复。
	if err := step(7, "重启恢复验证", func() error {
		db, err := store.Open(dbPath)
		if err != nil {
			return err
		}
		defer db.Close()
		app := service.New(db)

		c, err := app.Catalog.Get(catalogID)
		if err != nil {
			return err
		}
		clusters, err := app.Review.ListClusters(catalogID)
		if err != nil {
			return err
		}
		v, err := app.Versioning.Get(versionID)
		if err != nil {
			return err
		}
		if c.ID != catalogID {
			return fmt.Errorf("catalog not recovered")
		}
		if len(clusters) < 2 {
			return fmt.Errorf("expected >=2 clusters after reopen, got %d", len(clusters))
		}
		if v.Status != model.VersionPublished {
			return fmt.Errorf("version state lost after reopen")
		}
		return nil
	}); err != nil {
		return err
	}

	// 第八步：验证已封存目录拒绝写入（错误边界）。
	if err := step(8, "封存目录只读边界", func() error {
		db, err := store.Open(dbPath)
		if err != nil {
			return err
		}
		defer db.Close()
		app := service.New(db)

		if err := app.Catalog.MarkAnalyzing(catalogID); err != nil {
			return err
		}
		if err := app.Catalog.Publish(catalogID); err != nil {
			return err
		}
		if err := app.Catalog.Archive(catalogID); err != nil {
			return err
		}
		_, _, err = app.Catalog.IngestEvent(catalogID, event.EventInput{
			OriginTime: time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC),
			Latitude:   36.0, Longitude: 140.0, DepthKm: 10, Magnitude: 3.0,
		})
		if err != model.ErrArchived {
			return fmt.Errorf("expected ErrArchived on archived catalog, got %v", err)
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}
