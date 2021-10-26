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
	"strings"
	"syscall"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/gorilla/mux"
	"github.com/hightouchio/passage/tunnel"
	"github.com/hightouchio/passage/tunnel/postgres"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	viperConfig   = viper.New()
	serverCommand = &cobra.Command{
		Use:   "server",
		Short: "run the passage server",
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

	viperConfig.SetDefault("api.enabled", false)
	viperConfig.SetDefault("api.listenAddr", ":8080")

	viperConfig.SetDefault("tunnel.reverse.enabled", false)
	viperConfig.SetDefault("tunnel.reverse.ssh.bindHost", "localhost")

	viperConfig.SetDefault("tunnel.normal.enabled", false)
}

func init() {
	serverCommand.Flags().Bool("api", false, "run API server")
	viperConfig.BindPFlag("api.enabled", serverCommand.Flags().Lookup("api"))

	serverCommand.Flags().Bool("normal", false, "run normal tunnel server")
	viperConfig.BindPFlag("tunnel.normal.enabled", serverCommand.Flags().Lookup("normal"))

	serverCommand.Flags().Bool("reverse", false, "run reverse tunnel server")
	viperConfig.BindPFlag("tunnel.reverse.enabled", serverCommand.Flags().Lookup("reverse"))
}

type Config struct {
	Env       string
	LogLevel  string
	LogFormat string

	APIEnabled    bool
	APIListenAddr string

	TunnelReverseEnabled     bool
	TunnelReverseSSHBindHost string
	TunnelReverseSSHHostKey  string

	TunnelNormalEnabled bool

	StatsdAddr string
}

func getServerConfig() Config {
	return Config{
		Env:                      viperConfig.GetString("env"),
		LogLevel:                 viperConfig.GetString("log.level"),
		LogFormat:                viperConfig.GetString("log.format"),
		APIEnabled:               viperConfig.GetBool("api.enabled"),
		APIListenAddr:            viperConfig.GetString("api.listenAddr"),
		TunnelReverseEnabled:     viperConfig.GetBool("tunnel.reverse.enabled"),
		TunnelReverseSSHBindHost: viperConfig.GetString("tunnel.reverse.ssh.bindHost"),
		TunnelReverseSSHHostKey:  viperConfig.GetString("tunnel.reverse.ssh.hostKey"),
		TunnelNormalEnabled:      viperConfig.GetBool("tunnel.normal.enabled"),
		StatsdAddr:               viperConfig.GetString("statsd.addr"),
	}
}

func (c Config) Validate() error {
	if !c.APIEnabled && !c.TunnelNormalEnabled && c.TunnelReverseEnabled {
		return errors.New("must enable one of: api, normal, reverse")
	}

	if c.APIEnabled {
		if c.APIListenAddr == "" {
			return errors.New("must set api.listenAddr")
		}
	}

	if c.TunnelReverseEnabled {
		if c.TunnelReverseSSHBindHost == "" {
			return errors.New("must set tunnel.reverse.ssh.bindHost")
		}
		if c.TunnelReverseSSHHostKey == "" {
			return errors.New("must set tunnel.reverse.ssh.hostKey")
		}
	}

	return nil
}

func runServer(cmd *cobra.Command, args []string) error {
	serverConfig := getServerConfig()
	if err := serverConfig.Validate(); err != nil {
		return errors.Wrap(err, "error validating config")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	logger := logrus.New()
	switch serverConfig.LogLevel {
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

	switch serverConfig.LogFormat {
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
	if serverConfig.StatsdAddr!= "" {
		var err error
		statsdClient, err = statsd.New(serverConfig.StatsdAddr, statsd.WithMaxBytesPerPayload(4096))
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
			"env":     serverConfig.Env,
			"version": version,
		})

	// configure web server
	router := mux.NewRouter()
	router.Handle("/healthcheck", healthchecks)
	// inject global logger into request.
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(log.WithLogger(r.Context(), logger)))
		})
	})

	// Initialize SSH options for reverse tunnels
	var sshOptions tunnel.SSHOptions
	if serverConfig.TunnelReverseEnabled {
		// Decode config host key from Base64
		hostKey, err := base64.StdEncoding.DecodeString(serverConfig.TunnelReverseSSHHostKey)
		if err != nil {
			return errors.Wrap(err, "could not decode host key")
		}
		sshOptions.HostKey = hostKey
		// Set bind host.
		sshOptions.BindHost = serverConfig.TunnelReverseSSHBindHost
	}

	// Configure tunnel server
	tunnelServer := tunnel.NewServer(postgres.NewClient(db), statsClient.WithPrefix("tunnel"), sshOptions)

	if serverConfig.TunnelNormalEnabled {
		go tunnelServer.StartNormalTunnels(ctx)
		healthchecks.AddCheck("normal_tunnels", tunnelServer.CheckNormalTunnels)
	}

	if serverConfig.TunnelReverseEnabled {
		go tunnelServer.StartReverseTunnels(ctx)
		healthchecks.AddCheck("reverse_tunnels", tunnelServer.CheckReverseTunnels)
	}

	if serverConfig.APIEnabled {
		tunnelServer.ConfigureWebRoutes(router.PathPrefix("/api").Subrouter())

		// start HTTP server
		httpServer := &http.Server{Addr: serverConfig.APIListenAddr, Handler: router}
		go func() {
			logger.WithField("http_addr", serverConfig.APIListenAddr).Debug("starting http server")
			if err := httpServer.ListenAndServe(); err != nil {
				logger.WithError(err).Fatal("http server shutdown")
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
