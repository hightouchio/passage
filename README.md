# passage
[![CircleCI](https://circleci.com/gh/hightouchio/passage/tree/master.svg?style=svg)](https://circleci.com/gh/hightouchio/passage/tree/master)
[![License](https://shields.io/github/license/hightouchio/passage)](https://github.com/hightouchio/passage/blob/master/LICENSE)
![Go Version](https://shields.io/github/go-mod/go-version/hightouchio/passage)
[![GitHub Releases](https://shields.io/github/v/release/hightouchio/passage?display_name=tag)](https://github.com/hightouchio/passage/releases)
[![Docker Hub](https://shields.io/docker/v/hightouchio/passage)](https://hub.docker.com/r/hightouchio/passage)

passage is a utility for programmatically creating and managing SSH tunnels. The primary use case is to serve as a secure bridge between SaaS providers and resources that need to be accessed within customer environments. Passage acts as both a management API, as well as a daemon to manage the tunnels themselves.

With **Standard** tunnels, Passage acts as an SSH client, opening an SSH connection to an internet-facing remote bastion server, then from there opening an upstream connection to a private service within the remote network.

With **Reverse** tunnels, Passage acts as an SSH server, allowing remote clients to forward a local port from a hidden server to a dedicated port on the Passage instance, therefore achieving a tunnel without requiring a remote bastion server to be exposed to inbound traffic from the internet.

## Usage
Passage is primarily started through the subcommand `passage server`.

```
Usage:
  passage server [flags]

Flags:
  -h, --help      help for server
```

## Dependencies
Passage requires a PostgreSQL database, version 11 or later. Database schema is located in [`sql/1-schema.sql`](`sql/1-schema.sql`).

## Configuration
Passage can read its configuration from disk (YAML or JSON), or from environment variables.
Passage's config keys are paths in a configuration object, so the dot notation is used to indicate hierarchy.

To use environment variables, replace the dot with an underscore and prefix the variable with `PASSAGE_`.
For example, `tunnel.standard.ssh_user` becomes `PASSAGE_TUNNEL_STANDARD_SSH_USER`.

The following are required config options that you should set to quickstart with Passage. A full configuration reference is available at [docs/config.md](docs/config.md). 

| **Key**          | **Description**             | **Required** | **Alias**   |
|------------------|-----------------------------|--------------|-------------|
| tunnel.bind.host        | Bind host for internal tunnel ports.                                    | True         | 0.0.0.0      |
| tunnel.reverse.host.key  | Base64 encoded [host key](https://www.ssh.com/academy/ssh/host-key) for the reverse tunnel SSH server. | True, if reverse enabled. |             |
| postgres.uri     | Postgres connection string. | False        |             |
| postgres.host    | See `PGHOST`                | True         | `PGHOST`    |
| postgres.port    | See `PGPORT`                | True         | `PGPORT`    |
| postgres.user    | See `PGUSER`                | True         | `PGUSER`    |
| postgres.pass    | See `PGPASS`                | True         | `PGPASS`    |
| postgres.dbname  | See `PGDBNAME`              | True         | `PGDBNAME`  |
| postgres.sslmode | See `PGSSLMODE`             | True         | `PGSSLMODE` |
