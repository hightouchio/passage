package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/hightouchio/passage/stats"
	"github.com/hightouchio/passage/tunnel"
	"github.com/hightouchio/passage/tunnel/keystore"
	"github.com/hightouchio/passage/tunnel/postgres"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/fx"
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
		// Run migrations on application boot.
		runMigrations,

		// Start tunnel.Manager goroutines
		runTunnels,

		// Register control plane HTTP routes.
		registerAPIRoutes,
	)
}

// runTunnels is the entrypoint for tunnel servers
func runTunnels(lc fx.Lifecycle, server tunnel.API, sql *sqlx.DB, config *viper.Viper, keystore keystore.Keystore, healthchecks *healthcheckManager, st stats.Stats, logger *logrus.Logger) error {
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

	if config.GetBool(ConfigTunnelNormalEnabled) {
		runTunnelManager(tunnel.Normal, tunnel.InjectNormalTunnelDependencies(server.GetNormalTunnels, tunnel.NormalTunnelServices{
			SQL:      postgres.NewClient(sql),
			Keystore: keystore,
			Logger:   logger,
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
		sshServer := tunnel.NewSSHServer(
			net.JoinHostPort(
				config.GetString(ConfigTunnelReverseBindHost),
				config.GetString(ConfigTunnelReverseSshdPort),
			),
			hostKey,
			logger,
		)
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				go func() {
					// We want to pass context.Background() here, not the context.Context accepted from the hook,
					//	because the hook's context.Context is cancelled after the application has booted completely
					if err := sshServer.Start(context.Background()); err != nil {
						logger.Fatal(errors.Wrap(err, "sshd"))
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
			SQL:      postgres.NewClient(sql),
			Keystore: keystore,
			Logger:   logger,

			GlobalSSHServer: sshServer,
			GetSSHServer: func(sshdPort int) *tunnel.SSHServer {
				return tunnel.NewSSHServer(
					net.JoinHostPort(config.GetString(ConfigTunnelReverseBindHost), fmt.Sprintf("%d", sshdPort)),
					hostKey,
					logger,
				)
			},
		}))
	}

	return nil
}

// registerAPIRoutes attaches the API routes to the router
func registerAPIRoutes(config *viper.Viper, logger *logrus.Logger, router *mux.Router, tunnelServer tunnel.API) error {
	if !config.GetBool(ConfigApiEnabled) {
		return nil
	}
	tunnelServer.ConfigureWebRoutes(router.PathPrefix("/api").Subrouter())
	return nil
}

// runMigrations executes database migrations
func runMigrations(lc fx.Lifecycle, logger *logrus.Logger, db *sqlx.DB) error {
	logger.Debug("checking database migrations")
	applied, err := postgres.ApplyMigrations(db.DB)
	if err != nil {
		return errors.Wrap(err, "error running migrations")
	}

	if applied {
		logger.Info("database migrations applied")
	} else {
		logger.Debug("no database migrations to apply")
	}

	return nil
}
