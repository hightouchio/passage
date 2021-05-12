package main

import (
	"database/sql"
	"github.com/gorilla/mux"
	"github.com/hightouchio/passage/tunnel"
	"github.com/hightouchio/passage/tunnel/postgres"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	"io/ioutil"
	"net/http"
)

var version = "dev"
var name = "passage"

var (
	addr = kingpin.
		Flag("addr", "").
		Envar("ADDR").
		Default(":8080").
		String()

	postgresUri = kingpin.
			Flag("pg-uri", "").
			Envar("PG_URI").
			Default("postgres://postgres:postgres@localhost:5432/passage?sslmode=disable").
			String()

	bindHost = kingpin.
			Flag("bind-host", "").
			Envar("BIND_HOST").
			Default("localhost").
			String()

	hostKeyPath = kingpin.
			Flag("host-key-path", "").
			Envar("HOST_KEY_PATH").
			String()

	disableNormal = kingpin.
			Flag("disable-normal", "").
			Envar("DISABLE_NORMAL").
			Default("false").
			Bool()

	disableReverse = kingpin.
			Flag("disable-reverse", "").
			Envar("DISABLE_REVERSE").
			Default("false").
			Bool()
)

func main() {
	kingpin.Parse()

	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{})

	db, err := sql.Open("postgres", *postgresUri)
	if err != nil {
		logrus.WithError(err).Fatal("connect to postgres")
		return
	}
	defer db.Close()
	if err = db.Ping(); err != nil {
		logrus.WithError(err).Fatal("ping postgres")
		return
	}

	// read host key from disk
	hostKey, err := ioutil.ReadFile(*hostKeyPath)
	if err != nil {
		logrus.WithError(err).Fatal("read host key")
		return
	}

	router := mux.NewRouter()

	server := tunnel.NewServer(postgres.NewClient(db), tunnel.SSHOptions{
		BindHost: *bindHost,
		HostKey:  hostKey,
	})
	server.ConfigureWebRoutes(router.PathPrefix("/api").Subrouter())

	if !*disableNormal {
		logrus.Debug("starting normal tunnels")
		server.StartNormalTunnels()
		defer server.StopNormalTunnels()
	}

	if !*disableReverse {
		logrus.Debug("starting reverse tunnels")
		server.StartReverseTunnels()
		defer server.StopReverseTunnels()
	}

	httpServer := &http.Server{
		Addr:    *addr,
		Handler: router,
	}

	logrus.WithField("bindAddr", *addr).Debug("starting http server")
	if err := httpServer.ListenAndServe(); err != nil {
		logrus.WithError(err).Fatal("http server shutdown")
	}
}
