# go-proxy

A simple auto docker reverse proxy for home use.

Written in **Go** with *~220 loc*.

## Features

- subdomain matching **(domain name doesn't matter)**
- path matching
- Auto hot-reload when container start / die / stop.

## Why am I making this

I have tried different reverse proxy services, i.e. [nginx proxy manager](https://nginxproxymanager.com/), [traefik](https://github.com/traefik/traefik), [nginx-proxy](https://github.com/nginx-proxy/nginx-proxy). I have found that `traefik` is not easy to use, and I don't want to click buttons every time I spin up a new container (`nginx proxy manager`). For `nginx-proxy` I found it buggy and quite unusable.

## How to use

1. Clone the repo `git clone https://github.com/yusing/go-proxy`

2. Copy [compose.example.yml](compose.example.yml) to `compose.yml`

3. Add networks to make sure it is in the same network with other containers, or make sure `proxy.<alias>.host` is reachable

4. Modify the path to your SSL certs. See [Getting SSL Certs](#getting-ssl-certs)

5. Start `go-proxy` with `docker compose up -d`.

6. (Optional) If you are using ufw with vpn that drop all inbound traffic except vpn, run below to allow docker containers to connect to `go-proxy`

    In case the network of your container is in subnet `172.16.0.0/12` (bridge),
    and vpn network is under `100.64.0.0/10` (i.e. tailscale)

    `sudo ufw allow from 172.16.0.0/12 to 100.64.0.0/10`

    You can also list CIDRs of all docker bridge networks by:

    `docker network inspect $(docker network ls | awk '$3 == "bridge" { print $1}') | jq -r '.[] | .Name + " " + .IPAM.Config[0].Subnet' -`

7. start your docker app, and visit <container_name>.yourdomain.com

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
    Latency     4.94ms    1.88ms  43.49ms   85.82%
    Req/Sec     1.03k   123.57     1.22k    83.20%
  Latency Distribution
     50%    4.60ms
     75%    5.59ms
     90%    6.77ms
     99%   10.81ms
  203565 requests in 10.02s, 19.80MB read
Requests/sec:  20320.87
Transfer/sec:      1.98MB
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
