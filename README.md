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
| ENV                                | Name of the env for statsd reporting                                    | False                     |              |             |
| HTTP_ADDR                          | Bind address for the HTTP server                                        | True                      | 0.0.0.0:8080 |             |
| API_ENABLED                        | Expose Tunnel management APIs via HTTP                                  | False                     | False        |             |
| TUNNEL_BIND_HOST                   | Bind host for internal tunnel ports.                                    | False                     | 0.0.0.0      |             |
| TUNNEL_REFRESH_INTERVAL            | How frequently Passage should check Postgres for tunnel status changes. | False                     | 1 second     |             |
| TUNNEL_RESTART_INTERVAL            | How frequently Passage should attempt to restart a broken tunnel.       | False                     | 15 seconds   |             |

### Standard Tunneling
| **Key**                                    | **Description**                                                         | **Required**              | **Default**  | **Aliases** |
|--------------------------------------------|-------------------------------------------------------------------------|---------------------------|--------------|-------------|
| TUNNEL_STANDARD_ENABLED            | Enable Standard Tunnels.                                                | False                     | False        |             |
| TUNNEL_STANDARD_SSH_USER           | SSH client username for standard tunnels.                               | False                     | `passage`    |             |
| TUNNEL_STANDARD_DIAL_TIMEOUT       | Timeout for initial SSH dial.                                           | False                     | 15 seconds   |             |
| TUNNEL_STANDARD_KEEPALIVE_INTERVAL | Keepalive interval for Standard Tunnel SSH client connection.           | False                     | 1 minute     |             |
| TUNNEL_STANDARD_KEEPALIVE_TIMEOUT  | Keepalive timeout for Standard Tunnel SSH client connection.            | False                     | 15 seconds   |             |

### Reverse Tunneling
| **Key**                                    | **Description**                                                         | **Required**              | **Default**  | **Aliases** |
|--------------------------------------------|-------------------------------------------------------------------------|---------------------------|--------------|-------------|
| TUNNEL_REVERSE_ENABLED             | Enable Reverse Tunnels.                                                 | False                     | False        |             |
| TUNNEL_REVERSE_HOST_KEY            | Base64 encoded host key for the reverse tunnel SSH server.              | True, if reverse enabled. |              |             |
| TUNNEL_REVERSE_BIND_HOST           | Bind host for the reverse tunnel SSH server                             | True, if reverse enabled. |              |             |

### Service Discovery
| **Key**                                    | **Description**                                                         | **Required**              | **Default**  | **Aliases** |
|--------------------------------------------|-------------------------------------------------------------------------|---------------------------|--------------|-------------|
| DISCOVERY_TYPE                     | Tunnel service discovery type (`static` or `srv`)                       | False                     | `static`     |             |
| DISCOVERY_SRV_REGISTRY             | If `srv`, the DNS SRV registry to use.                                  | True, if `srv`.           |              |             |
| DISCOVERY_SRV_PREFIX               | TODO                                                                    | True, if `srv`.           |              |             |
| DISCOVERY_STATIC_HOST              | If `static`, the hostname to use.                                       | True, if `static`.        |              |             |

### Keystore
| **Key**                                    | **Description**                                                         | **Required**              | **Default**  | **Aliases** |
|--------------------------------------------|-------------------------------------------------------------------------|---------------------------|--------------|-------------|
| KEYSTORE_TYPE                      | Tunnel keystore type (`postgres` or `s3`)                               | True                      |              |             |
| KEYSTORE_POSTGRES_TABLE_NAME       | If `postgres`, the table name to use.                                   | True, if `postgres`       |              |             |
| KEYSTORE_S3_BUCKET_NAME            | If `s3`, the bucket name to use.                                        | True, if `s3`             |              |             |
| KEYSTORE_S3_KEY_PREFIX             | If `s3`, the prefix applied to keys.                                    | False                     |              |             |

### Database Connection
| **Key**                                    | **Description**                                                         | **Required**              | **Default**  | **Aliases** |
|--------------------------------------------|-------------------------------------------------------------------------|---------------------------|--------------|-------------|
| POSTGRES_URI                       | Postgres connection string.                                             | False                     |              |             |
| POSTGRES_HOST                      | See `PGHOST`                                                            | True                      |              |             |
| POSTGRES_PORT                      | See `PGPORT`                                                            | True                      |              |             |
| POSTGRES_USER                      | See `PGUSER`                                                            | True                      |              |             |
| POSTGRES_PASS                      | See `PGPASS`                                                            | True                      |              |             |
| POSTGRES_DBNAME                    | See `PGDBNAME`                                                          | True                      |              |             |
| POSTGRES_SSLMODE                   | See `PGSSLMODE`                                                         | True                      |              |             |

### Visibility
| **Key**                                    | **Description**                                                         | **Required**              | **Default**  | **Aliases** |
|--------------------------------------------|-------------------------------------------------------------------------|---------------------------|--------------|-------------|
| LOG_LEVEL                          | Visibility level for logs (debug/info/warn/error/fatal)                 | False                     | `info`       |             |
| LOG_FORMAT                         | Format of structured logs (json/text)                                   | False                     | `text`       |             |
| STATSD_ADDR                        | Address of a Statsd server to send metrics to.                          | False                     |              |             |
