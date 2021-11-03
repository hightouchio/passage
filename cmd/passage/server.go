package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/DataDog/datadog-go/statsd"
	"github.com/gorilla/mux"
	"github.com/hightouchio/passage/log"
	"github.com/hightouchio/passage/stats"
	"github.com/hightouchio/passage/tunnel"
	"github.com/hightouchio/passage/tunnel/discovery"
	"github.com/hightouchio/passage/tunnel/discovery/srv"
	"github.com/hightouchio/passage/tunnel/discovery/static"
	"github.com/hightouchio/passage/tunnel/postgres"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"net/http"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

var (
	viperConfig   = viper.New()
	serverCommand = &cobra.Command{
		Use:   "server",
		Short: "passage server is the entrypoint for the HTTP API, the standard tunnel server, and the reverse tunnel server.",
		RunE:  runServer,
	}
)

func init() {
	viperConfig.AutomaticEnv()
	viperConfig.SetEnvPrefix("PASSAGE")
	viperConfig.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	viperConfig.SetDefault("env", "")

	viperConfig.SetDefault("log.level", "info")
	viperConfig.SetDefault("log.format", "text")

	viperConfig.BindEnv("postgres.host", "PGHOST")
	viperConfig.BindEnv("postgres.port", "PGPORT")
	viperConfig.BindEnv("postgres.user", "PGUSER")
	viperConfig.BindEnv("postgres.password", "PGPASSWORD")
	viperConfig.BindEnv("postgres.dbname", "PGDBNAME")
	viperConfig.BindEnv("postgres.sslmode", "PGSSLMODE")

	viperConfig.SetDefault("api.enabled", false)
	viperConfig.SetDefault("api.listenAddr", ":8080")

	viperConfig.SetDefault("tunnel.discovery.type", "static")
	viperConfig.SetDefault("tunnel.discovery.host", "localhost")

	viperConfig.SetDefault("tunnel.reverse.enabled", false)
	viperConfig.SetDefault("tunnel.reverse.ssh.bindHost", "localhost")

	viperConfig.SetDefault("tunnel.standard.enabled", false)
}

func init() {
	serverCommand.Flags().Bool("api", false, "run API server")
	viperConfig.BindPFlag("api.enabled", serverCommand.Flags().Lookup("api"))

	serverCommand.Flags().Bool("standard", false, "run standard tunnel server")
	viperConfig.BindPFlag("tunnel.standard.enabled", serverCommand.Flags().Lookup("standard"))

	serverCommand.Flags().Bool("reverse", false, "run reverse tunnel server")
	viperConfig.BindPFlag("tunnel.reverse.enabled", serverCommand.Flags().Lookup("reverse"))
}

func runServer(cmd *cobra.Command, args []string) error {
	app := fx.New(
		fx.Provide(
			newConfig,
			newLogger,
			newPostgres,
			newTunnelServer,
			newTunnelDiscoveryService,
			newHealthcheck,
			newStats,
			newHTTPServer,
		 ),

		fx.Invoke(
			registerAPIRoutes,
			runStandardTunnels,
			runReverseTunnels,
		 ),

		 fx.NopLogger,
	)

	startCtx, cancel := context.WithTimeout(context.Background(), 15 * time.Second)
	defer cancel()

	if err := app.Start(startCtx); err != nil {
		logrus.Fatal(err)
	}

	<-app.Done()

	stopCtx, cancel := context.WithTimeout(context.Background(), 15 * time.Second)
	defer cancel()

	if err := app.Stop(stopCtx); err != nil {
		logrus.Fatal(err)
	}

	return nil
}

// registerAPIRoutes attaches the API routes to the router
func registerAPIRoutes(config Config, logger *logrus.Logger, router *mux.Router, tunnelServer tunnel.Server) error {
	if !config.APIEnabled {
		return nil
	}
	logger.Info("enabled tunnel APIs")
	tunnelServer.ConfigureWebRoutes(router)
	return nil
}

// runStandardTunnels is the entrypoint for the Standard Tunnels service
func runStandardTunnels(lc fx.Lifecycle, config Config, logger *logrus.Logger, server tunnel.Server, healthchecks *healthcheckManager) {
	if !config.TunnelStandardEnabled {
		return
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go server.StartStandardTunnels(ctx)
			healthchecks.AddCheck("tunnels_standard", server.CheckStandardTunnels)
			return nil
		},
		OnStop:  func (ctx context.Context) error {
			server.StopStandardTunnels(ctx)
			return nil
		},
	})
}

// runReverseTunnels is the entrypoint for the Reverse Tunnels service
func runReverseTunnels(lc fx.Lifecycle, config Config, logger *logrus.Logger, server tunnel.Server, healthchecks *healthcheckManager) {
	if !config.TunnelReverseEnabled {
		return
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go server.StartReverseTunnels(ctx)
			healthchecks.AddCheck("tunnels_reverse", server.CheckStandardTunnels)
			return nil
		},
		OnStop:  func (ctx context.Context) error {
			server.StopReverseTunnels(ctx)
			return nil
		},
	})
}

func newTunnelServer(config Config, sql *sqlx.DB, stats stats.Stats, discovery discovery.DiscoveryService) (tunnel.Server, error) {
	// Initialize SSH options for reverse tunnels
	var sshOptions tunnel.SSHOptions
	if config.TunnelReverseEnabled {
		 // Decode config host key from Base64
		 hostKey, err := base64.StdEncoding.DecodeString(config.TunnelReverseSSHHostKey)
		 if err != nil {
			 return tunnel.Server{}, errors.Wrap(err, "could not decode host key")
		 }
		 sshOptions.HostKey = hostKey
		 // Set bind host.
		 sshOptions.BindHost = config.TunnelReverseSSHBindHost
	}

	return tunnel.NewServer(
		postgres.NewClient(sql),
		stats.WithPrefix("tunnel"),
		discovery,
		sshOptions,
	), nil
}

func newTunnelDiscoveryService(config Config) (discovery.DiscoveryService, error) {
	var discoveryService discovery.DiscoveryService
	switch config.TunnelServiceDiscoveryType {
	case "srv":
		discoveryService = srv.Discovery{
			SrvRegistry: viper.GetString("tunnel.discovery.registry"),
			Prefix:      viper.GetString("tunnel.discovery.prefix"),
		}
		break

	case "static":
		discoveryService = static.Discovery{
			Host: viper.GetString("tunnel.discovery.host"),
		}
		break

	default:
		return nil, errors.New("unknown service discovery type")
	}
	return discoveryService, nil
}

func newHTTPServer(lc fx.Lifecycle, config Config, logger *logrus.Logger) *mux.Router {
	router := mux.NewRouter()
	server := &http.Server{Addr: config.APIListenAddr, Handler: router}

	// Inject global logger into each request.
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(log.WithLogger(r.Context(), logger)))
		})
	})

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.WithField("addr", server.Addr).Debug("starting HTTP server")
			go server.ListenAndServe()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return server.Shutdown(ctx)
		},
	})

	return router
}

type Config struct {
	Env       string
	LogLevel  string
	LogFormat string

	APIEnabled    bool
	APIListenAddr string

	TunnelServiceDiscoveryType string

	TunnelStandardEnabled bool

	TunnelReverseEnabled     bool
	TunnelReverseSSHBindHost string
	TunnelReverseSSHHostKey  string

	StatsdAddr string
}

func (c Config) Validate() error {
	if !c.APIEnabled && !c.TunnelStandardEnabled && c.TunnelReverseEnabled {
		return errors.New("must enable one of: api, standard, reverse")
	}

	return nil
}

func newConfig() (Config, error) {
	config := Config{
		Env:                        viperConfig.GetString("env"),
		LogLevel:                   viperConfig.GetString("log.level"),
		LogFormat:                  viperConfig.GetString("log.format"),
		APIEnabled:                 viperConfig.GetBool("api.enabled"),
		APIListenAddr:              viperConfig.GetString("api.listenAddr"),
		TunnelServiceDiscoveryType: viperConfig.GetString("tunnel.discovery.type"),
		TunnelStandardEnabled:      viperConfig.GetBool("tunnel.standard.enabled"),
		TunnelReverseEnabled:       viperConfig.GetBool("tunnel.reverse.enabled"),
		TunnelReverseSSHBindHost:   viperConfig.GetString("tunnel.reverse.ssh.bindHost"),
		TunnelReverseSSHHostKey:    viperConfig.GetString("tunnel.reverse.ssh.hostKey"),
		StatsdAddr:                 viperConfig.GetString("statsd.addr"),
	}
	if err := config.Validate(); err != nil {
		return Config{}, err
	}
	return config, nil
}

func newLogger(config Config) *logrus.Logger {
	logger := logrus.New()
	switch config.LogLevel {
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

	switch config.LogFormat {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{})
	default:
		logger.SetFormatter(&logrus.TextFormatter{})
	}
	return logger
}

// newPostgres initializes a connection to the Postgres database
func newPostgres(lc fx.Lifecycle, healthcheck *healthcheckManager) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", getPostgresConnString())
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

func getPostgresConnString() string {
	if viperConfig.IsSet("postgres.uri") {
		return viperConfig.GetString("postgres.uri")
	}

	return formatConnString(map[string]string{
		"host":     viperConfig.GetString("postgres.host"),
		"port":     viperConfig.GetString("postgres.port"),
		"user":     viperConfig.GetString("postgres.user"),
		"password": viperConfig.GetString("postgres.password"),
		"dbname":   viperConfig.GetString("postgres.dbname"),
		"sslmode":  viperConfig.GetString("postgres.sslmode"),
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
func newStats(config Config, logger *logrus.Logger) (stats.Stats, error) {
	var statsdClient statsd.ClientInterface
	if config.StatsdAddr != "" {
		var err error
		statsdClient, err = statsd.New(config.StatsdAddr, statsd.WithMaxBytesPerPayload(4096))
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
			"env":     config.Env,
			"version": version,
		}), nil
}
