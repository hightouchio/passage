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

const (
	ConfigEnv        = "env"
	ConfigHTTPAddr   = "http.addr"
	ConfigApiEnabled = "api.enabled"

	ConfigTunnelBindHost        = "tunnel.bind_host"
	ConfigTunnelRefreshInterval = "tunnel.refresh_interval"
	ConfigTunnelRestartInterval = "tunnel.restart_interval"

	ConfigTunnelStandardEnabled           = "tunnel.standard.enabled"
	ConfigTunnelStandardSshUser           = "tunnel.standard.ssh_user"
	ConfigTunnelStandardDialTimeout       = "tunnel.standard.dial_timeout"
	ConfigTunnelStandardKeepaliveInterval = "tunnel.standard.keepalive_interval"
	ConfigTunnelStandardKeepaliveTimeout  = "tunnel.standard.keepalive_timeout"

	ConfigTunnelReverseEnabled  = "tunnel.reverse.enabled"
	ConfigTunnelReverseHostKey  = "tunnel.reverse.host_key"
	ConfigTunnelReverseBindHost = "tunnel.reverse.bind_host"

	ConfigDiscoveryType        = "discovery.type"
	ConfigDiscoverySrvRegistry = "discovery.srv.registry"
	ConfigDiscoverySrvPrefix   = "discovery.srv.prefix"
	ConfigDiscoveryStaticHost  = "discovery.static.host"

	ConfigKeystoreType              = "keystore.type"
	ConfigKeystorePostgresTableName = "keystore.postgres.table_name"
	ConfigKeystoreS3BucketName      = "keystore.s3.bucket_name"
	ConfigKeystoreS3KeyPrefix       = "keystore.s3.key_prefix"

	ConfigPostgresUri     = "postgres.uri"
	ConfigPostgresHost    = "postgres.host"
	ConfigPostgresPort    = "postgres.port"
	ConfigPostgresUser    = "postgres.user"
	ConfigPostgresPass    = "postgres.pass"
	ConfigPostgresDbName  = "postgres.dbname"
	ConfigPostgresSslmode = "postgres.sslmode"

	ConfigLogLevel   = "log.level"
	ConfigLogFormat  = "log.format"
	ConfigStatsdAddr = "statsd.addr"
)

func initDefaults(config *viper.Viper) {
	config.SetDefault(ConfigHTTPAddr, ":8080")
	config.SetDefault(ConfigTunnelRefreshInterval, 1*time.Second)
	config.SetDefault(ConfigTunnelRestartInterval, 15*time.Second)
	config.SetDefault(ConfigTunnelStandardSshUser, "passage")
	config.SetDefault(ConfigTunnelStandardDialTimeout, 15*time.Second)
	config.SetDefault(ConfigTunnelStandardKeepaliveInterval, 1*time.Minute)
	config.SetDefault(ConfigTunnelStandardKeepaliveTimeout, 15*time.Second)
	config.SetDefault(ConfigTunnelReverseBindHost, "0.0.0.0")
	config.SetDefault(ConfigDiscoveryType, "static")
	config.SetDefault(ConfigDiscoveryStaticHost, "localhost")
	config.SetDefault(ConfigKeystoreType, "postgres")
	config.SetDefault(ConfigKeystorePostgresTableName, "passage.keys")
	config.SetDefault(ConfigLogLevel, "info")
	config.SetDefault(ConfigLogFormat, "text")
}

func runServer(cmd *cobra.Command, args []string) error {
	app := fx.New(
		fx.Provide(
			// Main entrypoint.
			newTunnelAPI,
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
			// Start tunnel.Manager goroutines
			runTunnels,
			// Link tunnel.API to the HTTP server
			registerAPIRoutes,
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

// runTunnels is the entrypoint for tunnel servers
func runTunnels(lc fx.Lifecycle, server tunnel.API, sql *sqlx.DB, config *viper.Viper, keystore keystore.Keystore, healthchecks *healthcheckManager, st stats.Stats) error {
	// Helper function for initializing a tunnel.Manager
	runTunnelManager := func(name string, listFunc tunnel.ListFunc) {
		manager := tunnel.NewManager(
			st.WithTags(stats.Tags{"tunnel_type": name}),
			listFunc,
			tunnel.TunnelOptions{
				BindHost: config.GetString(ConfigTunnelBindHost),
			},
			config.GetDuration(ConfigTunnelRefreshInterval),
			config.GetDuration(ConfigTunnelRestartInterval),
		)

		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				go manager.Start(ctx)
				healthchecks.AddCheck(fmt.Sprintf("tunnel_%s", name), manager.Check)
				return nil
			},
			OnStop: func(ctx context.Context) error {
				go manager.Stop(ctx)
				return nil
			},
		})
	}

	if config.GetBool(ConfigTunnelStandardEnabled) {
		runTunnelManager("standard", tunnel.InjectStandardTunnelDependencies(server.GetStandardTunnels, tunnel.StandardTunnelServices{
			SQL:      postgres.NewClient(sql),
			Keystore: keystore,
		}, tunnel.SSHClientOptions{
			User:              config.GetString(ConfigTunnelStandardSshUser),
			DialTimeout:       config.GetDuration(ConfigTunnelStandardDialTimeout),
			KeepaliveInterval: config.GetDuration(ConfigTunnelStandardKeepaliveInterval),
			KeepaliveTimeout:  config.GetDuration(ConfigTunnelStandardKeepaliveTimeout),
		}))
	}

	if config.GetBool(ConfigTunnelReverseEnabled) {
		// Decode config host key from Base64
		hostKey, err := base64.StdEncoding.DecodeString(config.GetString(ConfigTunnelReverseHostKey))
		if err != nil {
			return errors.Wrap(err, "could not decode host key")
		}

		runTunnelManager("reverse", tunnel.InjectReverseTunnelDependencies(server.GetReverseTunnels, tunnel.ReverseTunnelServices{
			SQL:      postgres.NewClient(sql),
			Keystore: keystore,
		}, tunnel.SSHServerOptions{
			BindHost: config.GetString(ConfigTunnelReverseBindHost),
			HostKey:  hostKey,
		}))
	}

	return nil
}

// registerAPIRoutes attaches the API routes to the router
func registerAPIRoutes(config *viper.Viper, logger *logrus.Logger, router *mux.Router, tunnelServer tunnel.API) error {
	if !config.GetBool(ConfigApiEnabled) {
		return nil
	}
	logger.Info("start tunnel web api")
	tunnelServer.ConfigureWebRoutes(router.PathPrefix("/api").Subrouter())
	return nil
}

func newTunnelAPI(sql *sqlx.DB, stats stats.Stats, keystore keystore.Keystore, discovery discovery.DiscoveryService) (tunnel.API, error) {
	return tunnel.API{
		SQL:              postgres.NewClient(sql),
		DiscoveryService: discovery,
		Keystore:         keystore,
		Stats:            stats.WithPrefix("tunnel"),
	}, nil
}

func newTunnelDiscoveryService(config *viper.Viper) (discovery.DiscoveryService, error) {
	var discoveryService discovery.DiscoveryService
	switch config.GetString(ConfigDiscoveryType) {
	case "srv":
		discoveryService = srv.Discovery{
			SrvRegistry: config.GetString(ConfigDiscoverySrvRegistry),
			Prefix:      config.GetString(ConfigDiscoverySrvPrefix),
		}
		break

	case "static":
		discoveryService = static.Discovery{
			Host: config.GetString(ConfigDiscoveryStaticHost),
		}
		break

	default:
		return nil, configError{"unknown discovery type"}
	}
	return discoveryService, nil
}

func newTunnelKeystore(config *viper.Viper, db *sqlx.DB) (keystore.Keystore, error) {
	if !config.IsSet(ConfigKeystoreType) {
		return nil, newConfigError(ConfigKeystoreType, "must be set")
	}

	switch keystoreType := config.GetString(ConfigKeystoreType); keystoreType {
	case "postgres":
		tableName := config.GetString(ConfigKeystorePostgresTableName)
		if tableName == "" {
			return nil, newConfigError(ConfigKeystorePostgresTableName, "must be set")
		}
		return pgkeystore.New(db, tableName), nil

	case "s3":
		bucketName := config.GetString(ConfigKeystoreS3BucketName)
		if bucketName == "" {
			return nil, newConfigError(ConfigKeystoreS3BucketName, "must be set")
		}
		sess, err := session.NewSession()
		if err != nil {
			return nil, configError{"could not init aws session"}
		}
		return s3keystore.Keystore{
			S3:         s3.New(sess),
			BucketName: bucketName,
			KeyPrefix:  config.GetString(ConfigKeystoreS3KeyPrefix),
		}, nil

	default:
		return nil, configError{fmt.Sprintf("unsupported keystore type: %s", keystoreType)}
	}
}

func newHTTPServer(lc fx.Lifecycle, config *viper.Viper, logger *logrus.Logger) *mux.Router {
	router := mux.NewRouter()
	server := &http.Server{Addr: config.GetString(ConfigHTTPAddr), Handler: router}

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

func newConfigError(parts ...string) error {
	return configError{strings.Join(parts, " ")}
}

func newConfig() (*viper.Viper, error) {
	config := viper.New()
	config.AutomaticEnv()
	config.SetEnvPrefix("PASSAGE")
	config.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	initDefaults(config)

	return config, nil
}

func newLogger(config *viper.Viper) *logrus.Logger {
	logger := logrus.New()
	switch config.GetString(ConfigLogLevel) {
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

	switch config.GetString(ConfigLogFormat) {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{})
	default:
		logger.SetFormatter(&logrus.TextFormatter{})
	}
	return logger
}

// newPostgres initializes a connection to the Postgres database
func newPostgres(lc fx.Lifecycle, config *viper.Viper, healthcheck *healthcheckManager) (*sqlx.DB, error) {
	config.SetDefault(ConfigPostgresHost, os.Getenv("PGHOST"))
	config.SetDefault(ConfigPostgresPort, os.Getenv("PGPORT"))
	config.SetDefault(ConfigPostgresUser, os.Getenv("PGUSER"))
	config.SetDefault(ConfigPostgresPass, os.Getenv("PGPASSWORD"))
	config.SetDefault(ConfigPostgresDbName, os.Getenv("PGDBNAME"))
	config.SetDefault(ConfigPostgresSslmode, os.Getenv("PGSSLMODE"))

	db, err := sqlx.Connect("postgres", getPostgresConnString(config))
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
	if config.IsSet(ConfigPostgresUri) {
		return config.GetString(ConfigPostgresUri)
	}

	return formatConnString(map[string]string{
		"host":     config.GetString(ConfigPostgresHost),
		"port":     config.GetString(ConfigPostgresPort),
		"user":     config.GetString(ConfigPostgresUser),
		"password": config.GetString(ConfigPostgresPass),
		"dbname":   config.GetString(ConfigPostgresDbName),
		"sslmode":  config.GetString(ConfigPostgresSslmode),
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

	if statsdAddr := config.GetString(ConfigStatsdAddr); statsdAddr != "" {
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
		WithTags(stats.Tags{"version": version}), nil
}
