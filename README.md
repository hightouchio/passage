# passage
[![Build](https://github.com/hightouchio/passage/actions/workflows/build.yml/badge.svg?branch=master)](https://github.com/hightouchio/passage/actions/workflows/build.yml)
![Go Version](https://shields.io/github/go-mod/go-version/hightouchio/passage)
[![GitHub Releases](https://shields.io/github/v/release/hightouchio/passage?display_name=tag&include_prereleases)](https://github.com/hightouchio/passage/releases)

Secure private tunnels as a service 🔐

## Get Started
Use Docker Compose to easily spin up a local Passage installation that you can use to try it out.
```bash 
$ docker compose up
```

## What is passage?
passage is a service for programmatically creating and managing [SSH tunnels](https://www.ssh.com/academy/ssh/tunneling). The primary use case is to serve as a secure bridge between SaaS providers and services that need to be accessed within customer networks. Passage acts both as a management API, and as a daemon which maintains the tunnels themselves. 

### Normal vs Reverse
With **Normal** tunnels, Passage acts as an SSH client, opening an SSH connection to an internet-facing remote bastion server, then from there opening an upstream connection to a private service within the remote network.

With **Reverse** tunnels, Passage acts as an SSH server, allowing remote clients to forward a local port from a hidden server to a dedicated port on the Passage instance, therefore achieving a tunnel without requiring a remote bastion server to be exposed to inbound traffic from the internet.

## Dependencies
- Postgres 11 or later
- A keystore to securely store and retrieve public and private keys.
  - Passage supports Postgres or S3 (default is a table in the same Postgres database)
  - Passage does not handle encryption of keys at rest.

## Configuration
Passage can read its configuration from disk (YAML or JSON), or from environment variables.
Passage's config keys are paths in a configuration object, so the dot notation is used to indicate hierarchy.

To use environment variables, replace the dot with an underscore and prefix the variable with `PASSAGE_`.
For example, `tunnel.normal.ssh_user` becomes `PASSAGE_TUNNEL_NORMAL_SSH_USER`.

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

## Deployment
passage exposes parts of your private network to the Internet. Therefore, its important to secure the network that passage operates in.

Passage opens two kinds of ports on machines that it runs on: (1) tunnel ports, and (2) `sshd` ports. Tunnel ports forward packets to the exposed, remote customer port (what your services will talk to).
`sshd` ports are used for reverse tunnels. These ports host SSH servers that remote clients will connect to with their remote port forwarding requests. These ports need to be exposed to the public Internet. 

An appropriate network configuration begins with Passage instances completely locked down. The following ingress openings should be made in your firewall: 
1. Expose tunnel ports to internal services
   1. Port range `49152 - 57343`
2. Expose `sshd` ports to public internet
   1. Port range `57344 - 65535`
3. Of course, any other ingress you need (load balancers, internal tools, etc.)

## Testing
Go unit tests can be run with `make test`.

There is an end-to-end test of both Normal and Reverse tunnels, using Docker networks to simulate network isolation, and docker compose for orchestration, that can be run with `make test-e2e`.


protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative tunnel/grpc/tunnel.proto