package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/jmoiron/sqlx"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/gorilla/mux"
	"github.com/hightouchio/passage/tunnel"
	"github.com/hightouchio/passage/tunnel/postgres"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
)

var version = "dev"
var name = "passage"

var (
	httpAddr = kingpin.
			Flag("http-addr", "").
			Envar("HTTP_ADDR").
			Default(":8080").
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

	runServicesStr = kingpin.
			Flag("services", "Services to run").
			Envar("SERVICES").
			Default("api,normal,reverse").
			String()

	statsdAddr = kingpin.
			Flag("statsd-addr", "").
			Envar("STATSD_ADDR").
			String()
)

func init() {
	kingpin.Parse()
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{})

	healthchecks := newHealthcheckManager()

	// connect to postgres
	db, err := sqlx.Connect("postgres", getPostgresConnString())
	if err != nil {
		logrus.WithError(err).Fatal("connect to postgres")
		return
	}
	defer db.Close()
	if err = db.Ping(); err != nil {
		logrus.WithError(err).Fatal("ping postgres")
		return
	}
	healthchecks.AddCheck("postgres", db.PingContext)

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

	// decode host key from base64
	hostKey, err := base64.StdEncoding.DecodeString(*hostKeyEncoded)
	if err != nil {
		logrus.WithError(err).Fatal("could not decode host key")
		return
	}

	// configure web server
	router := mux.NewRouter()
	router.Handle("/healthcheck", healthchecks)

	// configure tunnel server
	tunnelServer := tunnel.NewServer(postgres.NewClient(db), statsClient, tunnel.SSHOptions{
		BindHost: *bindHost,
		HostKey:  hostKey,
	})

	if shouldRunService("normal") {
		logrus.Debug("starting normal tunnels")
		go tunnelServer.StartNormalTunnels(ctx)
		healthchecks.AddCheck("normal_tunnels", tunnelServer.CheckNormalTunnels)
	}

	if shouldRunService("reverse") {
		logrus.Debug("starting reverse tunnels")
		go tunnelServer.StartReverseTunnels(ctx)
		healthchecks.AddCheck("reverse_tunnels", tunnelServer.CheckReverseTunnels)
	}

	if shouldRunService("api") {
		tunnelServer.ConfigureWebRoutes(router.PathPrefix("/api").Subrouter())

		// start HTTP server
		httpServer := &http.Server{Addr: *httpAddr, Handler: router}
		go func() {
			logrus.WithField("http_addr", *httpAddr).Debug("starting http server")
			if err := httpServer.ListenAndServe(); err != nil {
				logrus.WithError(err).Fatal("http server shutdown")
			}
		}()
		go func() {
			<-ctx.Done()
			httpServer.Shutdown(context.Background())
		}()
	}

	<-ctx.Done()
}

var runServices []string

func init() {
	runServices = strings.Split(*runServicesStr, ",")
	if len(runServices) == 0 || runServices[0] == "" {
		logrus.Fatal("must specify services to run")
	}
}

func shouldRunService(service string) bool {
	for _, s := range runServices {
		if s == service {
			return true
		}
	}
	return false
}

func getPostgresConnString() string {
	if os.Getenv("PG_URI") != "" {
		return os.Getenv("PG_URI")
	}

	return formatConnString(map[string]string{
		"host":     os.Getenv("PGHOST"),
		"port":     os.Getenv("PGPORT"),
		"user":     os.Getenv("PGUSER"),
		"password": os.Getenv("PGPASSWORD"),
		"dbname":   os.Getenv("PGDBNAME"),
		"sslmode":  os.Getenv("PGSSLMODE"),
	})
}

func formatConnString(mapping map[string]string) string {
	var r string
	for key, val := range mapping {
		if val != "" {
			r = r + " " + fmt.Sprintf("%s=%s", key, val)
		}
	}
	return r
}
