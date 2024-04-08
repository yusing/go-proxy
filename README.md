# go-proxy

A simple auto docker reverse proxy for home use. **Written in _Go_**

In the examples domain `x.y.z` is used, replace them with your domain

## Table of content

<!-- TOC -->
- [Table of content](#table-of-content)
- [Key Points](#key-points)
- [How to use](#how-to-use)
- [Tested Services](#tested-services)
  - [HTTP/HTTPs Reverse Proxy](#httphttps-reverse-proxy)
  - [TCP Proxy](#tcp-proxy)
  - [UDP Proxy](#udp-proxy)
- [Command-line args](#command-line-args)
  - [Commands](#commands)
- [Use JSON Schema in VSCode](#use-json-schema-in-vscode)
- [Environment variables](#environment-variables)
- [Config File](#config-file)
  - [Fields](#fields)
  - [Provider Kinds](#provider-kinds)
  - [Provider File](#provider-file)
  - [Supported DNS Challenge Providers](#supported-dns-challenge-providers)
- [Troubleshooting](#troubleshooting)
- [Benchmarks](#benchmarks)
- [Known issues](#known-issues)
- [Memory usage](#memory-usage)
- [Build it yourself](#build-it-yourself)
<!-- /TOC -->

## Key Points

- Fast (See [benchmarks](#benchmarks))
- Auto certificate obtaining and renewal (See [Config File](#config-file) and [Supported DNS Challenge Providers](#supported-dns-challenge-providers))
- Auto detect reverse proxies from docker
- Auto hot-reload on container `start` / `die` / `stop` or config file changes
- Custom proxy entries with `config.yml` and additional provider files
- Subdomain matching + Path matching **(domain name doesn't matter)**
- HTTP(s) reverse proxy + TCP/UDP Proxy
- HTTP(s) round robin load balance support (same subdomain and path across different hosts)
- Web UI on port 8080 (http) and port 8443 (https)

  - a simple panel to see all reverse proxies and health

    ![panel screenshot](screenshots/panel.png)

  - a config editor to edit config and provider files with validation

    **Validate and save file with Ctrl+S**

    ![config editor screenshot](screenshots/config_editor.png)

[🔼Back to top](#table-of-content)

## How to use

1. Setup DNS Records to your machine's IP address

   - A Record: `*.y.z` -> `10.0.10.1`
   - AAAA Record: `*.y.z` -> `::ffff:a00:a01`

2. Start `go-proxy` by

   - [Running from binary or as a system service](docs/binary.md)
   - [Running as a docker container](docs/docker.md)

3. Start editing config files
   - with text editor (i.e. Visual Studio Code)
   - or with web config editor by navigate to `http://ip:8080`

[🔼Back to top](#table-of-content)

## Tested Services

### HTTP/HTTPs Reverse Proxy

- Nginx
- Minio
- AdguardHome Dashboard
- etc.

### TCP Proxy

- Minecraft server
- PostgreSQL
- MariaDB

### UDP Proxy

- Adguardhome DNS
- Palworld Dedicated Server

[🔼Back to top](#table-of-content)

## Command-line args

`go-proxy [command]`

### Commands

- empty: start proxy server
- validate: validate config and exit
- reload: trigger a force reload of config

Examples:

- Binary: `go-proxy reload`
- Docker: `docker exec -it go-proxy /app/go-proxy reload`

[🔼Back to top](#table-of-content)

## Use JSON Schema in VSCode

Copy [`.vscode/settings.example.json`](.vscode/settings.example.json) to `.vscode/settings.json` and modify to fit your needs

```json
{
  "yaml.schemas": {
    "https://github.com/yusing/go-proxy/raw/main/schema/config.schema.json": [
      "config.example.yml",
      "config.yml"
    ],
    "https://github.com/yusing/go-proxy/raw/main/schema/providers.schema.json": [
      "providers.example.yml",
      "*.providers.yml"
    ]
  }
}
```

[🔼Back to top](#table-of-content)

## Environment variables

- `GOPROXY_DEBUG`: set to `1` or `true` to enable debug behaviors (i.e. output, etc.)
- `GOPROXY_HOST_NETWORK`: _(Docker only)_ set to `1` when `network_mode: host`
- `GOPROXY_NO_SCHEMA_VALIDATION`: disable schema validation on config load / reload **(for testing new DNS Challenge providers)**

[🔼Back to top](#table-of-content)

## Config File

See [config.example.yml](config.example.yml) for more

### Fields

- `autocert`: autocert configuration

  - `email`: ACME Email
  - `domains`: a list of domains for cert registration
  - `provider`: DNS Challenge provider, see [Supported DNS Challenge Providers](#supported-dns-challenge-providers)
  - `options`: [provider specific options](#supported-dns-challenge-providers)

- `providers`: reverse proxy providers configuration
  - `kind`: provider kind (string), see [Provider Kinds](#provider-kinds)
  - `value`: provider specific value

[🔼Back to top](#table-of-content)

### Provider Kinds

- `docker`: load reverse proxies from docker

  values:

  - `FROM_ENV`: value from environment (`DOCKER_HOST`)
  - full url to docker host (i.e. `tcp://host:2375`)

- `file`: load reverse proxies from provider file

  value: relative path of file to `config/`

[🔼Back to top](#table-of-content)

### Provider File

Fields are same as [docker labels](docs/docker.md#labels) starting from `scheme`

See [providers.example.yml](providers.example.yml) for examples

[🔼Back to top](#table-of-content)

### Supported DNS Challenge Providers

- Cloudflare

  - `auth_token`: your zone API token

  Follow [this guide](https://cloudkul.com/blog/automcatic-renew-and-generate-ssl-on-your-website-using-lego-client/) to create a new token with `Zone.DNS` read and edit permissions

- CloudDNS

  - `client_id`
  - `email`
  - `password`

- DuckDNS (thanks [earvingad](https://github.com/earvingad))

  - `token`: DuckDNS Token

To add more provider support, see [this](docs/add_dns_provider.md)

[🔼Back to top](#table-of-content)

## Troubleshooting

Q: How to fix when it shows "no matching route for subdomain \<subdomain>"?

A: Make sure the container is running, and \<subdomain> matches any container name / alias

[🔼Back to top](#table-of-content)

## Benchmarks

Benchmarked with `wrk` connecting `traefik/whoami`'s `/bench` endpoint

Remote benchmark (client running wrk and `go-proxy` server are different devices)

- Direct connection

  ```shell
  root@yusing-pc:~# wrk -t 10 -c 200 -d 10s -H "Host: bench.6uo.me" --latency http://10.0.100.3:8003/bench
  Running 10s test @ http://10.0.100.3:8003/bench
    10 threads and 200 connections
    Thread Stats   Avg      Stdev     Max   +/- Stdev
      Latency    94.75ms  199.92ms   1.68s    91.27%
      Req/Sec     4.24k     1.79k   18.79k    72.13%
    Latency Distribution
      50%    1.14ms
      75%  120.23ms
      90%  245.63ms
      99%    1.03s
    423444 requests in 10.10s, 50.88MB read
    Socket errors: connect 0, read 0, write 0, timeout 29
  Requests/sec:  41926.32
  Transfer/sec:      5.04MB
  ```

- With reverse proxy

  ```shell
  root@yusing-pc:~# wrk -t 10 -c 200 -d 10s -H "Host: bench.6uo.me" --latency http://10.0.1.7/bench
  Running 10s test @ http://10.0.1.7/bench
    10 threads and 200 connections
    Thread Stats   Avg      Stdev     Max   +/- Stdev
      Latency    79.35ms  169.79ms   1.69s    92.55%
      Req/Sec     4.27k     1.90k   19.61k    75.81%
    Latency Distribution
      50%    1.12ms
      75%  105.66ms
      90%  200.22ms
      99%  814.59ms
    409836 requests in 10.10s, 49.25MB read
    Socket errors: connect 0, read 0, write 0, timeout 18
  Requests/sec:  40581.61
  Transfer/sec:      4.88MB
  ```

Local benchmark (client running wrk and `go-proxy` server are under same proxmox host but different LXCs)

- Direct connection

  ```shell
  root@http-benchmark-client:~# wrk -t 10 -c 200 -d 10s --latency http://10.0.100.1/bench
  Running 10s test @ http://10.0.100.1/bench
    10 threads and 200 connections
    Thread Stats   Avg      Stdev     Max   +/- Stdev
      Latency   434.08us  539.35us   8.76ms   85.28%
      Req/Sec    67.71k     6.31k   87.21k    71.20%
    Latency Distribution
      50%  153.00us
      75%  646.00us
      90%    1.18ms
      99%    2.38ms
    6739591 requests in 10.01s, 809.85MB read
  Requests/sec: 673608.15
  Transfer/sec:     80.94MB
  ```

- With `go-proxy` reverse proxy

  ```shell
  root@http-benchmark-client:~# wrk -t 10 -c 200 -d 10s -H "Host: bench.6uo.me" --latency http://10.0.1.7/bench
  Running 10s test @ http://10.0.1.7/bench
    10 threads and 200 connections
    Thread Stats   Avg      Stdev     Max   +/- Stdev
      Latency     1.23ms    0.96ms  11.43ms   72.09%
      Req/Sec    17.48k     1.76k   21.48k    70.20%
    Latency Distribution
      50%    0.98ms
      75%    1.76ms
      90%    2.54ms
      99%    4.24ms
    1739079 requests in 10.01s, 208.97MB read
  Requests/sec: 173779.44
  Transfer/sec:     20.88MB
  ```

- With `traefik-v3`

  ```shell
  root@traefik-benchmark:~# wrk -t10 -c200 -d10s -H "Host: benchmark.whoami" --latency http://127.0.0.1:8000/bench
  Running 10s test @ http://127.0.0.1:8000/bench
    10 threads and 200 connections
    Thread Stats   Avg      Stdev     Max   +/- Stdev
      Latency     2.81ms   10.36ms 180.26ms   98.57%
      Req/Sec    11.35k     1.74k   13.76k    85.54%
    Latency Distribution
      50%    1.59ms
      75%    2.27ms
      90%    3.17ms
      99%   37.91ms
    1125723 requests in 10.01s, 109.50MB read
  Requests/sec: 112499.59
  Transfer/sec:     10.94MB
  ```

[🔼Back to top](#table-of-content)

## Known issues

- Cert "renewal" is actually obtaining a new cert instead of renewing the existing one

[🔼Back to top](#table-of-content)

## Memory usage

It takes ~15 MB for 50 proxy entries

[🔼Back to top](#table-of-content)

## Build it yourself

1. Install / Upgrade [go (>=1.22)](https://go.dev/doc/install) and `make` if not already

2. Clear cache if you have built this before (go < 1.22) with `go clean -cache`

3. get dependencies with `make get`

4. build binary with `make build`

5. start your container with `make up` (docker) or `bin/go-proxy` (binary)

[🔼Back to top](#table-of-content)
