package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/DataDog/datadog-go/statsd"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gorilla/mux"
	"github.com/hightouchio/passage/log"
	"github.com/hightouchio/passage/stats"
	"github.com/hightouchio/passage/tunnel"
	"github.com/hightouchio/passage/tunnel/discovery"
	"github.com/hightouchio/passage/tunnel/discovery/srv"
	"github.com/hightouchio/passage/tunnel/discovery/static"
	"github.com/hightouchio/passage/tunnel/keystore"
	pgkeystore "github.com/hightouchio/passage/tunnel/keystore/postgres"
	s3keystore "github.com/hightouchio/passage/tunnel/keystore/s3"
	"github.com/hightouchio/passage/tunnel/postgres"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.uber.org/dig"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

var (
	serverCommand = &cobra.Command{
		Use:   "server",
		Short: "passage server is the entrypoint for the HTTP API, the standard tunnel server, and the reverse tunnel server.",
		RunE:  runServer,
	}
)

func runServer(cmd *cobra.Command, args []string) error {
	app := fx.New(
		fx.Provide(
			// Main entrypoint.
			newTunnelServer,
			// Centralized DB for tunnel configs.
			newPostgres,
			// Service for storing and retrieving tunnel public and private keys.
			newTunnelKeystore,
			// Service to discover endpoints of currently running tunnels for a distributed system.
			newTunnelDiscoveryService,
			// Expose an HTTP server for anything that needs it.
			newHTTPServer,
			// Report metrics and events to a statsd collector.
			newStats,
			// Healthcheck manager to detect broken instances of Passage. Reports status over HTTP.
			newHealthcheck,
			// Viper configuration management.
			newConfig,
			// Logger.
			newLogger,
		),

		fx.Invoke(
			registerAPIRoutes,
			runStandardTunnels,
			runReverseTunnels,
		),

		fx.NopLogger,
	)

	startCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := app.Start(startCtx); err != nil {
		switch v := dig.RootCause(err).(type) {
		case configError:
			logrus.Fatalf("config error: %v", v)
		default:
			logrus.Fatalf("startup error: %v", v)
		}
	}

	<-app.Done()

	stopCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := app.Stop(stopCtx); err != nil {
		logrus.Fatalf("shutdown error: %v", dig.RootCause(err))
	}

	return nil
}

// registerAPIRoutes attaches the API routes to the router
func registerAPIRoutes(config *viper.Viper, logger *logrus.Logger, router *mux.Router, tunnelServer tunnel.Server) error {
	if !config.GetBool("api.enabled") {
		return nil
	}
	logger.Info("enabled tunnel APIs")
	tunnelServer.ConfigureWebRoutes(router.PathPrefix("/api").Subrouter())
	return nil
}

// runStandardTunnels is the entrypoint for the Standard Tunnels service
func runStandardTunnels(lc fx.Lifecycle, config *viper.Viper, logger *logrus.Logger, server tunnel.Server, healthchecks *healthcheckManager) {
	if !config.GetBool("tunnel.standard.enabled") {
		return
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go server.StartStandardTunnels(ctx)
			healthchecks.AddCheck("tunnel_standard", server.CheckStandardTunnels)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			server.StopStandardTunnels(ctx)
			return nil
		},
	})
}

// runReverseTunnels is the entrypoint for the Reverse Tunnels service
func runReverseTunnels(lc fx.Lifecycle, config *viper.Viper, logger *logrus.Logger, server tunnel.Server, healthchecks *healthcheckManager) {
	if !config.GetBool("tunnel.reverse.enabled") {
		return
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go server.StartReverseTunnels(ctx)
			healthchecks.AddCheck("tunnel_reverse", server.CheckStandardTunnels)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			server.StopReverseTunnels(ctx)
			return nil
		},
	})
}

func newTunnelServer(config *viper.Viper, sql *sqlx.DB, stats stats.Stats, keystore keystore.Keystore, discovery discovery.DiscoveryService) (tunnel.Server, error) {
	// Initialize SSH options for reverse tunnels
	var sshOptions tunnel.SSHOptions
	if config.GetBool("tunnel.reverse.enabled") {
		config.SetDefault("tunnel.reverse.bind_host", "0.0.0.0")

		// Decode config host key from Base64
		hostKey, err := base64.StdEncoding.DecodeString(config.GetString("tunnel.reverse.host_key"))
		if err != nil {
			return tunnel.Server{}, errors.Wrap(err, "could not decode host key")
		}
		sshOptions.HostKey = hostKey
		// Set bind host.
		sshOptions.BindHost = config.GetString("tunnel.reverse.bind_host")
	}

	// Set outbound SSH user.
	if config.GetBool("tunnel.standard.enabled") {
		config.SetDefault("tunnel.standard.ssh_user", "passage")
		sshOptions.User = config.GetString("tunnel.standard.ssh_user")
	}

	return tunnel.NewServer(
		postgres.NewClient(sql),
		stats.WithPrefix("tunnel"),
		discovery,
		keystore,
		sshOptions,
	), nil
}

func newTunnelDiscoveryService(config *viper.Viper) (discovery.DiscoveryService, error) {
	config.SetDefault("discovery.type", "static")
	config.SetDefault("discovery.host", "localhost")

	var discoveryService discovery.DiscoveryService
	switch config.GetString("discovery.type") {
	case "srv":
		discoveryService = srv.Discovery{
			SrvRegistry: config.GetString("discovery.srv.registry"),
			Prefix:      config.GetString("discovery.srv.prefix"),
		}
		break

	case "static":
		discoveryService = static.Discovery{
			Host: config.GetString("discovery.static.host"),
		}
		break

	default:
		return nil, configError{"unknown discovery type"}
	}
	return discoveryService, nil
}

func newTunnelKeystore(config *viper.Viper, db *sqlx.DB) (keystore.Keystore, error) {
	if !config.IsSet("keystore.type") {
		return nil, configError{"keystore must be set"}
	}

	switch keystoreType := config.GetString("keystore.type"); keystoreType {
	case "postgres":
		tableName := config.GetString("keystore.postgres.table_name")
		if tableName == "" {
			return nil, configError{"keystore.postgres.table_name must be set"}
		}
		return pgkeystore.New(db, config.GetString("postgres.table_name")), nil

	case "s3":
		bucketName := config.GetString("keystore.s3.bucket_name")
		if bucketName == "" {
			return nil, configError{"keystore.s3.bucket_name must be set"}
		}
		sess, err := session.NewSession()
		if err != nil {
			return nil, configError{"could not init aws session"}
		}
		return s3keystore.Keystore{
			S3:         s3.New(sess),
			BucketName: bucketName,
			KeyPrefix:  config.GetString("keystore.s3.key_prefix"),
		}, nil

	default:
		return nil, configError{fmt.Sprintf("unsupported keystore type: %s", keystoreType)}
	}
}

func newHTTPServer(lc fx.Lifecycle, config *viper.Viper, logger *logrus.Logger) *mux.Router {
	router := mux.NewRouter()
	server := &http.Server{Addr: config.GetString("http.addr"), Handler: router}

	// Inject global logger into each request.
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(log.WithLogger(r.Context(), logger)))
		})
	})

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.WithField("addr", server.Addr).Debug("starting HTTP server")
			go func() {
				if err := server.ListenAndServe(); err != nil {
					logrus.Fatal(errors.Wrap(err, "could not start HTTP server"))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return server.Shutdown(ctx)
		},
	})

	return router
}

type configError struct {
	msg string
}

func (e configError) Error() string {
	return e.msg
}

func newConfig() (*viper.Viper, error) {
	viperConfig := viper.New()
	viperConfig.AutomaticEnv()
	viperConfig.SetEnvPrefix("PASSAGE")
	viperConfig.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	viperConfig.SetDefault("env", "")
	viperConfig.SetDefault("http.addr", ":8080")

	return viperConfig, nil
}

func newLogger(config *viper.Viper) *logrus.Logger {
	config.SetDefault("log.level", "info")
	config.SetDefault("log.format", "text")

	logger := logrus.New()
	switch config.GetString("log.level") {
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

	switch config.GetString("log.format") {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{})
	default:
		logger.SetFormatter(&logrus.TextFormatter{})
	}
	return logger
}

// newPostgres initializes a connection to the Postgres database
func newPostgres(lc fx.Lifecycle, config *viper.Viper, healthcheck *healthcheckManager) (*sqlx.DB, error) {
	config.SetDefault("postgres.host", os.Getenv("PGHOST"))
	config.SetDefault("postgres.port", os.Getenv("PGPORT"))
	config.SetDefault("postgres.user", os.Getenv("PGUSER"))
	config.SetDefault("postgres.password", os.Getenv("PGPASSWORD"))
	config.SetDefault("postgres.dbname", os.Getenv("PGDBNAME"))
	config.SetDefault("postgres.sslmode", os.Getenv("PGSSLMODE"))

	db, err := sqlx.Connect("postgres", getPostgresConnString(config.Sub("postgres")))
	if err != nil {
		return nil, errors.Wrap(err, "could not connect to postgres")
	}
	healthcheck.AddCheck("postgres", db.PingContext)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := db.PingContext(ctx); err != nil {
				return errors.Wrap(err, "could not ping postgres")
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return db.Close()
		},
	})

	return db, nil
}

func getPostgresConnString(config *viper.Viper) string {
	if config.IsSet("uri") {
		return config.GetString("uri")
	}

	return formatConnString(map[string]string{
		"host":     config.GetString("host"),
		"port":     config.GetString("port"),
		"user":     config.GetString("user"),
		"password": config.GetString("password"),
		"dbname":   config.GetString("dbname"),
		"sslmode":  config.GetString("sslmode"),
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

// newHealthcheck provides a healthcheck registry and attaches to the HTTP server
func newHealthcheck(router *mux.Router) *healthcheckManager {
	mgr := newHealthcheckManager()
	router.Handle("/healthcheck", mgr)
	return mgr
}

// newStats initializes a Stats client for the server
func newStats(config *viper.Viper, logger *logrus.Logger) (stats.Stats, error) {
	var statsdClient statsd.ClientInterface

	if statsdAddr := config.GetString("statsd.addr"); statsdAddr != "" {
		var err error
		statsdClient, err = statsd.New(statsdAddr, statsd.WithMaxBytesPerPayload(4096))
		if err != nil {
			return stats.Stats{}, errors.Wrap(err, "could not initialize statsd client")
		}
	} else {
		statsdClient = &statsd.NoOpClient{}
	}
	return stats.
		New(statsdClient, logger).
		WithPrefix("passage").
		WithTags(stats.Tags{
			"service": "passage",
			"env":     config.GetString("env"),
			"version": version,
		}), nil
}
