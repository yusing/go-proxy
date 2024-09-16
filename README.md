# go-proxy

A [lightweight](docs/benchmark_result.md), easy-to-use, and efficient reverse proxy and load balancer with a web UI.

**Table of content**

<!-- TOC -->

- [go-proxy](#go-proxy)
  - [Key Points](#key-points)
  - [Getting Started](#getting-started)
    - [Commands line arguments](#commands-line-arguments)
    - [Environment variables](#environment-variables)
    - [Use JSON Schema in VSCode](#use-json-schema-in-vscode)
    - [Config File](#config-file)
    - [Provider File](#provider-file)
  - [Known issues](#known-issues)
  - [Build it yourself](#build-it-yourself)

## Key Points

- Easy to use
- Auto certificate obtaining and renewal (See [Supported DNS Challenge Providers](docs/dns_providers.md))
- Auto configuration for docker contaienrs
- Auto hot-reload on container state / config file changes
- Support HTTP(s), TCP and UDP
- Web UI for configuration and monitoring (See [screenshots](https://github.com/yusing/go-proxy-frontend?tab=readme-ov-file#screenshots))
- Written in **[Go](https://go.dev)**

[🔼Back to top](#table-of-content)

## Getting Started

1. Setup DNS Records

   - A Record: `*.y.z` -> `10.0.10.1`
   - AAAA Record: `*.y.z` -> `::ffff:a00:a01`

2. Setup `go-proxy` [See here](docs/docker.md)

3. Configure `go-proxy`
   - with text editor (i.e. Visual Studio Code)
   - or with web config editor via `http://gp.y.z`

[🔼Back to top](#table-of-content)

### Commands line arguments

| Argument   | Description                      |
| ---------- | -------------------------------- |
| empty      | start proxy server               |
| `validate` | validate config and exit         |
| `reload`   | trigger a force reload of config |

**run with `docker exec <container_name> /app/go-proxy <command>`**

### Environment variables

| Environment Variable           | Description               | Default | Values  |
| ------------------------------ | ------------------------- | ------- | ------- |
| `GOPROXY_NO_SCHEMA_VALIDATION` | disable schema validation | `false` | boolean |
| `GOPROXY_DEBUG`                | enable debug behaviors    | `false` | boolean |

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

Fields are same as [docker labels](docs/docker.md#labels) starting from `scheme`

See [providers.example.yml](providers.example.yml) for examples

[🔼Back to top](#table-of-content)

## Known issues

- Cert "renewal" is actually obtaining a new cert instead of renewing the existing one

[🔼Back to top](#table-of-content)

## Build it yourself

1. Clone the repository `git clone https://github.com/yusing/go-proxy --depth=1`

2. Install / Upgrade [go (>=1.22)](https://go.dev/doc/install) and `make` if not already

3. Clear cache if you have built this before (go < 1.22) with `go clean -cache`

4. get dependencies with `make get`

5. build binary with `make build`

6. start your container with `make up` (docker) or `bin/go-proxy` (binary)

[🔼Back to top](#table-of-content)
