# go-proxy

A simple auto docker reverse proxy for home use. *Written in **Go***

In the examples domain `x.y.z` is used, replace them with your domain

## Table of content

- [Features](#features)
- [Why am I making this](#why-am-i-making-this)
- [How to use](#how-to-use)
- [Configuration](#configuration)
  - [Single Port Configuration](#single-port-configuration-example)
  - [Multiple Ports Configuration](#multiple-ports-configuration-example)
  - [TCP/UDP Configuration](#tcpudp-configuration-example)
- [Troubleshooting](#troubleshooting)
- [Benchmarks](#benchmarks)
- [Memory usage](#memory-usage)
- [Build it yourself](#build-it-yourself)
- [Getting SSL certs](#getting-ssl-certs)

## Features

- subdomain matching **(domain name doesn't matter)**
- path matching
- HTTP proxy
- TCP/UDP Proxy (experimental, unable to release port on hot-reload)
- Auto hot-reload when container start / die / stop.
- Simple panel to see all reverse proxies and health (visit port :8443 of go-proxy `https://*.y.z:8443`)

    ![panel screenshot](screenshots/panel.png)

## Why am I making this

1. It's fun.
2. I have tried different reverse proxy services, i.e. [nginx proxy manager](https://nginxproxymanager.com/), [traefik](https://github.com/traefik/traefik), [nginx-proxy](https://github.com/nginx-proxy/nginx-proxy). I have found that `traefik` is not easy to use, and I don't want to click buttons every time I spin up a new container (`nginx proxy manager`). For `nginx-proxy` I found it buggy and quite unusable.

## How to use

1. Clone the repo git clone `https://github.com/yusing/go-proxy`

2. Copy content from [compose.example.yml](compose.example.yml) and create your own `compose.yml`

3. Add networks to make sure it is in the same network with other containers, or make sure `proxy.<alias>.host` is reachable

4. Modify the path to your SSL certs. See [Getting SSL Certs](#getting-ssl-certs)

5. Start `go-proxy` with `docker compose up -d`.

6. (Optional) If you are using ufw with vpn that drop all inbound traffic except vpn, run below to allow docker containers to connect to `go-proxy`

    In case the network of your container is in subnet `172.16.0.0/12` (bridge),
    and vpn network is under `100.64.0.0/10` (i.e. tailscale)

    `sudo ufw allow from 172.16.0.0/12 to 100.64.0.0/10`

    You can also list CIDRs of all docker bridge networks by:

    `docker network inspect $(docker network ls | awk '$3 == "bridge" { print $1}') | jq -r '.[] | .Name + " " + .IPAM.Config[0].Subnet' -`

7. start your docker app, and visit <container_name>.y.z

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
- `proxy.<alias>.path`: path matching (for http proxy only)
  - defaults to empty

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
    proxy.aliases: minio,minio-console
    proxy.minio.port: 9000
    proxy.minio-console.port: 9001

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

## Troubleshooting

Q: How to fix when it shows "no matching route for subdomain \<subdomain>"?

A: Make sure the container is running, and \<subdomain> matches any container name / alias

## Benchmarks

Benchmarked with `wrk` connecting `traefik/whoami`'s `/bench` endpoint

Direct connection

```shell
% wrk -t20 -c100 -d10s --latency http://homelab:4999/bench
Running 10s test @ http://homelab:4999/bench
  20 threads and 100 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     3.74ms    1.19ms  19.94ms   81.53%
    Req/Sec     1.35k   103.96     1.60k    73.60%
  Latency Distribution
     50%    3.46ms
     75%    4.16ms
     90%    4.98ms
     99%    8.04ms
  269696 requests in 10.01s, 32.41MB read
Requests/sec:  26950.35
Transfer/sec:      3.24MB
```

With **go-proxy** reverse proxy

```shell
% wrk -t20 -c100 -d10s --latency https://whoami.mydomain.com/bench
Running 10s test @ https://whoami.6uo.me/bench
  20 threads and 100 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     4.02ms    2.13ms  47.49ms   95.14%
    Req/Sec     1.28k   139.15     1.47k    91.67%
  Latency Distribution
     50%    3.60ms
     75%    4.36ms
     90%    5.29ms
     99%    8.83ms
  253874 requests in 10.02s, 24.70MB read
Requests/sec:  25342.46
Transfer/sec:      2.47MB
```

## Memory usage

It takes ~ 0.1-0.4MB for each HTTP Proxy, and <2MB for each TCP/UDP Proxy

## Build it yourself

1. [Install go](https://go.dev/doc/install) if not already

2. get dependencies with `sh scripts/get.sh`

3. build binary with `sh scripts/build.sh`

4. start your container with `docker compose up -d`

## Getting SSL certs

I personally use `nginx-proxy-manager` to get SSL certs with auto renewal by Cloudflare DNS challenge. You may symlink the certs from `nginx-proxy-manager` to somewhere else, and mount them to `go-proxy`'s `/certs`
