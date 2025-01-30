# GoDoxy

[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=yusing_go-proxy&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=yusing_go-proxy)
[![Lines of Code](https://sonarcloud.io/api/project_badges/measure?project=yusing_go-proxy&metric=ncloc)](https://sonarcloud.io/summary/new_code?id=yusing_go-proxy)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=yusing_go-proxy&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=yusing_go-proxy)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=yusing_go-proxy&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=yusing_go-proxy)
[![Vulnerabilities](https://sonarcloud.io/api/project_badges/measure?project=yusing_go-proxy&metric=vulnerabilities)](https://sonarcloud.io/summary/new_code?id=yusing_go-proxy)
[![](https://dcbadge.limes.pink/api/server/umReR62nRd)](https://discord.gg/umReR62nRd)

[繁體中文文檔請看此](README_CHT.md)

A lightweight, easy-to-use, and [performant](https://github.com/yusing/go-proxy/wiki/Benchmarks) reverse proxy with a Web UI and dashboard.

![Screenshot](https://github.com/user-attachments/assets/4bb371f4-6e4c-425c-89b2-b9e962bdd46f)

_Join our [Discord](https://discord.gg/umReR62nRd) for help and discussions_

## Table of content

<!-- TOC -->

- [GoDoxy](#godoxy)
  - [Table of content](#table-of-content)
  - [Key Features](#key-features)
  - [Getting Started](#getting-started)
    - [Prerequisites](#prerequisites)
    - [Setup](#setup)
    - [Manual Setup](#manual-setup)
    - [Folder structrue](#folder-structrue)
    - [Use JSON Schema in VSCode](#use-json-schema-in-vscode)
  - [Screenshots](#screenshots)
    - [idlesleeper](#idlesleeper)
  - [Build it yourself](#build-it-yourself)

## Key Features

- Easy to use
  - Effortless configuration
  - Simple multi-node setup
  - Error messages is clear and detailed, easy troubleshooting
- Auto SSL cert management (See [Supported DNS-01 Challenge Providers](https://github.com/yusing/go-proxy/wiki/Supported-DNS%E2%80%9001-Providers))
- Auto configuration for docker containers
- Auto hot-reload on container state / config file changes
- **idlesleeper**: stop containers on idle, wake it up on traffic _(optional, see [screenshots](#idlesleeper))_
- HTTP(s) reserve proxy
- OpenID Connect support
- [HTTP middleware support](https://github.com/yusing/go-proxy/wiki/Middlewares)
- [Custom error pages support](https://github.com/yusing/go-proxy/wiki/Middlewares#custom-error-pages)
- TCP and UDP port forwarding
- **Web UI with App dashboard and config editor**
- Supports linux/amd64, linux/arm64
- Written in **[Go](https://go.dev)**

[🔼Back to top](#table-of-content)

## Getting Started

For full documentation, **[See Wiki](https://github.com/yusing/go-proxy/wiki)**

### Prerequisites

Setup DNS Records point to machine which runs `GoDoxy`, e.g.

- A Record: `*.y.z` -> `10.0.10.1`
- AAAA Record: `*.y.z` -> `::ffff:a00:a01`

### Setup

1.  Pull the latest docker images

    ```shell
    docker pull ghcr.io/yusing/go-proxy:latest
    ```

2.  Create new directory, `cd` into it, then run setup, or [set up manually](#manual-setup)

    ```shell
    docker run --rm -v .:/setup ghcr.io/yusing/go-proxy /app/godoxy setup
    ```

3.  _(Optional)_ setup WebUI login (skip if you use OIDC)

    - set random JWT secret

      ```shell
      sed -i "s|API_JWT_SECRET=.*|API_JWT_SECRET=$(openssl rand -base64 32)|g" .env
      ```

    - change username and password for WebUI authentication
      ```shell
      USERNAME=admin
      PASSWORD=some-password
      sed -i "s|API_USERNAME=.*|API_USERNAME=${USERNAME}|g" .env
      sed -i "s|API_PASSWORD=.*|API_PASSWORD=${PASSWORD}|g" .env
      ```

4.  _(Optional)_ setup `docker-socket-proxy` other docker nodes (see [Multi docker nodes setup](https://github.com/yusing/go-proxy/wiki/Configurations#multi-docker-nodes-setup)) then add them inside `config.yml`

5.  Start the container `docker compose up -d`

6.  You may now do some extra configuration on WebUI `https://gp.y.z`

[🔼Back to top](#table-of-content)

### Manual Setup

1. Make `config` directory then grab `config.example.yml` into `config/config.yml`

   `mkdir -p config && wget https://raw.githubusercontent.com/yusing/go-proxy/v0.9/config.example.yml -O config/config.yml`

2. Grab `.env.example` into `.env`

   `wget https://raw.githubusercontent.com/yusing/go-proxy/v0.9/.env.example -O .env`

3. Grab `compose.example.yml` into `compose.yml`

   `wget https://raw.githubusercontent.com/yusing/go-proxy/v0.9/compose.example.yml -O compose.yml`

### Folder structrue

```shell
├── certs
│   ├── cert.crt
│   └── priv.key
├── compose.yml
├── config
│   ├── config.yml
│   ├── middlewares
│   │   ├── middleware1.yml
│   │   ├── middleware2.yml
│   ├── provider1.yml
│   └── provider2.yml
└── .env
```

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
