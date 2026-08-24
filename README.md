# aftershock — 地震余震簇时空窗识别服务

地震学研究人员将连续地震目录划分为余震簇，识别跨区域重叠簇并保留重新归属的依据。

## 业务闭环

导入带定位误差的地震事件 → 按主震候选、时空窗与震级阈值构建余震簇 →
输出孤立事件与边界冲突 → 研究人员锁定主震、拆分或合并簇 → 发布不可变目录版本。

## 标准命令

```bash
# 构建 / 静态检查 / 测试
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go vet   ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go test  ./...

# 端到端自检
go run ./cmd/aftershock --smoke-test --db smoke.db

# 启动 HTTP 服务
go run ./cmd/aftershock --addr :8080 --db aftershock.db
```

## 状态机

- 地震目录：`importing → analyzing → published → archived`
- 事件：`pending → valid / unstable / duplicate`
- 余震簇：`candidate → overlapping → confirmed / split / merged`
- 目录版本：`draft → published → superseded`

## 持久化

SQLite（modernc.org/sqlite，纯 Go 无 CGO）。保存原始事件、误差范围、簇关联、边界冲突与版本快照；
重启后从游标恢复未完成归属，事件指纹幂等，封存目录只能建立替代版本。

## 关键不变量

- 事件 `UNIQUE(catalog_id, fingerprint)` 幂等，同一目录重复事件不重复计数。
- 主震触发窗口 = 时间窗 × 距离窗（宇津-关经验半径）× 震级下限。
- 一个事件落入多个主震窗口即产生边界冲突，须显式裁决。
- 已发布/封存目录只读；迟到写入只能通过替代版本体现。
