# passage
[![CircleCI](https://circleci.com/gh/hightouchio/passage/tree/master.svg?style=svg)](https://circleci.com/gh/hightouchio/passage/tree/master)
[![License](https://shields.io/github/license/hightouchio/passage)](https://github.com/hightouchio/passage/blob/master/LICENSE)
![Go Version](https://shields.io/github/go-mod/go-version/hightouchio/passage)
[![GitHub Releases](https://shields.io/github/v/release/hightouchio/passage?display_name=tag)](https://github.com/hightouchio/passage/releases)
[![Docker Hub](https://shields.io/docker/v/hightouchio/passage)](https://hub.docker.com/r/hightouchio/passage)

Secure private tunnels as a service 🔐

## Get Started
```bash 
$ docker compose up
$ curl -X POST -d '{"sshHost":"localhost"}' http://localhost:6000/api/tunnel/standard  
```

passage is primarily started through the subcommand `passage server`.
```
Usage:
  passage server [flags]

Flags:
  -h, --help      help for server
```

## What is passage?
passage is a service for programmatically creating and managing SSH tunnels. The primary use case is to serve as a secure bridge between SaaS providers and services that need to be accessed within customer networks. Passage acts both as a management API, and as a daemon which maintains the tunnels themselves. 

### Standard vs Reverse
With **Standard** tunnels, Passage acts as an SSH client, opening an SSH connection to an internet-facing remote bastion server, then from there opening an upstream connection to a private service within the remote network.

With **Reverse** tunnels, Passage acts as an SSH server, allowing remote clients to forward a local port from a hidden server to a dedicated port on the Passage instance, therefore achieving a tunnel without requiring a remote bastion server to be exposed to inbound traffic from the internet.

## Dependencies
- Postgres 11 or later ([`schema`](`sql/1-schema.sql`))
- A keystore to securely store and retrieve public and private keys.
  - Passage supports Postgres or S3 (default is a table in the same Postgres database)
  - Passage does not handle encryption of keys at rest.

## Configuration
Passage can read its configuration from disk (YAML or JSON), or from environment variables.
Passage's config keys are paths in a configuration object, so the dot notation is used to indicate hierarchy.

To use environment variables, replace the dot with an underscore and prefix the variable with `PASSAGE_`.
For example, `tunnel.standard.ssh_user` becomes `PASSAGE_TUNNEL_STANDARD_SSH_USER`.

The following are required config options that you must set to quickstart with Passage. A full configuration reference is available at [docs/config.md](docs/config.md). 

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
