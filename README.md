# passage
[![CircleCI](https://circleci.com/gh/hightouchio/passage/tree/master.svg?style=svg)](https://circleci.com/gh/hightouchio/passage/tree/master)

passage is a utility for programmatically creating and managing SSH tunnels. The chief use case is to serve as a secure bridge between SaaS providers and resources that need to be accessed within customer environments.

## Usage
Passage is primarily started through the subcommand `passage server`.

```
Usage:
  passage server [flags]

Flags:
      --api       run API server
  -h, --help      help for server
      --normal    run normal tunnel server
      --reverse   run reverse tunnel server
```

## Dependencies
Passage requires a PostgreSQL database, version 11 or later. Database schema is located in [`sql/schema.sql`](`sql/schema.sql`).

## Configuration
Passage is configured primarily through environment variables, as listed below.

| Name | Description | Required | Default |
| ---- | ----------- | -------- | ------- |
| PASSAGE_ENV | Name of the env for statsd reporting. | False | *None.* |
| PASSAGE_LOG_LEVEL | Visibility level for logs (`debug, info, warn, error, fatal`). | False | `info` |
| PASSAGE_LOG_FORMAT | Format of structured logs (`json, text`) | False | `text` |
| PASSAGE_API_ENABLED | Enable the API server for programmatic tunnel management. Also `--api` via CLI flags. | False | `false` |
| PASSAGE_API_LISTEN_ADDR | Bind address for the API HTTP server. | True, if API enabled. | *None.* |
| PASSAGE_TUNNEL_NORMAL_ENABLED | Enable the normal tunnel server. Also `--normal` via CLI flags. | False | `false` |
| PASSAGE_TUNNEL_REVERSE_ENABLED | Enable the reverse tunnel server. Also `--reverse` via CLI flags. | False | `false` |
| PASSAGE_TUNNEL_REVERSE_SSH_BIND_HOST | Bind address for the reverse tunnel SSH server. | True, if reverse tunnel enabled. | `localhost` |
| PASSAGE_TUNNEL_REVERSE_SSH_HOST_KEY | Base64 encoded host key for the reverse tunnel SSH server. | True, if reverse tunnel enabled. | *None.* |
| PASSAGE_STATSD_ADDR | Address of a Statsd server to send metrics to. | False | *None.* |
