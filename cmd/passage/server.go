package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/hightouchio/passage/log"
	"github.com/hightouchio/passage/stats"
	"github.com/hightouchio/passage/tunnel"
	"github.com/hightouchio/passage/tunnel/discovery"
	"github.com/hightouchio/passage/tunnel/keystore"
	"github.com/hightouchio/passage/tunnel/postgres"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"net"
)

var (
	serverCommand = &cobra.Command{
		Use:   "server",
		Short: "passage server is the entrypoint for the HTTP API, the normal tunnel server, and the reverse tunnel server.",
		RunE:  runServer,
	}
)

// runServer boots the API and Tunnel servers
func runServer(cmd *cobra.Command, args []string) error {
	return startApplication(
		// Run telemetry systems
		runTelemetry,

		// Run migrations on application boot.
		runMigrations,

		// Start tunnel.Manager goroutines
		runTunnels,

		// Register control plane HTTP routes.
		registerAPIRoutes,
	)
}

// runTunnels is the entrypoint for tunnel servers
func runTunnels(
	lc fx.Lifecycle,
	server tunnel.API,
	sql *sqlx.DB,
	config *viper.Viper,
	discovery discovery.Service,
	keystore keystore.Keystore,
	healthchecks *healthcheckManager,
	st stats.Stats,
	logger *log.Logger,
	serviceDiscovery discovery.Service,
) error {
	// Helper function for initializing a tunnel.Manager
	runTunnelManager := func(name string, listFunc tunnel.ListFunc) {
		manager := tunnel.NewManager(
			logger.Named("Manager").With("tunnel_type", name),
			st.WithTags(stats.Tags{"tunnel_type": name}),
			listFunc,
			tunnel.TunnelOptions{
				BindHost: config.GetString(ConfigTunnelBindHost),
			},
			config.GetDuration(ConfigTunnelRefreshInterval),
			config.GetDuration(ConfigTunnelRestartInterval),
			serviceDiscovery,
		)

		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				manager.Start()
				healthchecks.AddCheck(fmt.Sprintf("tunnel_%s", name), manager.Check)
				return nil
			},
			OnStop: func(ctx context.Context) error {
				manager.Stop()
				return nil
			},
		})
	}

	if config.GetBool(ConfigTunnelNormalEnabled) {
		runTunnelManager(tunnel.Normal, tunnel.InjectNormalTunnelDependencies(server.GetNormalTunnels, tunnel.NormalTunnelServices{
			SQL:       postgres.NewClient(sql),
			Keystore:  keystore,
			Discovery: discovery,
		}, tunnel.SSHClientOptions{
			User:              config.GetString(ConfigTunnelNormalSshUser),
			DialTimeout:       config.GetDuration(ConfigTunnelNormalDialTimeout),
			KeepaliveInterval: config.GetDuration(ConfigTunnelNormalKeepaliveInterval),
		}))
	}

	if config.GetBool(ConfigTunnelReverseEnabled) {
		hostKey, err := base64.StdEncoding.DecodeString(config.GetString(ConfigTunnelReverseHostKey))
		if err != nil {
			return errors.Wrap(err, "decode host key")
		}

		// Create SSH Server for Reverse Tunnels
		logger := logger.Named("SSHD")
		sshServer := tunnel.NewSSHServer(
			net.JoinHostPort(
				config.GetString(ConfigTunnelReverseBindHost),
				config.GetString(ConfigTunnelReverseSshdPort),
			),
			hostKey,
			logger,
			st,
		)
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				go func() {
					// We want to pass context.Background() here, not the context.Context accepted from the hook,
					//	because the hook's context.Context is cancelled after the application has booted completely
					if err := sshServer.Start(); err != nil {
						if !errors.Is(err, tunnel.ErrSshServerClosed) {
							logger.Errorw("SSH", zap.Error(err))
						}
					}
				}()
				return nil
			},
			OnStop: func(ctx context.Context) error {
				_ = sshServer.Close()
				return nil
			},
		})

		runTunnelManager(tunnel.Reverse, tunnel.InjectReverseTunnelDependencies(server.GetReverseTunnels, tunnel.ReverseTunnelServices{
			SQL:       postgres.NewClient(sql),
			Keystore:  keystore,
			Discovery: discovery,
			SSHServer: sshServer,
		}))
	}

	return nil
}

// registerAPIRoutes attaches the API routes to the router
func registerAPIRoutes(config *viper.Viper, router *mux.Router, tunnelServer tunnel.API) error {
	if !config.GetBool(ConfigApiEnabled) {
		return nil
	}
	tunnelServer.ConfigureWebRoutes(router.PathPrefix("/api").Subrouter())
	return nil
}

// runMigrations executes database migrations
func runMigrations(lc fx.Lifecycle, log *log.Logger, db *sqlx.DB) error {
	logger := log.Named("Migrations")
	logger.Debug("Checking database migrations")

	applied, err := postgres.ApplyMigrations(db.DB)
	if err != nil {
		return errors.Wrap(err, "error running migrations")
	}

	if applied {
		logger.Info("Database migrations applied")
	} else {
		logger.Debug("No database migrations to apply")
	}

	return nil
}
