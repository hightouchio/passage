package main

import (
	"database/sql"
	"encoding/base64"
	"github.com/DataDog/datadog-go/statsd"
	"github.com/gorilla/mux"
	"github.com/hightouchio/passage/tunnel"
	"github.com/hightouchio/passage/tunnel/postgres"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	"net/http"
)

var version = "dev"
var name = "passage"

var (
	httpAddr = kingpin.
			Flag("http-addr", "").
			Envar("HTTP_ADDR").
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

	hostKeyEncoded = kingpin.
			Flag("host-key", "Base64 encoded").
			Envar("HOST_KEY").
			Required().
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

	statsdAddr = kingpin.
			Flag("statsd-addr", "").
			Envar("STATSD_ADDR").
			String()
)

func main() {
	kingpin.Parse()

	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{})

	// connect to postgres
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

	// initialize statsd client
	var statsClient statsd.ClientInterface
	if *statsdAddr != "" {
		var err error
		statsClient, err = statsd.New(*statsdAddr, statsd.WithMaxBytesPerPayload(4096))
		if err != nil {
			logrus.WithError(err).Fatal("error initializing stated client")
			return
		}
	} else {
		statsClient = &statsd.NoOpClient{}
	}
	router := mux.NewRouter()

	// decode host key from base64
	hostKey, err := base64.StdEncoding.DecodeString(*hostKeyEncoded)
	if err != nil {
		logrus.WithError(err).Fatal("could not decode host key")
		return
	}

	// configur tunnel server
	server := tunnel.NewServer(postgres.NewClient(db), statsClient, tunnel.SSHOptions{
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
		Addr:    *httpAddr,
		Handler: router,
	}

	logrus.WithField("http_addr", *httpAddr).Debug("starting http server")
	if err := httpServer.ListenAndServe(); err != nil {
		logrus.WithError(err).Fatal("http server shutdown")
	}
}
