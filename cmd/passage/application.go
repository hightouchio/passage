package main

import (
	"cloud.google.com/go/storage"
	"context"
	"fmt"
	"github.com/DataDog/datadog-go/statsd"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gorilla/mux"
	"github.com/hightouchio/passage/log"
	"github.com/hightouchio/passage/stats"
	"github.com/hightouchio/passage/tunnel"
	"go.uber.org/zap"
	"net/http/pprof"

	"github.com/hightouchio/passage/tunnel/discovery"
	discoveryConsul "github.com/hightouchio/passage/tunnel/discovery/consul"
	"github.com/hightouchio/passage/tunnel/keystore"
	keystoreGCS "github.com/hightouchio/passage/tunnel/keystore/gcs"
	keystoreInMemory "github.com/hightouchio/passage/tunnel/keystore/in_memory"
	keystorePostgres "github.com/hightouchio/passage/tunnel/keystore/postgres"
	keystoreS3 "github.com/hightouchio/passage/tunnel/keystore/s3"

	consul "github.com/hashicorp/consul/api"

	"github.com/hightouchio/passage/tunnel/postgres"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"go.uber.org/dig"
	"go.uber.org/fx"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	ConfigEnv          = "env"
	ConfigHTTPAddr     = "http.addr"
	ConfigApiEnabled   = "api.enabled"
	ConfigPprofEnabled = "pprof.enabled"

	ConfigTunnelBindHost        = "tunnel.bind_host"
	ConfigTunnelRefreshInterval = "tunnel.refresh_interval"
	ConfigTunnelRestartInterval = "tunnel.restart_interval"

	ConfigTunnelNormalEnabled           = "tunnel.normal.enabled"
	ConfigTunnelNormalSshUser           = "tunnel.normal.ssh_user"
	ConfigTunnelNormalDialTimeout       = "tunnel.normal.dial.timeout"
	ConfigTunnelNormalKeepaliveInterval = "tunnel.normal.keepalive_interval"

	ConfigTunnelReverseEnabled              = "tunnel.reverse.enabled"
	ConfigTunnelReverseHostKey              = "tunnel.reverse.host_key"
	ConfigTunnelReverseBindHost             = "tunnel.reverse.bind_host"
	ConfigTunnelReverseSshdPort             = "tunnel.reverse.sshd_port"
	ConfigTunnelReverseEnableIndividualSSHD = "tunnel.reverse.enable_individual_sshd"

	ConfigDiscoveryType           = "discovery.type"
	ConfigDiscoverySrvRegistry    = "discovery.srv.registry"
	ConfigDiscoverySrvPrefix      = "discovery.srv.prefix"
	ConfigDiscoveryStaticHost     = "discovery.static.host"
	ConfigDiscoveryDnsHostNormal  = "discovery.dns.host_normal"
	ConfigDiscoveryDnsHostReverse = "discovery.dns.host_reverse"

	ConfigKeystoreType              = "keystore.type"
	ConfigKeystorePostgresTableName = "keystore.postgres.table_name"

	ConfigKeystoreS3BucketName     = "keystore.s3.bucket_name"
	ConfigKeystoreS3KeyPrefix      = "keystore.s3.key_prefix"
	ConfigKeystoreS3Endpoint       = "keystore.s3.endpoint"
	ConfigKeystoreS3DisableSSL     = "keystore.s3.disable_ssl"
	ConfigKeystoreS3ForcePathStyle = "keystore.s3.force_path_style"
	ConfigKeystoreS3BucketRegion   = "keystore.s3.bucket_region"

	ConfigKeystoreGCSBucketName = "keystore.gcs.bucket_name"
	ConfigKeystoreGCSKeyPrefix  = "keystore.gcs.key_prefix"

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
	config.SetDefault(ConfigTunnelNormalSshUser, "passage")
	config.SetDefault(ConfigTunnelNormalDialTimeout, 15*time.Second)
	config.SetDefault(ConfigTunnelNormalKeepaliveInterval, 1*time.Minute)
	config.SetDefault(ConfigTunnelReverseBindHost, "0.0.0.0")
	config.SetDefault(ConfigTunnelReverseSshdPort, 22)
	config.SetDefault(ConfigTunnelReverseEnableIndividualSSHD, true)
	config.SetDefault(ConfigDiscoveryType, "static")
	config.SetDefault(ConfigDiscoveryStaticHost, "localhost")
	config.SetDefault(ConfigKeystoreType, "in-memory")
	config.SetDefault(ConfigLogLevel, "info")
	config.SetDefault(ConfigLogFormat, "text")
}

// startApplication boots the application dependency injection framework and executes the bootFuncs
func startApplication(bootFuncs ...interface{}) error {
	app := fx.New(
		// Define dependencies.
		fx.Provide(
			// Control plane API.
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

		// Execute entrypoint functions.
		fx.Invoke(bootFuncs...),

		fx.NopLogger,
	)

	startCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	go func() {
		if err := app.Start(startCtx); err != nil {
			switch v := dig.RootCause(err).(type) {
			case configError:
				log.Get().Fatalf("Config error: %v", v)
			default:
				log.Get().Fatalf("Startup error: %v", v)
			}
		}

		log.Get().Named("Passage").Infow("Start", zap.String("version", version))
	}()

	<-app.Done()

	log.Get().Named("Passage").Infow("Stop")

	stopCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := app.Stop(stopCtx); err != nil {
		log.Get().Fatalf("Shutdown error: %v", dig.RootCause(err))
	}

	return nil
}

func newTunnelAPI(sql *sqlx.DB, stats stats.Stats, keystore keystore.Keystore, discovery discovery.DiscoveryService) (tunnel.API, error) {
	return tunnel.API{
		SQL:              postgres.NewClient(sql),
		DiscoveryService: discovery,
		Keystore:         keystore,
		Stats:            stats,
	}, nil
}

func newTunnelDiscoveryService(config *viper.Viper) (discovery.DiscoveryService, error) {
	var discoveryService discovery.DiscoveryService
	switch config.GetString(ConfigDiscoveryType) {
	case "consul":
		consulApi, err := consul.NewClient(consul.DefaultConfig())
		if err != nil {
			return nil, errors.Wrap(err, "could not init Consul client")
		}
		discoveryService = discoveryConsul.Discovery{
			Consul:         consulApi,
			HostAddress:    "127.0.0.1", // TODO: Drive off of Pod IP
			HealthcheckTTL: 30 * time.Second,
		}

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
	case "in-memory":
		return keystoreInMemory.New(), nil

	case "postgres":
		tableName := config.GetString(ConfigKeystorePostgresTableName)
		if tableName == "" {
			return nil, newConfigError(ConfigKeystorePostgresTableName, "must be set")
		}
		return keystorePostgres.New(db, tableName), nil

	case "s3":
		bucketName := config.GetString(ConfigKeystoreS3BucketName)
		if bucketName == "" {
			return nil, newConfigError(ConfigKeystoreS3BucketName, "must be set")
		}

		// Configure AWS session
		awsConfig := &aws.Config{}
		if config.IsSet(ConfigKeystoreS3BucketRegion) {
			awsConfig.Region = aws.String(config.GetString(ConfigKeystoreS3BucketRegion))
		}
		if config.IsSet(ConfigKeystoreS3Endpoint) {
			awsConfig.Endpoint = aws.String(config.GetString(ConfigKeystoreS3Endpoint))
		}
		if config.IsSet(ConfigKeystoreS3DisableSSL) {
			awsConfig.DisableSSL = aws.Bool(config.GetBool(ConfigKeystoreS3DisableSSL))
		}
		if config.IsSet(ConfigKeystoreS3ForcePathStyle) {
			awsConfig.S3ForcePathStyle = aws.Bool(config.GetBool(ConfigKeystoreS3ForcePathStyle))
		}
		sess, err := session.NewSession(awsConfig)
		if err != nil {
			return nil, configError{"could not init aws session"}
		}

		// Init S3 keystore
		return keystoreS3.S3{
			S3:         s3.New(sess),
			BucketName: bucketName,
			KeyPrefix:  config.GetString(ConfigKeystoreS3KeyPrefix),
		}, nil

	case "gcs":
		bucketName := config.GetString(ConfigKeystoreGCSBucketName)
		if bucketName == "" {
			return nil, newConfigError(ConfigKeystoreGCSBucketName, "must be set")
		}

		client, err := storage.NewClient(context.Background())
		if err != nil {
			return nil, errors.Wrap(err, "could not init GCS client")
		}

		return keystoreGCS.GCS{
			Client:     client,
			BucketName: bucketName,
			KeyPrefix:  config.GetString(ConfigKeystoreGCSKeyPrefix),
		}, nil

	default:
		return nil, configError{fmt.Sprintf("unsupported keystore type: %s", keystoreType)}
	}
}

func newHTTPServer(lc fx.Lifecycle, config *viper.Viper, log *log.Logger) *mux.Router {
	router := mux.NewRouter()
	server := &http.Server{Addr: config.GetString(ConfigHTTPAddr), Handler: router}

	logger := log.Named("HTTP")

	// Log every request.
	router.Use(LoggingMiddleware(logger))

	// Conditionally enable pprof profiling
	if config.GetBool(ConfigPprofEnabled) {
		router.PathPrefix("/debug/pprof/").HandlerFunc(pprof.Index)
		router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		router.HandleFunc("/debug/pprof/profile", pprof.Profile)
		router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		router.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Start")
			go func() {
				if err := server.ListenAndServe(); err != nil {
					logger.Errorw("HTTP Listener", zap.Error(err))
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

func newLogger(config *viper.Viper) *log.Logger {
	// Init new zap
	log.Init(config.GetString(ConfigLogLevel), config.GetString(ConfigLogFormat))
	return log.Get()
}

// newPostgres initializes a connection to the Postgres database
func newPostgres(lc fx.Lifecycle, config *viper.Viper) (*sqlx.DB, error) {
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

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Ping database.
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
func newStats(config *viper.Viper) (stats.Stats, error) {
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
	st := stats.New(statsdClient)
	if version != "" {
		st = st.WithTags(stats.Tags{"version": version})
	}
	return st, nil
}
