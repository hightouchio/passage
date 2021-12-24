# Config Reference
## General
| **Key**                 | **Description**                                                         | **Required** | **Default**  |
|-------------------------|-------------------------------------------------------------------------|--------------|--------------|
| http.addr               | Bind address for the HTTP server                                        | True         | 0.0.0.0:8080 |
| api.enabled             | Expose Tunnel management APIs via HTTP                                  | True         | True         |
| tunnel.bind.host        | Bind host for internal tunnel ports.                                    | True         | 0.0.0.0      |
| tunnel.refresh.interval | How frequently Passage should check Postgres for tunnel status changes. | False        | 1 second     |
| tunnel.restart.interval | How frequently Passage should attempt to restart a broken tunnel.       | False        | 15 seconds   |

## Normal Tunnels
| **Key**                            | **Description**                                               | **Required** | **Default** |
|------------------------------------|---------------------------------------------------------------|--------------|-------------|
| tunnel.normal.enabled            | Enable normal Tunnels.                                      | True         | True        |
| tunnel.normal.ssh.user           | SSH client username for normal tunnels.                     | True         | `passage`   |
| tunnel.normal.dial.timeout       | Timeout for initial SSH dial.                                 | False        | 15 seconds  |
| tunnel.normal.keepalive.interval | Keepalive interval for normal Tunnel SSH client connection. | False        | 1 minute    |
| tunnel.normal.keepalive.timeout  | Keepalive timeout for normal Tunnel SSH client connection.  | False        | 15 seconds  |

## Reverse Tunnels
| **Key**                  | **Description**                                            | **Required**              | **Default** |
|--------------------------|------------------------------------------------------------|---------------------------|-------------|
| tunnel.reverse.enabled   | Enable Reverse Tunnels.                                    | True                      | False       |
| tunnel.reverse.host.key  | Base64 encoded [host key](https://www.ssh.com/academy/ssh/host-key) for the reverse tunnel SSH server. | True, if reverse enabled. |             |
| tunnel.reverse.bind.host | Bind host for the reverse tunnel SSH server                | True, if reverse enabled. | 0.0.0.0 |

## Service Discovery
A production passage deployment may have the normal tunnel server running separately from the reverse tunnel server, and an API server running separately from the two.

| **Key**                | **Description**                                   | **Required**       | **Default** |
|------------------------|---------------------------------------------------|--------------------|-------------|
| discovery.type         | Tunnel service discovery type (`static` or `srv`) | True               | `static`    |
| discovery.srv.registry | If `srv`, the DNS SRV registry to use.            | True, if `srv`.    |             |
| discovery.srv.prefix   | TODO                                              | True, if `srv`.    |             |
| discovery.static.host  | If `static`, the hostname to use.                 | True, if `static`. |             |

## Keystore
Passage needs a place to securely store SSH private keys for normal Tunnels and public keys for Reverse Tunnels. By default, Passage will store keys unencrypted in a Postgres table, but that should not be deployed to production.

With the `s3` keystore, Passage will store keys in an S3 bucket. If you choose to go this route, make sure you have properly configured bucket policies and IAM permissions to restrict access to _only_ Passage. Also, it is recommended that you enable at-rest bucket encryption with KMS.

| **Key**                      | **Description**                           | **Required**        |
|------------------------------|-------------------------------------------|---------------------|
| keystore.type                | Tunnel keystore type (`postgres` or `s3`) | True                |
| keystore.postgres.table.name | If `postgres`, the table name to use.     | True, if `postgres` |
| keystore.s3.bucket.name      | If `s3`, the bucket name to use.          | True, if `s3`       |
| keystore.s3.key.prefix       | If `s3`, the prefix applied to keys.      | False               |

## Database Connection
| **Key**          | **Description**             | **Required** | **Alias**   |
|------------------|-----------------------------|--------------|-------------|
| postgres.uri     | Postgres connection string. | False        |             |
| postgres.host    | See `PGHOST`                | True         | `PGHOST`    |
| postgres.port    | See `PGPORT`                | True         | `PGPORT`    |
| postgres.user    | See `PGUSER`                | True         | `PGUSER`    |
| postgres.pass    | See `PGPASS`                | True         | `PGPASS`    |
| postgres.dbname  | See `PGDBNAME`              | True         | `PGDBNAME`  |
| postgres.sslmode | See `PGSSLMODE`             | True         | `PGSSLMODE` |

## Visibility
| **Key**     | **Description**                                         | **Required** | **Default** |
|-------------|---------------------------------------------------------|--------------|-------------|
| env         | Name of the env for logging and metrics.                | False        |             |
| log.level   | Visibility level for logs (debug/info/warn/error/fatal) | False        | `info`      |
| log.format  | Format of structured logs (json/text)                   | False        | `text`      |
| statsd.addr | Address of a Statsd server to send metrics to.          | False        |             |
