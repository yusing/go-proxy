# go-proxy

[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=yusing_go-proxy&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=yusing_go-proxy)
[![Lines of Code](https://sonarcloud.io/api/project_badges/measure?project=yusing_go-proxy&metric=ncloc)](https://sonarcloud.io/summary/new_code?id=yusing_go-proxy)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=yusing_go-proxy&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=yusing_go-proxy)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=yusing_go-proxy&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=yusing_go-proxy)
[![Vulnerabilities](https://sonarcloud.io/api/project_badges/measure?project=yusing_go-proxy&metric=vulnerabilities)](https://sonarcloud.io/summary/new_code?id=yusing_go-proxy)

[繁體中文文檔請看此](README_CHT.md)

A lightweight, easy-to-use, and [performant](docs/benchmark_result.md) reverse proxy with a web UI.

## Table of content

<!-- TOC -->

- [go-proxy](#go-proxy)
  - [Table of content](#table-of-content)
  - [Key Points](#key-points)
  - [Getting Started](#getting-started)
    - [Setup](#setup)
    - [Commands line arguments](#commands-line-arguments)
    - [Environment variables](#environment-variables)
    - [Use JSON Schema in VSCode](#use-json-schema-in-vscode)
    - [Config File](#config-file)
    - [Provider File](#provider-file)
  - [Known issues](#known-issues)
  - [Build it yourself](#build-it-yourself)

## Key Points

- Easy to use
  - Effortless configuration
  - Error messages is clear and detailed, easy troubleshooting
- Auto certificate obtaining and renewal (See [Supported DNS Challenge Providers](docs/dns_providers.md))
- Auto configuration for docker containers
- Auto hot-reload on container state / config file changes
- Stop containers on idle, wake it up on traffic _(optional)_
- HTTP(s) reserve proxy
- TCP and UDP port forwarding
- Web UI for configuration and monitoring (See [screenshots](https://github.com/yusing/go-proxy-frontend?tab=readme-ov-file#screenshots))
- Written in **[Go](https://go.dev)**

[🔼Back to top](#table-of-content)

## Getting Started

### Setup

1. Setup DNS Records, e.g.

   - A Record: `*.y.z` -> `10.0.10.1`
   - AAAA Record: `*.y.z` -> `::ffff:a00:a01`

2. Setup `go-proxy` [See here](docs/docker.md)

3. Configure `go-proxy`
   - with text editor (e.g. Visual Studio Code)
   - or with web config editor via `http://gp.y.z`

[🔼Back to top](#table-of-content)

### Commands line arguments

| Argument    | Description                      | Example                    |
| ----------- | -------------------------------- | -------------------------- |
| empty       | start proxy server               |                            |
| `validate`  | validate config and exit         |                            |
| `reload`    | trigger a force reload of config |                            |
| `ls-config` | list config and exit             | `go-proxy ls-config \| jq` |
| `ls-route`  | list proxy entries and exit      | `go-proxy ls-route \| jq`  |

**run with `docker exec <container_name> /app/go-proxy <command>`**

### Environment variables

| Environment Variable           | Description                   | Default | Values  |
| ------------------------------ | ----------------------------- | ------- | ------- |
| `GOPROXY_NO_SCHEMA_VALIDATION` | disable schema validation     | `false` | boolean |
| `GOPROXY_DEBUG`                | enable debug behaviors        | `false` | boolean |
| `GOPROXY_HTTP_PORT`            | http server port              | `80`    | integer |
| `GOPROXY_HTTPS_PORT`           | http server port (if enabled) | `443`   | integer |
| `GOPROXY_API_PORT`             | api server port               | `8888`  | integer |

### Use JSON Schema in VSCode

Copy [`.vscode/settings.example.json`](.vscode/settings.example.json) to `.vscode/settings.json` and modify it to fit your needs

[🔼Back to top](#table-of-content)

### Config File

See [config.example.yml](config.example.yml) for more

```yaml
# autocert configuration
autocert:
  email: # ACME Email
  domains: # a list of domains for cert registration
  provider: # DNS Challenge provider
  options: # provider specific options
    - ...
# reverse proxy providers configuration
providers:
  include:
    - providers.yml
    - other_file_1.yml
    - ...
  docker:
    local: $DOCKER_HOST
    remote-1: tcp://10.0.2.1:2375
    remote-2: ssh://root:1234@10.0.2.2
```

[🔼Back to top](#table-of-content)

### Provider File

See [Fields](docs/docker.md#fields)

See [providers.example.yml](providers.example.yml) for examples

[🔼Back to top](#table-of-content)

## Known issues

- Cert "renewal" is actually obtaining a new cert instead of renewing the existing one

- `autocert` config is not hot-reloadable

[🔼Back to top](#table-of-content)

## Build it yourself

1. Clone the repository `git clone https://github.com/yusing/go-proxy --depth=1`

2. Install / Upgrade [go (>=1.22)](https://go.dev/doc/install) and `make` if not already

3. Clear cache if you have built this before (go < 1.22) with `go clean -cache`

4. get dependencies with `make get`

5. build binary with `make build`

[🔼Back to top](#table-of-content)
