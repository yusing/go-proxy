# Docker compose guide

## Table of content

<!-- TOC -->

- [Docker compose guide](#docker-compose-guide)
  - [Table of content](#table-of-content)
  - [Setup](#setup)
  - [Labels](#labels)
    - [Syntax](#syntax)
    - [Fields](#fields)
      - [Key-value mapping example](#key-value-mapping-example)
      - [List example](#list-example)
  - [Troubleshooting](#troubleshooting)
  - [Docker compose examples](#docker-compose-examples)
    - [Services URLs for above examples](#services-urls-for-above-examples)

## Setup

1.  Install `wget` if not already

    - Ubuntu based: `sudo apt install -y wget`
    - Fedora based: `sudo yum install -y wget`
    - Arch based: `sudo pacman -Sy wget`

2.  Run setup script

    `bash <(wget -qO- https://github.com/yusing/go-proxy/raw/main/setup-docker.sh)`

    It will setup folder structure and required config files

3.  Verify folder structure and then `cd go-proxy`

    ```plain
    go-proxy
    ├── certs
    ├── compose.yml
    └── config
        ├── config.yml
        └── providers.yml
    ```

4.  Enable HTTPs _(optional)_

    Mount a folder (to store obtained certs) or (containing existing cert)

    ```yaml
    services:
      go-proxy:
        ...
        volumes:
          - ./certs:/app/certs
    ```

    To use **autocert**, complete that section in `config.yml`, e.g.

    ```yaml
    autocert:
      email: john.doe@x.y.z # ACME Email
      domains: # a list of domains for cert registration
        - x.y.z
      provider: cloudflare
      options:
        - auth_token: c1234565789-abcdefghijklmnopqrst # your zone API token
    ```

    To use **existing certificate**, set path for cert and key in `config.yml`, e.g.

    ```yaml
    autocert:
      cert_path: /app/certs/cert.crt
      key_path: /app/certs/priv.key
    ```

5.  Modify `compose.yml` to fit your needs

6.  Run `docker compose up -d` to start the container

7.  Navigate to Web panel `http://gp.yourdomain.com` or use **Visual Studio Code (provides schema check)** to edit proxy config

[🔼Back to top](#table-of-content)

## Labels

### Syntax

| Label                   | Description                                              | Default          |
| ----------------------- | -------------------------------------------------------- | ---------------- |
| `proxy.aliases`         | comma separated aliases for subdomain and label matching | `container_name` |
| `proxy.exclude`         | to be excluded from `go-proxy`                           | false            |
| `proxy.<alias>.<field>` | set field for specific alias                             | N/A              |
| `proxy.*.<field>`       | set field for all aliases                                | N/A              |

### Fields

| Field                 | Description                                                                                    | Default                                                                          | Allowed Values / Syntax                                                                                                                                 |
| --------------------- | ---------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `scheme`              | proxy protocol                                                                                 | <ul><li>`http` for numeric port</li><li>`tcp` for `x:y` port</li></ul>           | `http`, `https`, `tcp`, `udp`                                                                                                                           |
| `host`                | proxy host                                                                                     | <ul><li>Docker: docker client IP / hostname </li><li>File: `localhost`</li></ul> | IP address, hostname                                                                                                                                    |
| `port`                | proxy port **(http/s)**                                                                        | first port in `ports:`                                                           | number in range of `1 - 65535`                                                                                                                          |
| `port` **(required)** | proxy port **(tcp/udp)**                                                                       | N/A                                                                              | `x:y` <br><ul><li>x: port for `go-proxy` to listen on</li><li>y: port or [_service name_](../src/common/constants.go#L55) of target container</li></ul> |
| `no_tls_verify`       | whether skip tls verify **(https only)**                                                       | `false`                                                                          | boolean                                                                                                                                                 |
| `path_patterns`       | proxy path patterns **(http/s only)**<br> only requests that matched a pattern will be proxied | empty **(proxy all requests)**                                                   | yaml style list[<sup>1</sup>](#list-example) of path patterns ([syntax](https://pkg.go.dev/net/http#hdr-Patterns-ServeMux))                             |
| `set_headers`         | header to set **(http/s only)**                                                                | empty                                                                            | yaml style key-value mapping[<sup>2</sup>](#key-value-mapping-example) of header-value pairs                                                            |
| `hide_headers`        | header to hide **(http/s only)**                                                               | empty                                                                            | yaml style list[<sup>1</sup>](#list-example) of headers                                                                                                 |

[🔼Back to top](#table-of-content)

#### Key-value mapping example

Docker Compose

```yaml
services:
  nginx:
    ...
    labels:
      # values from duplicated header keys will be combined
      proxy.nginx.set_headers: | # remember to add the '|'
        X-Custom-Header1: value1, value2
        X-Custom-Header2: value3
        X-Custom-Header2: value4
      # X-Custom-Header2 will be "value3, value4"
```

File Provider

```yaml
service_a:
  host: service_a.internal
  set_headers:
    # do not duplicate header keys, as it is not allowed in YAML
    X-Custom-Header1: value1, value2
    X-Custom-Header2: value3
```

[🔼Back to top](#table-of-content)

#### List example

Docker Compose

```yaml
services:
  nginx:
    ...
    labels:
      proxy.nginx.path_patterns: | # remember to add the '|'
        - GET /
        - POST /auth
      proxy.nginx.hide_headers: | # remember to add the '|'
        - X-Custom-Header1
        - X-Custom-Header2
```

File Provider

```yaml
service_a:
  host: service_a.internal
  path_patterns:
    - GET /
    - POST /auth
  hide_headers:
    - X-Custom-Header1
    - X-Custom-Header2
```

[🔼Back to top](#table-of-content)

## Troubleshooting

- Container not showing up in proxies list

  Please check that either `ports` or label `proxy.<alias>.port` is declared, e.g.

  ```yaml
  services:
    nginx-1: # Option 1
      ...
      ports:
        - 80
    nginx-2: # Option 2
      ...
      container_name: nginx-2
      labels:
        proxy.nginx-2.port: 80
  ```

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
    ports:
      - 80
  go-proxy:
    image: ghcr.io/yusing/go-proxy:latest
    container_name: go-proxy
    restart: always
    network_mode: host
    volumes:
      - ./config:/app/config
      - /var/run/docker.sock:/var/run/docker.sock:ro
  go-proxy-frontend:
    image: ghcr.io/yusing/go-proxy-frontend:latest
    container_name: go-proxy-frontend
    restart: unless-stopped
    network_mode: host
    labels:
      - proxy.aliases=gp
      - proxy.gp.port=8888
    depends_on:
      - go-proxy
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
