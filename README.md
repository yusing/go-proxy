# go-proxy

A [lightweight](docs/benchmark_result.md), easy-to-use, and efficient reverse proxy and load balancer with a web UI.

**Table of content**

<!-- TOC -->
- [Key Points](#key-points)
- [Getting Started](#getting-started)
  - [Commands](#commands)
  - [Environment variables](#environment-variables)
  - [Use JSON Schema in VSCode](#use-json-schema-in-vscode)
  - [Config File](#config-file)
  - [Provider File](#provider-file)
- [Known issues](#known-issues)
- [Build it yourself](#build-it-yourself)
<!-- /TOC -->

## Key Points

- Easy to use
- Auto certificate obtaining and renewal (See [Supported DNS Challenge Providers](docs/dns_providers.md))
- Auto configuration for docker contaienrs
- Auto hot-reload on container state / config file changes
- Support HTTP(s), TCP and UDP
- Support HTTP(s) round robin load balancing
- Web UI for configuration and monitoring (See [screenshots](screeenshots))
- Written in **[Go](https://go.dev)**

[🔼Back to top](#table-of-content)

## Getting Started

1. Setup DNS Records

   - A Record: `*.y.z` -> `10.0.10.1`
   - AAAA Record: `*.y.z` -> `::ffff:a00:a01`

2. Start `go-proxy`

   - [Binary / systemd service](docs/binary.md)
   - [Docker](docs/docker.md)

3. Configure `go-proxy`
   - with text editor (i.e. Visual Studio Code)
   - or with web config editor via `http://ip:8080`

[🔼Back to top](#table-of-content)

### Commands

- `go-proxy` start proxy server
- `go-proxy validate` validate config and exit
- `go-proxy reload` trigger a force reload of config

**For docker containers, run `docker exec -it go-proxy /app/go-proxy <command>`**

### Environment variables

Booleans:

- `GOPROXY_DEBUG` enable debug behaviors
- `GOPROXY_NO_SCHEMA_VALIDATION`: disable schema validation **(useful for testing new DNS Challenge providers)**

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
  entry_1:
    kind: docker
    value: # `FROM_ENV` or full url to docker host
  entry_2:
    kind: file
    value: # relative path of file to `config/`
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

1. Install / Upgrade [go (>=1.22)](https://go.dev/doc/install) and `make` if not already

2. Clear cache if you have built this before (go < 1.22) with `go clean -cache`

3. get dependencies with `make get`

4. build binary with `make build`

5. start your container with `make up` (docker) or `bin/go-proxy` (binary)

[🔼Back to top](#table-of-content)
