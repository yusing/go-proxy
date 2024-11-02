# go-proxy

[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=yusing_go-proxy&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=yusing_go-proxy)
[![Lines of Code](https://sonarcloud.io/api/project_badges/measure?project=yusing_go-proxy&metric=ncloc)](https://sonarcloud.io/summary/new_code?id=yusing_go-proxy)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=yusing_go-proxy&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=yusing_go-proxy)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=yusing_go-proxy&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=yusing_go-proxy)
[![Vulnerabilities](https://sonarcloud.io/api/project_badges/measure?project=yusing_go-proxy&metric=vulnerabilities)](https://sonarcloud.io/summary/new_code?id=yusing_go-proxy)
[![](https://dcbadge.limes.pink/api/server/umReR62nRd)](https://discord.gg/umReR62nRd)

[繁體中文文檔請看此](README_CHT.md)

A lightweight, easy-to-use, and [performant](https://github.com/yusing/go-proxy/wiki/Benchmarks) reverse proxy with a Web UI and dashboard.

![Screenshot](screenshots/webui.png)

_Join our [Discord](https://discord.gg/umReR62nRd) for help and discussions_

## Table of content

<!-- TOC -->

- [go-proxy](#go-proxy)
  - [Table of content](#table-of-content)
  - [Key Features](#key-features)
  - [Getting Started](#getting-started)
    - [Setup](#setup)
    - [Manual Setup](#manual-setup)
    - [Use JSON Schema in VSCode](#use-json-schema-in-vscode)
  - [Screenshots](#screenshots)
    - [idlesleeper](#idlesleeper)
  - [Build it yourself](#build-it-yourself)

## Key Features

-   Easy to use
    -   Effortless configuration
    -   Simple multi-node setup
    -   Error messages is clear and detailed, easy troubleshooting
-   Auto SSL cert management (See [Supported DNS-01 Challenge Providers](https://github.com/yusing/go-proxy/wiki/Supported-DNS%E2%80%9001-Providers)) 
-   Auto configuration for docker containers
-   Auto hot-reload on container state / config file changes
-   **idlesleeper**: stop containers on idle, wake it up on traffic _(optional, see [screenshots](#idlesleeper))_
-   HTTP(s) reserve proxy
-   [HTTP middleware support](https://github.com/yusing/go-proxy/wiki/Middlewares)
-   [Custom error pages support](https://github.com/yusing/go-proxy/wiki/Middlewares#custom-error-pages)
-   TCP and UDP port forwarding
-   **Web UI with App dashboard**
-   Supports linux/amd64, linux/arm64
-   Written in **[Go](https://go.dev)**

[🔼Back to top](#table-of-content)

## Getting Started

### Setup

1.  Pull docker image 
    
    ```shell
    docker pull ghcr.io/yusing/go-proxy:latest
    ```

2.  Create new directory, `cd` into it, then run setup, or [set up manually](#manual-setup) 

    ```shell
    docker run --rm -v .:/setup ghcr.io/yusing/go-proxy /app/go-proxy setup
    # Then set the JWT secret
    sed -i "s|GOPROXY_API_JWT_SECRET=.*|GOPROXY_API_JWT_SECRET=$(openssl rand -base64 32)|g" .env
    ```

3.  Setup DNS Records point to machine which runs `go-proxy`, e.g.

    -   A Record: `*.y.z` -> `10.0.10.1`
    -   AAAA Record: `*.y.z` -> `::ffff:a00:a01`

4.  Setup `docker-socket-proxy` other docker nodes _(if any)_ (see [Multi docker nodes setup](https://github.com/yusing/go-proxy/wiki/Configurations#multi-docker-nodes-setup)) and then them inside `config.yml`

5.  Run go-proxy `docker compose up -d` 
    then list all routes to see if further configurations are needed:
    `docker exec go-proxy /app/go-proxy ls-routes`

6.  You may now do some extra configuration
    -   With text editor (e.g. Visual Studio Code)
    -   With Web UI via `http://localhost:3000` or `https://gp.y.z`
    -   For more info, [See Wiki]([wiki](https://github.com/yusing/go-proxy/wiki))

[🔼Back to top](#table-of-content)

### Manual Setup

1. Make `config` directory then grab `config.example.yml` into `config/config.yml`
  `mkdir -p config && wget https://raw.githubusercontent.com/yusing/go-proxy/v0.7/config.example.yml -O config/config.yml`

2. Grab `.env.example` into `.env`
  `wget https://raw.githubusercontent.com/yusing/go-proxy/v0.7/.env.example -O .env`

3. Grab `compose.example.yml` into `compose.yml`
   `wget https://raw.githubusercontent.com/yusing/go-proxy/v0.7/compose.example.yml -O compose.yml`

4. Set the JWT secret
   `sed -i "s|GOPROXY_API_JWT_SECRET=.*|GOPROXY_API_JWT_SECRET=$(openssl rand -base64 32)|g" .env`

5. Start the container `docker compose up -d`

### Use JSON Schema in VSCode

Copy [`.vscode/settings.example.json`](.vscode/settings.example.json) to `.vscode/settings.json` and modify it to fit your needs

[🔼Back to top](#table-of-content)

## Screenshots

### idlesleeper

![idlesleeper](screenshots/idlesleeper.webp)


[🔼Back to top](#table-of-content)

## Build it yourself

1. Clone the repository `git clone https://github.com/yusing/go-proxy --depth=1`

2. Install / Upgrade [go (>=1.22)](https://go.dev/doc/install) and `make` if not already

3. Clear cache if you have built this before (go < 1.22) with `go clean -cache`

4. get dependencies with `make get`

5. build binary with `make build`

[🔼Back to top](#table-of-content)
