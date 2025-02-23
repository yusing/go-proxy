<div align="center">

# GoDoxy

[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=yusing_go-proxy&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=yusing_godoxy)
![GitHub last commit](https://img.shields.io/github/last-commit/yusing/godoxy)
[![Lines of Code](https://sonarcloud.io/api/project_badges/measure?project=yusing_go-proxy&metric=ncloc)](https://sonarcloud.io/summary/new_code?id=yusing_godoxy)
[![](https://dcbadge.limes.pink/api/server/umReR62nRd?style=flat)](https://discord.gg/umReR62nRd)

A lightweight, simple, and [performant](https://github.com/yusing/godoxy/wiki/Benchmarks) reverse proxy with WebUI.

For full documentation, check out **[Wiki](https://github.com/yusing/godoxy/wiki)**

**EN** | <a href="README_CHT.md">中文</a>

<!-- [![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=yusing_godoxy&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=yusing_godoxy)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=yusing_godoxy&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=yusing_godoxy)
[![Vulnerabilities](https://sonarcloud.io/api/project_badges/measure?project=yusing_godoxy&metric=vulnerabilities)](https://sonarcloud.io/summary/new_code?id=yusing_godoxy) -->

<img src="screenshots/webui.png" style="max-width: 650">

</div>

## Table of content

<!-- TOC -->

- [GoDoxy](#godoxy)
  - [Table of content](#table-of-content)
  - [Key Features](#key-features)
  - [Prerequisites](#prerequisites)
  - [Setup](#setup)
    - [Manual Setup](#manual-setup)
    - [Folder structrue](#folder-structrue)
  - [Screenshots](#screenshots)
    - [idlesleeper](#idlesleeper)
  - [Build it yourself](#build-it-yourself)

## Key Features

- Easy to use
  - Effortless configuration
  - Simple multi-node setup
  - Error messages is clear and detailed, easy troubleshooting
- Auto SSL cert management (See [Supported DNS-01 Challenge Providers](https://github.com/yusing/godoxy/wiki/Supported-DNS%E2%80%9001-Providers))
- Auto configuration for docker containers
- Auto hot-reload on container state / config file changes
- **idlesleeper**: stop containers on idle, wake it up on traffic _(optional, see [screenshots](#idlesleeper))_
- HTTP(s) reserve proxy
- OpenID Connect support
- [HTTP middleware support](https://github.com/yusing/godoxy/wiki/Middlewares)
- [Custom error pages support](https://github.com/yusing/godoxy/wiki/Middlewares#custom-error-pages)
- TCP and UDP port forwarding
- **Web UI with App dashboard and config editor**
- Supports linux/amd64, linux/arm64
- Written in **[Go](https://go.dev)**

[🔼Back to top](#table-of-content)

## Prerequisites

Setup DNS Records point to machine which runs `GoDoxy`, e.g.

- A Record: `*.y.z` -> `10.0.10.1`
- AAAA Record: `*.y.z` -> `::ffff:a00:a01`

## Setup

**NOTE:** GoDoxy is designed to be (and only works when) running in `host` network mode, do not change it. To change listening ports, modify `.env`.

1. Prepare a new directory for docker compose and config files.

2. Run setup script inside the directory, or [set up manually](#manual-setup)

    ```shell
    /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/yusing/godoxy/main/scripts/setup.sh)"
    ```

3. Start the container `docker compose up -d` and wait for it to be ready

4. You may now do some extra configuration on WebUI `https://godoxy.yourdomain.com`

[🔼Back to top](#table-of-content)

### Manual Setup

1. Make `config` directory then grab `config.example.yml` into `config/config.yml`

   `mkdir -p config && wget https://raw.githubusercontent.com/yusing/godoxy/main/config.example.yml -O config/config.yml`

2. Grab `.env.example` into `.env`

   `wget https://raw.githubusercontent.com/yusing/godoxy/main/.env.example -O .env`

3. Grab `compose.example.yml` into `compose.yml`

   `wget https://raw.githubusercontent.com/yusing/godoxy/main/compose.example.yml -O compose.yml`

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
├── data
│   ├── metrics # metrics data
│   │   ├── uptime.json
│   │   └── system_info.json
└── .env
```

## Screenshots

### idlesleeper

![idlesleeper](screenshots/idlesleeper.webp)

### Uptime Monitor

![uptime monitor](screenshots/uptime.png)

### System Monitor

![system monitor](screenshots/system-monitor.png)

[🔼Back to top](#table-of-content)

## Build it yourself

1. Clone the repository `git clone https://github.com/yusing/godoxy --depth=1`

2. Install / Upgrade [go (>=1.22)](https://go.dev/doc/install) and `make` if not already

3. Clear cache if you have built this before (go < 1.22) with `go clean -cache`

4. get dependencies with `make get`

5. build binary with `make build`

[🔼Back to top](#table-of-content)
