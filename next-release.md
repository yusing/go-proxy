# GoDoxy v0.8 changes:

## Breaking changes

- **Removed** `redirect_to_https` in `config.yml`, superseded by `redirectHTTP` as an entrypoint middleware

- **New** notification config format, support webhook notification, support multiple notification providers

  old

  ```yaml
  providers:
    notification:
      gotify:
        url: ...
        token: ...
  ```

  new

  ```yaml
  providers:
    notification:
      - name: gotify
        provider: gotify
        url: ...
        token: ...
      - name: discord
        provider: webhook
        url: https://discord.com/api/webhooks/...
        template: discord
  ```

  Webhook notification fields:

  | Field      | Description            | Required                       | Allowed values   |
  | ---------- | ---------------------- | ------------------------------ | ---------------- |
  | name       | name of the provider   | Yes                            |                  |
  | provider   |                        | Yes                            | `webhook`        |
  | url        | webhook URL            | Yes                            | Full URL         |
  | template   | webhook template       | No                             | empty, `discord` |
  | token      | webhook token          | No                             |                  |
  | payload    | webhook payload        | No **(if `template` is used)** | valid json       |
  | method     | webhook request method | No                             | `GET POST PUT`   |
  | mime_type  | mime type              | No                             |                  |
  | color_mode | color mode             | No                             | `hex` `dec`      |

  Available payload variables:

  | Variable | Description                 | Format                               |
  | -------- | --------------------------- | ------------------------------------ |
  | $title   | message title               | json string                          |
  | $message | message in markdown format  | json string                          |
  | $fields  | extra fields in json format | json object                          |
  | $color   | embed color by `color_mode` | `0xff0000` (hex) or `16711680` (dec) |

## Non-breaking changes

- services health notification now in markdown format like `Uptime Kuma` for both webhook and Gotify

- docker services now use docker container health status if possible, fallback to GoDoxy health check on failure / no docker health check, e.g.

  ```yaml
  # docker compose
  services:
    app:
      ...
      container_name: app
      healthcheck:
        test: ["CMD-SHELL", "curl --fail http://localhost:8080 || exit 1"]
        interval: 5s
  ```

  Health check result will be equivalent to `docker inspect --format='{{json .State.Health}}' app`

- `proxy.<alias>.path_patterns` fully support http.ServeMux patterns `[METHOD ][HOST]/[PATH]` (See https://pkg.go.dev/net/http#hdr-Patterns-ServeMux)

- caching ACME private key in order to reuse ACME account, to prevent from ACME rate limit

- **New:** fully support string as inline YAML for docker labels

  ```yaml
  services:
    app:
      ...
      labels:
        # add '|' after colon ':' to treat it as string
        proxy.app: |
          scheme: http
          host: 10.0.0.254
          port: 80
          path_patterns:
            - GET /
            - POST /auth
          healthcheck:
            disabled: false
            path: /
            interval: 5s
        proxy.app1.healthcheck: |
          path: /ping
          use_get: true
        proxy.app1.load_balance: |
          link: app
          mode: ip_hash
  ```

- **New:** support entrypoint middlewares (applied to all routes, before route middlewares)

  ```yaml
  entrypoint:
    middlewares:
      - use: CIDRWhitelist
        allow:
          - "127.0.0.1"
          - "10.0.0.0/8"
          - "192.168.0.0/16"
        status: 403
        message: "Forbidden"
  ```

- **New:** support exact host matching, i.e.

  ```yaml
  # include file
  app1.domain.tld:
    host: 10.0.0.1

  # docker compose
  services:
    app1:
      ...
      proxy.aliases: app1.domain.tld
  ```

  will only match exactly `app1.domain.tld`
  **`match_domains` in config will be ignored for this route**

- **New:** support host matching without a subdomain, i.e.

  ```yaml
  app1:
    host: 10.0.0.1
  ```

  will now also match `app1.tld`

- **New:** support `x-properties` (will be ignored, like in docker compose), useful with YAML anchor e.g.

  ```yaml
  x-proxy: &proxy # this will be ignored in GoDoxy
    scheme: https
    healthcheck:
      disable: true
    middlewares:
      hideXForwarded:
      modifyRequest:
        setHeaders:
          Host: $req_host

  api.openai.com:
    <<: *proxy # extends from x-proxy
    host: api.openai.com
  api.groq.com:
    <<: *proxy # extends from x-proxy
    host: api.groq.com
  ```

- new middleware name aliases:

  - `modifyRequest` = `request`
  - `modifyResponse` = `response`

- **New:** support `$` variables in `request` and `response` middlewares (like nginx config)

  - `$req_method`: request http method
  - `$req_scheme`: request URL scheme (http/https)
  - `$req_host`: request host without port
  - `$req_port`: request port
  - `$req_addr`: request host with port (if present)
  - `$req_path`: request URL path
  - `$req_query`: raw query string
  - `$req_url`: full request URL
  - `$req_uri`: request URI (encoded path?query)
  - `$req_content_type`: request Content-Type header
  - `$req_content_length`: length of request body (if present)
  - `$remote_addr`: client's remote address (may changed by middlewares like `RealIP` and `CloudflareRealIP`)
  - `$remote_host`: client's remote ip parse from `$remote_addr`
  - `$remote_port`: client's remote port parse from `$remote_addr` (may be empty)
  - `$resp_content_type`: response Content-Type header
  - `$resp_content_length`: length response body
  - `$status_code`: response status code
  - `$upstream_name`: upstream server name (alias)
  - `$upstream_scheme`: upstream server scheme
  - `$upstream_host`: upstream server host
  - `$upstream_port`: upstream server port
  - `$upstream_addr`: upstream server address with port (if present)
  - `$upstream_url`: full upstream server URL
  - `$header(name)`: get request header by name
  - `$resp_header(name)`: get response header by name
  - `$arg(name)`: get URL query parameter by name

- **New:** Access Logging (entrypoint and per route), i.e.

  **mount logs directory before setting this**

  ```yaml
  # config.yml
  entrypoint:
    access_log:
      format: json # common, combined, json
      path: /app/logs/access.json.log
      filters:
        cidr:
          negative: true # no log for local requests
          values:
            - 127.0.0.1/32
            - 172.0.0.0/8
            - 192.168.0.0/16
            - 10.0.0.0/16
      fields:
        headers:
          default: drop # drop app headers in log
          config: # keep only these
            X-Real-Ip: keep
            CF-Connecting-Ip: keep
            X-Forwarded-For: keep

  # include file
  # same as above but under route config
  app:
    access_log:
      format: json # common, combined, json
      ...

  # docker labels
  labels:
    proxy.app.access_log: |
      format: json
      path: /app/logs/access.json.log
      filters:
        cidr:
          negative: true
          values:
            - 127.0.0.1/32
            - 172.0.0.0/8
            - 192.168.0.0/16
            - 10.0.0.0/16
  ```

  To integrate with **goaccess**, currently need to use **caddy** as a file web server. Below should work with `combined` log format.

  ```yaml
  # compose.yml
  services:
  app:
    image: reg.6uo.me/yusing/goproxy
    ...
    volumes:
      ...
      - ./logs:/app/logs
  caddy:
    image: caddy
    restart: always
    labels:
      proxy.goaccess.port: 80
      proxy.goaccess.middlewares.request.set_headers.host: goaccess
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      - ./logs:/var/www/goaccess:ro
    depends_on:
      - goaccess
  goaccess:
    image: hectorm/goaccess:latest
    restart: always
    volumes:
      - ./logs:/srv/logs
    command: > # for combined format
      /srv/logs/access.log
      -o /srv/logs/report.html
      -j 4 # 4 threads
      --real-time-html
      --ws-url=<your goaccess url>:443 # i.e. goaccess.my.app:443/ws
      --log-format='%v %h %^[%d:%t %^] "%r" %s %b "%R" "%u"'
  ```

  Caddyfile

  ```caddyfile
  {
      auto_https off
  }

  goaccess:80 {
      @websockets {
          header Connection *Upgrade
          header Upgrade websocket
      }

      handle @websockets {
          reverse_proxy goaccess:7890
      }

      root * /var/www/goaccess
      file_server
      rewrite / /report.html
  }
  ```

## Fixes

- duplicated notification after config reload
- `timeout` was defaulted to `0` in some cases causing health check to fail
- `redirectHTTP` middleware may not work on non standard http port
- various other small bugs
- `realIP` and `cloudflareRealIP` middlewares
- prometheus metrics gone after a single route reload
- WebUI app links now works when `match_domains` is not set
- WebUI config editor now display validation errors properly
- upgraded dependencies to the latest
