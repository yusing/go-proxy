# go-proxy

A simple auto docker reverse proxy for home use.

Written in **Go** with *~180 loc*.

## Features

- subdomain matching **(domain name doesn't matter)**
- path matching
- Auto hot-reload when container start / die / stop.

## Why am I making this

I have tried different reverse proxy services, i.e. [nginx proxy manager](https://nginxproxymanager.com/), [traefik](https://github.com/traefik/traefik), [nginx-proxy](https://github.com/nginx-proxy/nginx-proxy). I have found that `traefik` is not easy to use, and I don't want to click buttons every time I spin up a new container (`nginx proxy manager`). For `nginx-proxy` I found it buggy and quite unusable.

## How to use

1. Clone the repo `git clone https://github.com/yusing/go-proxy`

2. Copy [compose.example.yml](compose.example.yml) to `compose.yml`

3. add networks to make sure it is in the same network with other containers, or make sure `proxy.<alias>.host` is reachable

4. modify the path to your SSL certs. See [Getting SSL Certs](#getting-ssl-certs)

5. start `go-proxy` with `docker compose up -d`.

6. start your docker app, and visit <container_name>.yourdomain.com

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
  - defaults to first expose port (declared in `Dockerfile` or `docker-compose.yml`)
- `proxy.<alias>.path`: path matching
  - defaults to empty

```yaml
version: '3'
services:
  whoami:
    image: traefik/whoami # port 80 is exposed
    container_name: whoami
# (default) https://whoami.yourdomain.com

# enable both subdomain and path matching:
whoami:
  image: traefik/whoami
  container_name: whoami
  labels:
    - proxy.aliases=whoami,apps
    - proxy.apps.path=/whoami
# 1. visit https://whoami.yourdomain.com
# 2. visit https://apps.yourdomain.com/whoami
```

For multiple port container (i.e. minio)

```yaml
version: '3'
services:
  minio:
    image: quay.io/minio/minio
    container_name: minio
    command:
      - server
      - /data
      - --console-address
      - "9001"
    env_file: minio.env
    expose:
      - 9000
      - 9001
    volumes:
      - ./data/minio/data:/data
    labels:
      proxy.aliases: minio,minio-console
      proxy.minio.port: 9000
      proxy.minio-console.port: 9001

# visit https://minio.yourdomain.com to access minio
# visit https://minio-console.yourdomain.com/whoami to access minio console
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
    Latency     3.71ms    2.26ms  48.10ms   94.95%
    Req/Sec     1.41k   179.01     2.11k    69.97%
  Latency Distribution
     50%    3.32ms
     75%    3.98ms
     90%    4.97ms
     99%   11.36ms
  282804 requests in 10.10s, 33.98MB read
Requests/sec:  27998.62
Transfer/sec:      3.36MB
```

With **go-proxy** reverse proxy

```shell
% wrk -t20 -c100 -d10s --latency https://whoami.mydomain.com/bench
Running 10s test @ https://whoami.mydomain.com/bench
  20 threads and 100 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     4.41ms    2.56ms  77.80ms   95.38%
    Req/Sec     1.18k   156.44     1.63k    86.51%
  Latency Distribution
     50%    3.93ms
     75%    4.76ms
     90%    5.92ms
     99%   10.46ms
  235374 requests in 10.10s, 22.90MB read
Requests/sec:  23302.42
Transfer/sec:      2.27MB
```

## Build it yourself

1. [Install go](https://go.dev/doc/install) if not already

2. Get dependencies with `go get`

3. build image with following commands

    ```shell
    mkdir -p bin
    CGO_ENABLED=0 GOOS=<platform> go build -o bin/go-proxy
    docker build -t <tag> .
    ```

## Getting SSL certs

I personally use `nginx-proxy-manager` to get SSL certs with auto renewal by Cloudflare DNS challenge. You may symlink the certs from `nginx-proxy-manager` to somewhere else, and mount them to `go-proxy`'s `/certs`
