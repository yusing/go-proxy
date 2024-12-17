GoDoxy v0.8 changes:

- **Breaking** notification config format changed, support webhook notification, support multiple notification providers
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

- **Breaking** removed `redirect_to_https` in `config.yml`, superseded by `redirectHTTP` as an entrypoint middleware

- services health notification now in markdown format like `Uptime Kuma` for both webhook and Gotify

- docker services use docker now health check if possible, fallback to GoDoxy health check on failure / no docker health check

- support entrypoint middlewares (applied to routes, before route middlewares)

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

- support exact host matching, i.e.

```yaml
app1.domain.tld:
  host: 10.0.0.1
```

will only match exactly `app1.domain.tld`
**If `match_domains` are used in config, `domain.tld` must be one of it**

- support `x-properties` (like in docker compose), example usage

```yaml
x-proxy: &proxy
  scheme: https
  healthcheck:
    disable: true
  middlewares:
    hideXForwarded:
    modifyRequest:
      setHeaders:
        Host: $req_host

api.openai.com:
  <<: *proxy
  host: api.openai.com
api.groq.com:
  <<: *proxy
  host: api.groq.com
```

- new middleware name aliases:

  - `modifyRequest` = `request`
  - `modifyResponse` = `response`

- support `$` variables in `request` and `response` middlewares (like nginx config)

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

- `proxy.<alias>.path_patterns` fully support http.ServeMux patterns `[METHOD ][HOST]/[PATH]` (See https://pkg.go.dev/net/http#hdr-Patterns-ServeMux)

- caching ACME private key in order to reuse ACME account, to prevent from ACME rate limit

- fixed
  - duplicated notification after config reload
  - `timeout` was defaulted to `0` in some cases causing health check to fail
  - `redirectHTTP` middleware may not work on non standard http port
  - various other small bugs
