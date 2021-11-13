# passage
[![CircleCI](https://circleci.com/gh/hightouchio/passage/tree/master.svg?style=svg)](https://circleci.com/gh/hightouchio/passage/tree/master)

passage is a utility for programmatically creating and managing SSH tunnels, both standard and reverse. The chief use case is to serve as a secure bridge between SaaS providers and resources that need to be accessed within customer environments. Passage acts as both a management API, as well as a daemon to manage the tunnels themselves.

With **Standard** tunnels, Passage acts as an SSH client, opening an SSH connection to an internet-facing remote bastion server, then from there opening an upstream connection to a private service within the remote network.

With **Reverse** tunnels, Passage acts as an SSH server, allowing remote clients to forward a local port from a hidden server to a dedicated port on the Passage instance, therefore achieving a tunnel without requiring a remote bastion server to be exposed to the internet.

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
Passage is configured primarily through environment variables, as listed below.

| **Key**                                    | **Description**                                                         | **Required**              | **Default**  | **Aliases** |
|--------------------------------------------|-------------------------------------------------------------------------|---------------------------|--------------|-------------|
| PASSAGE_ENV                                | Name of the env for statsd reporting                                    | False                     |              |             |
| PASSAGE_HTTP_ADDR                          | Bind address for the HTTP server                                        | True                      | 0.0.0.0:8080 |             |
| PASSAGE_API_ENABLED                        | Expose Tunnel management APIs via HTTP                                  | False                     | False        |             |
| PASSAGE_TUNNEL_STANDARD_ENABLED            | Enable the standard tunnel server                                       | False                     | False        |             |
| PASSAGE_TUNNEL_BIND_HOST                   | Bind host for internal tunnel ports.                                    | False                     | 0.0.0.0      |             |
| PASSAGE_TUNNEL_REFRESH_INTERVAL            | How frequently Passage should check Postgres for tunnel status changes. | False                     | 1 second     |             |
| PASSAGE_TUNNEL_RESTART_INTERVAL            | How frequently Passage should attempt to restart a broken tunnel.       | False                     | 15 seconds   |             |
| PASSAGE_TUNNEL_STANDARD_ENABLED            | Enable Standard Tunnels.                                                | False                     | False        |             |
| PASSAGE_TUNNEL_STANDARD_SSH_USER           | SSH client username for standard tunnels.                               | False                     | `passage`    |             |
| PASSAGE_TUNNEL_STANDARD_DIAL_TIMEOUT       | Timeout for initial SSH dial.                                           | False                     | 15 seconds   |             |
| PASSAGE_TUNNEL_STANDARD_KEEPALIVE_INTERVAL | Keepalive interval for Standard Tunnel SSH client connection.           | False                     | 1 minute     |             |
| PASSAGE_TUNNEL_STANDARD_KEEPALIVE_TIMEOUT  | Keepalive timeout for Standard Tunnel SSH client connection.            | False                     | 15 seconds   |             |
| PASSAGE_TUNNEL_REVERSE_ENABLED             | Enable Reverse Tunnels.                                                 | False                     | False        |             |
| PASSAGE_TUNNEL_REVERSE_HOST_KEY            | Base64 encoded host key for the reverse tunnel SSH server.              | True, if reverse enabled. |              |             |
| PASSAGE_TUNNEL_REVERSE_BIND_HOST           | Bind host for the reverse tunnel SSH server                             | True, if reverse enabled. |              |             |
| PASSAGE_DISCOVERY_TYPE                     | Tunnel service discovery type (`static` or `srv`)                       | False                     | `static`     |             |
| PASSAGE_DISCOVERY_SRV_REGISTRY             | If `srv`, the DNS SRV registry to use.                                  | True, if `srv`.           |              |             |
| PASSAGE_DISCOVERY_SRV_PREFIX               | TODO                                                                    | True, if `srv`.           |              |             |
| PASSAGE_DISCOVERY_STATIC_HOST              | If `static`, the hostname to use.                                       | True, if `static`.        |              |             |
| PASSAGE_KEYSTORE_TYPE                      | Tunnel keystore type (`postgres` or `s3`)                               | True                      |              |             |
| PASSAGE_KEYSTORE_POSTGRES_TABLE_NAME       | If `postgres`, the table name to use.                                   | True, if `postgres`       |              |             |
| PASSAGE_KEYSTORE_S3_BUCKET_NAME            | If `s3`, the bucket name to use.                                        | True, if `s3`             |              |             |
| PASSAGE_KEYSTORE_S3_KEY_PREFIX             | If `s3`, the prefix applied to keys.                                    | False                     |              |             |
| PASSAGE_POSTGRES_URI                       | Postgres connection string.                                             | False                     |              |             |
| PASSAGE_POSTGRES_HOST                      | See `PGHOST`                                                            | True                      |              |             |
| PASSAGE_POSTGRES_PORT                      | See `PGPORT`                                                            | True                      |              |             |
| PASSAGE_POSTGRES_USER                      | See `PGUSER`                                                            | True                      |              |             |
| PASSAGE_POSTGRES_PASS                      | See `PGPASS`                                                            | True                      |              |             |
| PASSAGE_POSTGRES_DBNAME                    | See `PGDBNAME`                                                          | True                      |              |             |
| PASSAGE_POSTGRES_SSLMODE                   | See `PGSSLMODE`                                                         | True                      |              |             |
| PASSAGE_LOG_LEVEL                          | Visibility level for logs (debug/info/warn/error/fatal)                 | False                     | `info`       |             |
| PASSAGE_LOG_FORMAT                         | Format of structured logs (json/text)                                   | False                     | `text`       |             |
| PASSAGE_STATSD_ADDR                        | Address of a Statsd server to send metrics to.                          | False                     |              |             |
