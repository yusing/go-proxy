# Docker container guide

## Table of content

<!-- TOC -->

- [Docker container guide](#docker-container-guide)
  - [Table of content](#table-of-content)
  - [Setup](#setup)
  - [Labels](#labels)
  - [Troubleshooting](#troubleshooting)
  - [Docker compose examples](#docker-compose-examples)
    - [Local docker provider in bridge network](#local-docker-provider-in-bridge-network)
      - [Proxy setup](#proxy-setup)
    - [Services URLs for above examples](#services-urls-for-above-examples)

## Setup

1. Install `wget` if not already

2. Run setup script

   `bash <(wget -qO- https://github.com/yusing/go-proxy/raw/main/setup-docker.sh)`

   What it does:

   - Create required directories
   - Setup `config.yml` and `compose.yml`

3. Verify folder structure and then `cd go-proxy`

   ```plain
   go-proxy
   ├── certs
   ├── compose.yml
   └── config
       ├── config.yml
       └── providers.yml
   ```

4. Enable HTTPs _(optional)_

   - To use autocert feature

     - completing `autocert` section in `config/config.yml`
     - mount `certs/` to `/app/certs` to store obtained certs

   - To use existing certificate

     mount your wildcard (`*.y.z`) SSL cert

     - cert / chain / fullchain -> `/app/certs/cert.crt`
     - private key -> `/app/certs/priv.key`

5. Modify `compose.yml` fit your needs

   Add networks to make sure it is in the same network with other containers, or make sure `proxy.<alias>.host` is reachable

6. Run `docker compose up -d` to start the container

7. Start editing config files in `http://<ip>:8080`

[🔼Back to top](#table-of-content)

## Labels

- `proxy.aliases`: comma separated aliases for subdomain matching

  - default: container name

- `proxy.*.<field>`: wildcard label for all aliases

_Labels below should have a **`proxy.<alias>.`** prefix._

_i.e. `proxy.nginx.scheme: http`_

- `scheme`: proxy protocol
  - default:
    - if `port` is like `x:y`: `tcp`
    - if `port` is a number: `http`
  - allowed: `http`, `https`, `tcp`, `udp`
- `host`: proxy host
  - default: `container_name`
  - allowed: IP address, hostname
- `port`: proxy port
  - default: first port in `ports:`
  - `http(s)`: number in range og `0 - 65535`
  - `tcp`, `udp`: `x:y`
    - `x`: port for `go-proxy` to listen on
    - `y`: port, or _service name_ of target container
      see [constants.go:14 for _service names_](../src/common/constants.go#L74)
- `no_tls_verify`: whether skip tls verify when scheme is https
  - default: `false`
- `path`: proxy path _(http(s) proxy only)_
  - default: empty
- `path_mode`: mode for path handling

  - default: empty
  - allowed: empty, `forward`

    - `empty`: remove path prefix from URL when proxying
      1. apps.y.z/webdav -> webdav:80
      2. apps.y.z./webdav/path/to/file -> webdav:80/path/to/file
    - `forward`: path remain unchanged
      1. apps.y.z/webdav -> webdav:80/webdav
      2. apps.y.z./webdav/path/to/file -> webdav:80/webdav/path/to/file

- `set_headers`: a list of header to set, (key:value, one by line)

  Duplicated keys will be treated as multiple-value headers

  ```yaml
  labels:
    proxy.app.set_headers: |
      X-Custom-Header1: value1
      X-Custom-Header1: value2
      X-Custom-Header2: value2
  ```

- `hide_headers`: comma seperated list of headers to hide

- `load_balance`: enable load balance
  - allowed: `1`, `true`

[🔼Back to top](#table-of-content)

## Troubleshooting

- Firewall issues

  If you are using `ufw` with vpn that drop all inbound traffic except vpn, run below:

  `sudo ufw allow from 172.16.0.0/16 to 100.64.0.0/10`

  Explaination:

  Docker network is usually `172.16.0.0/16`

  Tailscale is used as an example, `100.64.0.0/10` will be the CIDR

  You can also list CIDRs of all docker bridge networks by:

  `docker network inspect $(docker network ls | awk '$3 == "bridge" { print $1}') | jq -r '.[] | .Name + " " + .IPAM.Config[0].Subnet' -`

[🔼Back to top](#table-of-content)

## Docker compose examples

### Local docker provider in bridge network

```yaml
volumes:
  adg-work:
  adg-conf:
  mc-data:
  palworld:
  nginx:
services:
  adg:
    image: adguard/adguardhome
    restart: unless-stopped
    labels:
      - proxy.aliases=adg,adg-dns,adg-setup
      - proxy.adg.port=80
      - proxy.adg-setup.port=3000
      - proxy.adg-dns.scheme=udp
      - proxy.adg-dns.port=20000:dns
    volumes:
      - adg-work:/opt/adguardhome/work
      - adg-conf:/opt/adguardhome/conf
  mc:
    image: itzg/minecraft-server
    tty: true
    stdin_open: true
    container_name: mc
    restart: unless-stopped
    labels:
      - proxy.mc.scheme=tcp
      - proxy.mc.port=20001:25565
    environment:
      - EULA=TRUE
    volumes:
      - mc-data:/data
  palworld:
    image: thijsvanloef/palworld-server-docker:latest
    restart: unless-stopped
    container_name: pal
    stop_grace_period: 30s
    labels:
      - proxy.aliases=pal1,pal2
      - proxy.*.scheme=udp
      - proxy.pal1.port=20002:8211
      - proxy.pal2.port=20003:27015
    environment: ...
    volumes:
      - palworld:/palworld
  nginx:
    image: nginx
    container_name: nginx
    volumes:
      - nginx:/usr/share/nginx/html
  go-proxy:
    image: ghcr.io/yusing/go-proxy
    container_name: go-proxy
    restart: always
    ports:
      - 80:80 # http
      - 443:443 # optional, https
      - 8080:8080 # http panel
      - 8443:8443 # optional, https panel

      - 53:20000/udp # adguardhome
      - 25565:20001/tcp # minecraft
      - 8211:20002/udp # palworld
      - 27015:20003/udp # palworld
    volumes:
      - ./config:/app/config
      - /var/run/docker.sock:/var/run/docker.sock:ro
    labels:
      - proxy.aliases=gp
      - proxy.gp.port=8080
```

[🔼Back to top](#table-of-content)

#### Proxy setup

```yaml
go-proxy:
  image: ghcr.io/yusing/go-proxy
  container_name: go-proxy
  restart: always
  network_mode: host
  volumes:
    - ./config:/app/config
    - /var/run/docker.sock:/var/run/docker.sock:ro
  labels:
    - proxy.aliases=gp
    - proxy.gp.port=8080
```

[🔼Back to top](#table-of-content)

### Services URLs for above examples

- `gp.yourdomain.com`: go-proxy web panel
- `adg-setup.yourdomain.com`: adguard setup (first time setup)
- `adg.yourdomain.com`: adguard dashboard
- `nginx.yourdomain.com`: nginx
- `yourdomain.com:53`: adguard dns
- `yourdomain.com:25565`: minecraft server
- `yourdomain.com:8211`: palworld server

[🔼Back to top](#table-of-content)
