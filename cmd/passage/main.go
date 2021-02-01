package main

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/apex/log"
	"github.com/hightouchio/passage/pkg/service"
	"github.com/hightouchio/passage/pkg/store/postgres"
	"github.com/hightouchio/passage/pkg/tunnels"
	"github.com/hightouchio/passage/pkg/worker"
	_ "github.com/lib/pq"
	"gopkg.in/alecthomas/kingpin.v2"
)

var version = "dev"
var name = "passage"

var (
	addr = kingpin.
		Flag("addr", "").
		Default(":8080").
		String()
	pg = kingpin.
		Flag("pg", "").
		Default("postgres://postgres:postgres@localhost:5432/passage?sslmode=disable").
		String()
)

func main() {
	kingpin.Parse()

	db, err := sql.Open("postgres", *pg)
	if err != nil {
		log.WithError(err).Fatal("connect to postgres")
		return
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.WithError(err).Fatal("ping postgres")
		return
	}

	tunnels := tunnels.NewTunnels(postgres.NewTunnels(db))

	worker := worker.NewWorker(
		tunnels,
		time.Second,
	)

	log.Info("starting worker")
	worker.Start()

	server := &http.Server{
		Addr:    *addr,
		Handler: service.NewService(tunnels),
	}

	log.Infof("starting server on %s", *addr)
	if err := server.ListenAndServe(); err != nil {
		log.WithError(err).Fatal("listen and serve")
	}
}
