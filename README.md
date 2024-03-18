# go-proxy

A simple auto docker reverse proxy for home use. **Written in *Go***

In the examples domain `x.y.z` is used, replace them with your domain

## Table of content

- [Key Points](#key-points)
- [How to use](#how-to-use)
  - [Binary] (#binary)
  - [Docker] (#docker)
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

## How to use (docker)

1. Download and extract the latest release (or clone the repository if you want to try out experimental features)
2. Copy `config.example.yml` to `config.yml` and modify the content to fit your needs
3. Do the same for `providers.example.yml`
4. See [Binary](#binary) or [docker](#docker)

### Binary
  1. (Optional) Prepare your certificates in `certs/` to enable https. See [Getting SSL Certs](#getting-ssl-certs)
    - cert / chain / fullchain: ./certs/cert.crt
    - private key: ./certs/priv.key
  2. run the binary `bin/go-proxy`
  3. enjoy

### Docker
  1. Copy content from [compose.example.yml](compose.example.yml) and create your own `compose.yml`

  2. Add networks to make sure it is in the same network with other containers, or make sure `proxy.<alias>.host` is reachable

  3. (Optional) Mount your SSL certs to enable https. See [Getting SSL Certs](#getting-ssl-certs)
    - cert / chain / fullchain -> /app/certs/cert.crt
    - private key -> /app/certs/priv.key

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
  root@yusing-pc:~# wrk -t 10 -c 200 -d 30s --latency http://bench.6uo.me/bench
  Running 30s test @ http://bench.6uo.me/bench
    10 threads and 200 connections
    Thread Stats   Avg      Stdev     Max   +/- Stdev
      Latency     4.50ms    1.44ms  27.53ms   86.48%
      Req/Sec     4.48k   375.00     5.12k    84.73%
    Latency Distribution
      50%    4.09ms
      75%    5.06ms
      90%    6.03ms
      99%    9.41ms
    1338996 requests in 30.01s, 160.90MB read
  Requests/sec:  44616.36
  Transfer/sec:      5.36MB
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

- With reverse proxy
  ```
  root@http-benchmark-client:~# wrk -t 10 -c 200 -d 10s --latency http://bench.6uo.me/bench
  Running 10s test @ http://bench.6uo.me/bench
    10 threads and 200 connections
    Thread Stats   Avg      Stdev     Max   +/- Stdev
      Latency     1.78ms    5.49ms 117.53ms   99.00%
      Req/Sec    16.31k     2.30k   21.01k    86.69%
    Latency Distribution
      50%    1.12ms
      75%    1.88ms
      90%    2.80ms
      99%    7.27ms
    1634774 requests in 10.10s, 196.44MB read
  Requests/sec: 161858.70
  Transfer/sec:     19.45MB
  ```

## Memory usage

It takes ~ 0.1-0.4MB for each HTTP Proxy, and <2MB for each TCP/UDP Proxy

## Build it yourself

1. Install [go](https://go.dev/doc/install) and `make` if not already

2. get dependencies with `make get`

3. build binary with `make build`

4. start your container with `docker compose up -d`

## Getting SSL certs

I personally use `nginx-proxy-manager` to get SSL certs with auto renewal by Cloudflare DNS challenge. You may symlink the certs from `nginx-proxy-manager` to somewhere else, and mount them to `go-proxy`'s `/certs`

[panel port]: 8443
