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

    -   Ubuntu based: `sudo apt install -y wget`
    -   Fedora based: `sudo yum install -y wget`
    -   Arch based: `sudo pacman -Sy wget`

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

| Label                    | Description                                                                                                                                                         | Example                        | Default                     | Accepted values                                                           |
| ------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------ | --------------------------- | ------------------------------------------------------------------------- |
| `proxy.aliases`          | comma separated aliases for subdomain and label matching                                                                                                            | `gitlab,gitlab-reg,gitlab-ssh` | `container_name`            | any                                                                       |
| `proxy.exclude`          | to be excluded from `go-proxy`                                                                                                                                      |                                | false                       | boolean                                                                   |
| `proxy.idle_timeout`     | time for idle (no traffic) before put it into sleep **(http/s only)**<br> _**NOTE: idlewatcher will only be enabled containers that has non-empty `idle_timeout`**_ | `1h`                           | empty or `0` **(disabled)** | `number[unit]...`, e.g. `1m30s`                                           |
| `proxy.wake_timeout`     | time to wait for target site to be ready                                                                                                                            |                                | `10s`                       | `number[unit]...`                                                         |
| `proxy.stop_method`      | method to stop after `idle_timeout`                                                                                                                                 |                                | `stop`                      | `stop`, `pause`, `kill`                                                   |
| `proxy.stop_timeout`     | time to wait for stop command                                                                                                                                       |                                | `10s`                       | `number[unit]...`                                                         |
| `proxy.stop_signal`      | signal sent to container for `stop` and `kill` methods                                                                                                              |                                | docker's default            | `SIGINT`, `SIGTERM`, `SIGHUP`, `SIGQUIT` and those without **SIG** prefix |
| `proxy.<alias>.<field>`  | set field for specific alias                                                                                                                                        | `proxy.gitlab-ssh.scheme`      | N/A                         | N/A                                                                       |
| `proxy.$<index>.<field>` | set field for specific alias at index (starting from **1**)                                                                                                         | `proxy.$3.port`                | N/A                         | N/A                                                                       |
| `proxy.*.<field>`        | set field for all aliases                                                                                                                                           | `proxy.*.set_headers`          | N/A                         | N/A                                                                       |

### Fields

| Field                 | Description                                                                                    | Default                                                                          | Allowed Values / Syntax                                                                                                                                                                                                 |
| --------------------- | ---------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `scheme`              | proxy protocol                                                                                 | <ul><li>`http` for numeric port</li><li>`tcp` for `x:y` port</li></ul>           | `http`, `https`, `tcp`, `udp`                                                                                                                                                                                           |
| `host`                | proxy host                                                                                     | <ul><li>Docker: docker client IP / hostname </li><li>File: `localhost`</li></ul> | IP address, hostname                                                                                                                                                                                                    |
| `port`                | proxy port **(http/s)**                                                                        | first port returned from docker                                                  | number in range of `1 - 65535`                                                                                                                                                                                          |
| `port` **(required)** | proxy port **(tcp/udp)**                                                                       | N/A                                                                              | `x:y` <br><ul><li>**x**: port for `go-proxy` to listen on.<br>**x** can be 0, which means listen on a random port</li><li>**y**: port or [_service name_](../src/common/constants.go#L55) of target container</li></ul> |
| `no_tls_verify`       | whether skip tls verify **(https only)**                                                       | `false`                                                                          | boolean                                                                                                                                                                                                                 |
| `path_patterns`       | proxy path patterns **(http/s only)**<br> only requests that matched a pattern will be proxied | empty **(proxy all requests)**                                                   | yaml style list[<sup>1</sup>](#list-example) of ([path patterns](https://pkg.go.dev/net/http#hdr-Patterns-ServeMux))                                                                                                    |
| `set_headers`         | header to set **(http/s only)**                                                                | empty                                                                            | yaml style key-value mapping[<sup>2</sup>](#key-value-mapping-example) of header-value pairs                                                                                                                            |
| `hide_headers`        | header to hide **(http/s only)**                                                               | empty                                                                            | yaml style list[<sup>1</sup>](#list-example) of headers                                                                                                                                                                 |

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

-   Container not showing up in proxies list

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
        network_mode: host
        labels:
          proxy.nginx-2.port: 80
    ```

-   Firewall issues

    If you are using `ufw` with vpn that drop all inbound traffic except vpn, run below:

    `sudo ufw allow from 172.16.0.0/16 to 100.64.0.0/10`

    Explaination:

    Docker network is usually `172.16.0.0/16`

    Tailscale is used as an example, `100.64.0.0/10` will be the CIDR

    You can also list CIDRs of all docker bridge networks by:

    `docker network inspect $(docker network ls | awk '$3 == "bridge" { print $1}') | jq -r '.[] | .Name + " " + .IPAM.Config[0].Subnet' -`

[🔼Back to top](#table-of-content)

## Docker compose examples

More examples in [here](examples/)

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
            - proxy.$1.port=80
            - proxy.$2.scheme=udp
            - proxy.$2.port=20000:dns
            - proxy.$3.port=3000
        volumes:
            - adg-work:/opt/adguardhome/work
            - adg-conf:/opt/adguardhome/conf
        ports:
            - 80
            - 3000
            - 53/udp
    mc:
        image: itzg/minecraft-server
        tty: true
        stdin_open: true
        container_name: mc
        restart: unless-stopped
        ports:
            - 25565
        labels:
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
        ports:
            - 8211/udp
            - 27015/udp
        labels:
            - proxy.aliases=pal1,pal2
            - proxy.*.scheme=udp
            - proxy.$1.port=20002:8211
            - proxy.$2.port=20003:27015
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
        labels:
            proxy.idle_timeout: 1m
    go-proxy:
        image: ghcr.io/yusing/go-proxy:latest
        container_name: go-proxy
        restart: always
        network_mode: host
        volumes:
            - ./config:/app/config
            - /var/run/docker.sock:/var/run/docker.sock
    go-proxy-frontend:
        image: ghcr.io/yusing/go-proxy-frontend:latest
        container_name: go-proxy-frontend
        restart: unless-stopped
        network_mode: host
        labels:
            - proxy.aliases=gp
            - proxy.gp.port=3000
        depends_on:
            - go-proxy
```

[🔼Back to top](#table-of-content)

### Services URLs for above examples

-   `gp.yourdomain.com`: go-proxy web panel
-   `adg-setup.yourdomain.com`: adguard setup (first time setup)
-   `adg.yourdomain.com`: adguard dashboard
-   `nginx.yourdomain.com`: nginx
-   `yourdomain.com:2000`: adguard dns (udp)
-   `yourdomain.com:20001`: minecraft server
-   `yourdomain.com:20002`: palworld server

[🔼Back to top](#table-of-content)
