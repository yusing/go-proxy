# go-proxy

A simple auto docker reverse proxy for home use. **Written in _Go_**

In the examples domain `x.y.z` is used, replace them with your domain

## Table of content

- [Key Points](#key-points)
- [How to use](#how-to-use)
  - [Binary](#binary)
  - [Docker](#docker)
- [Configuration](#configuration)
  - [Single Port Configuration](#single-port-configuration-example)
  - [Multiple Ports Configuration](#multiple-ports-configuration-example)
  - [TCP/UDP Configuration](#tcpudp-configuration-example)
  - [Load balancing Configuration](#load-balancing-configuration-example)
- [Troubleshooting](#troubleshooting)
- [Benchmarks](#benchmarks)
- [Memory usage](#memory-usage)
- [Build it yourself](#build-it-yourself)
- [Getting SSL certs](#getting-ssl-certs)

## Key Points

- fast, nearly no performance penalty for end users when comparing to direct IP connections (See [benchmarks](#benchmarks))
- auto detect reverse proxies from docker
- additional reverse proxies from provider yaml file
- allow multiple docker / file providers by custom `config.yml` file
- subdomain matching **(domain name doesn't matter)**
- path matching
- HTTP proxy
- TCP/UDP Proxy
- HTTP round robin load balance support (same subdomain and path across different hosts)
- Auto hot-reload on container start / die / stop or config changes.
- Simple panel to see all reverse proxies and health (visit port [panel port] of go-proxy `https://*.y.z:[panel port]`)

  ![panel screenshot](screenshots/panel.png)

## How to use

1. Download and extract the latest release (or clone the repository if you want to try out experimental features)

2. Copy `config.example.yml` to `config.yml` and modify the content to fit your needs

3. Do the same for `providers.example.yml`

4. See [Binary](#binary) or [docker](#docker)

### Binary

1. (Optional) Prepare your certificates in `certs/` to enable https. See [Getting SSL Certs](#getting-ssl-certs)


    - cert / chain / fullchain: `./certs/cert.crt`
    - private key: `./certs/priv.key`

2. run the binary `bin/go-proxy`

3. enjoy

### Docker

1. Copy content from [compose.example.yml](compose.example.yml) and create your own `compose.yml`

2. Add networks to make sure it is in the same network with other containers, or make sure `proxy.<alias>.host` is reachable

3. (Optional) Mount your SSL certs to enable https. See [Getting SSL Certs](#getting-ssl-certs)


    - cert / chain / fullchain -> `/app/certs/cert.crt`
    - private key -> `/app/certs/priv.key`

4. Start `go-proxy` with `docker compose up -d` or `make up`.

5. (Optional) If you are using ufw with vpn that drop all inbound traffic except vpn, run below to allow docker containers to connect to `go-proxy`


    In case the network of your container is in subnet `172.16.0.0/16` (bridge),
    and vpn network is under `100.64.0.0/10` (i.e. tailscale)

    `sudo ufw allow from 172.16.0.0/16 to 100.64.0.0/10`

    You can also list CIDRs of all docker bridge networks by:

    `docker network inspect $(docker network ls | awk '$3 == "bridge" { print $1}') | jq -r '.[] | .Name + " " + .IPAM.Config[0].Subnet' -`

6. start your docker app, and visit <container_name>.y.z

7. check the logs with `docker compose logs` or `make logs` to see if there is any error, check panel at [panel port] for active proxies

## Known issues

None

## Configuration

With container name, no label needs to be added.

However, there are some labels you can manipulate with:

- `proxy.aliases`: comma separated aliases for subdomain matching
  - defaults to `container_name`
- `proxy.*.<field>`: wildcard config for all aliases
- `proxy.<alias>.scheme`: container port protocol (`http` or `https`)
  - defaults to `http`
- `proxy.<alias>.host`: proxy host
  - defaults to `container_name`
- `proxy.<alias>.port`: proxy port
  - http/https: defaults to first expose port (declared in `Dockerfile` or `docker-compose.yml`)
  - tcp/udp: is in format of `[<listeningPort>:]<targetPort>`
    - when `listeningPort` is omitted (not suggested), a free port will be used automatically.
    - `targetPort` must be a number, or the predefined names (see [stream.go](src/go-proxy/stream.go#L28))
- `no_tls_verify`: whether skip tls verify when scheme is https
  - defaults to false
- `proxy.<alias>.path`: path matching (for http proxy only)
  - defaults to empty
- `proxy.<alias>.path_mode`: mode for path handling

  - defaults to empty
  - allowed: \<empty>, forward, sub
    - empty: remove path prefix from URL when proxying
      1. apps.y.z/webdav -> webdav:80
      2. apps.y.z./webdav/path/to/file -> webdav:80/path/to/file
    - forward: path remain unchanged
      1. apps.y.z/webdav -> webdav:80/webdav
      2. apps.y.z./webdav/path/to/file -> webdav:80/webdav/path/to/file
    - sub: (experimental) remove path prefix from URL and also append path to HTML link attributes (`src`, `href` and `action`) and Javascript `fetch(url)` by response body substitution
      e.g. apps.y.z/app1 -> webdav:80, `href="/path/to/file"` -> `href="/app1/path/to/file"`

- `proxy.<alias>.load_balance`: enable load balance
  - allowed: `1`, `true`

### Single port configuration example

```yaml
# (default) https://<container_name>.y.z
whoami:
  image: traefik/whoami
  container_name: whoami # => whoami.y.z

# enable both subdomain and path matching:
whoami:
  image: traefik/whoami
  container_name: whoami
  labels:
    - proxy.aliases=whoami,apps
    - proxy.apps.path=/whoami
# 1. visit https://whoami.y.z
# 2. visit https://apps.y.z/whoami
```

### Multiple ports configuration example

```yaml
minio:
  image: quay.io/minio/minio
  container_name: minio
  ...
  labels:
    - proxy.aliases=minio,minio-console
    - proxy.minio.port=9000
    - proxy.minio-console.port=9001

# visit https://minio.y.z to access minio
# visit https://minio-console.y.z/whoami to access minio console
```

### TCP/UDP configuration example

```yaml
# In the app
app-db:
  image: postgres:15
  container_name: app-db
  ...
  labels:
    # Optional (postgres is in the known image map)
    - proxy.app-db.scheme=tcp

    # Optional (first free port will be used for listening port)
    - proxy.app-db.port=20000:postgres

# In go-proxy
go-proxy:
  ...
  ports:
    - 80:80
    ...
    - 20000:20000/tcp
    # or 20000-20010:20000-20010/tcp to declare large range at once

# access app-db via <*>.y.z:20000
```

## Load balancing Configuration Example

```yaml
nginx:
  ...
  deploy:
    mode: replicated
    replicas: 3
  labels:
    - proxy.nginx.load_balance=1 # allowed: [1, true]
```

## Troubleshooting

Q: How to fix when it shows "no matching route for subdomain \<subdomain>"?

A: Make sure the container is running, and \<subdomain> matches any container name / alias

## Benchmarks

Benchmarked with `wrk` connecting `traefik/whoami`'s `/bench` endpoint

Remote benchmark (client running wrk and `go-proxy` server are different devices)

- Direct connection

  ```shell
  root@yusing-pc:~# wrk -t 10 -c 200 -d 30s --latency http://10.0.100.1/bench
  Running 30s test @ http://10.0.100.1/bench
    10 threads and 200 connections
    Thread Stats   Avg      Stdev     Max   +/- Stdev
      Latency     4.34ms    1.16ms  22.76ms   85.77%
      Req/Sec     4.63k   435.14     5.47k    90.07%
    Latency Distribution
      50%    3.95ms
      75%    4.71ms
      90%    5.68ms
      99%    8.61ms
    1383812 requests in 30.02s, 166.28MB read
  Requests/sec:  46100.87
  Transfer/sec:      5.54MB
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

  ```
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
  ```
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
  ```
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

## Memory usage

It takes ~30 MB for 50 proxy entries

## Build it yourself

1. Install [go](https://go.dev/doc/install) and `make` if not already

2. get dependencies with `make get`

3. build binary with `make build`

4. start your container with `docker compose up -d`

## Getting SSL certs

I personally use `nginx-proxy-manager` to get SSL certs with auto renewal by Cloudflare DNS challenge. You may symlink the certs from `nginx-proxy-manager` to `certs/` folder relative to project root. (For docker) mount them to `go-proxy`'s `/app/certs`

[panel port]: 8443
