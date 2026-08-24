基于 Go 实现的地震余震簇时空窗识别服务，一款纯后端地震分析服务，处理地震事件聚类、时空窗口识别与可追溯结果发布。

# aftershock 评测说明

地震余震簇时空窗识别服务（纯后端）。

## 运行契约

- **启动服务**：`/app/aftershock --addr :8080 --db aftershock.db`
- **端到端自检**：`/app/aftershock --smoke-test --db smoke.db`
  - 真实导入带定位误差的地震事件（两个主震 + 各自余震 + 一个跨窗口重叠事件）、
    按时空窗识别余震簇并检测重叠边界冲突、锁定主震、拆分冲突事件、
    裁决冲突并发布目录版本，关闭并重开同一数据库验证持久化与重启恢复，
    最终以退出码 0 结束。
  - 这是 Docker `CMD` 与双架构验证的唯一判据，**只传 flag，不传路径位置参数**。

## Docker 双架构验证

```bash
# amd64
docker buildx build --platform linux/amd64 --load -t aftershock:amd64 .
docker run --rm aftershock:amd64 --smoke-test

# arm64
docker buildx build --platform linux/arm64 --load -t aftershock:arm64 .
docker run --rm aftershock:arm64 --smoke-test
```

两项 `docker run` 均须退出码 0。

## 主要 API（前缀 /api）

- 目录：`POST /api/catalogs`、`GET /api/catalogs`、`GET /api/catalogs/{id}`、
  `POST /api/catalogs/{id}/analyzing`、`POST /api/catalogs/{id}/publish`、`POST /api/catalogs/{id}/archive`
- 事件：`POST /api/catalogs/{id}/events`、`GET /api/catalogs/{id}/events`、`GET /api/events/{id}`、
  `POST /api/events/{id}/mark-unstable`、`POST /api/events/{id}/mark-valid`
- 簇识别与复核：`POST /api/catalogs/{id}/cluster`、`GET /api/catalogs/{id}/clusters`、
  `GET /api/clusters/{id}`、`GET /api/clusters/{id}/members`、
  `POST /api/clusters/{id}/lock-mainshock`、`POST /api/clusters/{id}/split`、`POST /api/clusters/{id}/merge`
- 冲突：`GET /api/catalogs/{id}/conflicts`、`POST /api/conflicts/{id}/resolve`
- 版本：`POST /api/versions`、`GET /api/versions?catalog_id=`、`GET /api/versions/{id}`、`POST /api/versions/{id}/publish`
- 统计/健康：`GET /api/stats`、`GET /api/health`

## 环境

- Go 1.26.3，`CGO_ENABLED=0`，`GOPROXY=https://goproxy.cn,direct`，`GOSUMDB=sum.golang.google.cn`
- SQLite 驱动 modernc.org/sqlite v1.52.0（纯 Go），SQLite 3.46.1
