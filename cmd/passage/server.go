package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/hightouchio/passage/log"
	"github.com/hightouchio/passage/stats"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/gorilla/mux"
	"github.com/hightouchio/passage/tunnel"
	"github.com/hightouchio/passage/tunnel/postgres"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
)

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

	statsdAddr = kingpin.
		Flag("statsd-addr", "").
		Envar("STATSD_ADDR").
		String()
)

var (
	serverCommand = &cobra.Command{
		Use:   "server",
		Short: "run the passage server",
		RunE:   runServer,
	}

	runAPIServer bool
	runNormalTunnelServer bool
	runReverseTunnelServer bool
)

func init() {
	serverCommand.Flags().BoolVar(&runAPIServer, "api", false, "run API server")
	serverCommand.Flags().BoolVar(&runNormalTunnelServer, "normal", false, "run normal tunnel server")
	serverCommand.Flags().BoolVar(&runReverseTunnelServer,  "reverse", false, "run reverse tunnel server")
}

func runServer(cmd *cobra.Command, args []string) error {
	if !runAPIServer && !runNormalTunnelServer && !runReverseTunnelServer {
		return errors.New("must choose at least one server to run")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	switch os.Getenv("LOG_LEVEL") {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "info":
		logger.SetLevel(logrus.InfoLevel)
	case "warning", "warn":
		logger.SetLevel(logrus.WarnLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}

	switch os.Getenv("LOG_FORMAT") {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{})
	default:
		logger.SetFormatter(&logrus.TextFormatter{})
	}

	healthchecks := newHealthcheckManager()

	// connect to postgres
	db, err := sqlx.Connect("postgres", getPostgresConnString())
	if err != nil {
		return errors.Wrap(err, "could not connect to postgres")
	}
	defer db.Close()
	if err = db.Ping(); err != nil {
		return errors.Wrap(err, "could not ping postgres")
	}
	healthchecks.AddCheck("postgres", db.PingContext)

	// initialize statsd client
	var statsdClient statsd.ClientInterface
	if *statsdAddr != "" {
		var err error
		statsdClient, err = statsd.New(*statsdAddr, statsd.WithMaxBytesPerPayload(4096))
		if err != nil {
			return errors.Wrap(err, "could not initialize statsd client")
		}
	} else {
		statsdClient = &statsd.NoOpClient{}
	}
	statsClient := stats.
		New(statsdClient, logger).
		WithPrefix("passage").
		WithTags(stats.Tags{
			"service": "passage",
			"env":     "production",
		})

	// decode host key from base64
	hostKey, err := base64.StdEncoding.DecodeString(*hostKeyEncoded)
	if err != nil {
		return errors.Wrap(err, "could not decode host key")
	}

	// configure web server
	router := mux.NewRouter()
	router.Handle("/healthcheck", healthchecks)
	// inject global logger into request.
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(log.WithLogger(r.Context(), logger)))
		})
	})

	// configure tunnel server
	tunnelServer := tunnel.NewServer(postgres.NewClient(db), statsClient.WithPrefix("tunnel"), tunnel.SSHOptions{
		BindHost: *bindHost,
		HostKey:  hostKey,
	})

	if runNormalTunnelServer {
		go tunnelServer.StartNormalTunnels(ctx)
		healthchecks.AddCheck("normal_tunnels", tunnelServer.CheckNormalTunnels)
	}

	if runReverseTunnelServer {
		go tunnelServer.StartReverseTunnels(ctx)
		healthchecks.AddCheck("reverse_tunnels", tunnelServer.CheckReverseTunnels)
	}

	if runAPIServer {
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
	return nil
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
