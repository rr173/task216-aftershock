// Command aftershock 是地震余震簇时空窗识别服务的入口。
//
// 用法：
//
//	aftershock --addr :8080 --db aftershock.db      # 启动 HTTP 服务
//	aftershock --smoke-test [--db smoke.db]         # 端到端自检（Docker CMD 判据）
package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"task216-aftershock/internal/httpapi"
	"task216-aftershock/internal/service"
	"task216-aftershock/internal/smoke"
	"task216-aftershock/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dbPath := flag.String("db", "aftershock.db", "SQLite database path")
	smokeMode := flag.Bool("smoke-test", false, "run end-to-end smoke test and exit")
	flag.Parse()

	if *smokeMode {
		smoke.Main([]string{*dbPath})
		return
	}

	db, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	app := service.New(db)
	srv := httpapi.New(app)

	log.Printf("aftershock cluster identification listening on %s (db=%s)", *addr, *dbPath)
	if err := http.ListenAndServe(*addr, srv.Handler()); err != nil {
		log.Fatalf("serve: %v", err)
		os.Exit(1)
	}
}
