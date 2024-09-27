# Middlewares

## Table of content

<!-- TOC -->

- [Middlewares](#middlewares)
  - [Table of content](#table-of-content)
  - [Available middlewares](#available-middlewares)
    - [Redirect http](#redirect-http)
    - [Modify request or response](#modify-request-or-response)
      - [Set headers](#set-headers)
      - [Add headers](#add-headers)
      - [Hide headers](#hide-headers)
    - [X-Forwarded-\* Headers](#x-forwarded--headers)
      - [Add X-Forwarded-\*](#add-x-forwarded-)
      - [Set X-Forwarded-\*](#set-x-forwarded-)
    - [Forward Authorization header (experimental)](#forward-authorization-header-experimental)
  - [Examples](#examples)
    - [Authentik](#authentik)

<!-- TOC -->

## Available middlewares

### Redirect http

Redirect http requests to https

```yaml
# docker labels
proxy.app1.middlewares.redirect_http:

# include file
app1:
  middlewares:
    redirect_http:
```

nginx equivalent:
```nginx
server {
    listen 80;
    server_name domain.tld;
    return 301 https://$host$request_uri;
}
```

[🔼Back to top](#table-of-content)

### Modify request or response

```yaml
# docker labels
proxy.app1.middlewares.modify_request.field:
proxy.app1.middlewares.modify_response.field:

# include file
app1:
  middlewares:
    modify_request:
      field:
    modify_response:
      field:
```

#### Set headers

```yaml
# docker labels
proxy.app1.middlewares.modify_request.set_headers: |
  X-Custom-Header1: value1, value2
  X-Custom-Header2: value3

# include file
app1:
  middlewares:
    modify_request:
      set_headers:
        X-Custom-Header1: value1, value2
        X-Custom-Header2: value3
```

nginx equivalent:
```nginx
location / {
    add_header X-Custom-Header1 value1, value2;
    add_header X-Custom-Header2 value3;
}
```

#### Add headers

```yaml
# docker labels
proxy.app1.middlewares.modify_request.add_headers: |
  X-Custom-Header1: value1, value2
  X-Custom-Header2: value3

# include file
app1:
  middlewares:
    modify_request:
      add_headers:
        X-Custom-Header1: value1, value2
        X-Custom-Header2: value3
```

nginx equivalent:
```nginx
location / {
    more_set_headers "X-Custom-Header1: value1, value2";
    more_set_headers "X-Custom-Header2: value3";
}
```

#### Hide headers

```yaml
# docker labels
proxy.app1.middlewares.modify_request.hide_headers: |
  - X-Custom-Header1
  - X-Custom-Header2

# include file
app1:
  middlewares:
    modify_request:
      hide_headers:
        - X-Custom-Header1
        - X-Custom-Header2
```

nginx equivalent:
```nginx
location / {
    more_clear_headers "X-Custom-Header1";
    more_clear_headers "X-Custom-Header2";
}
```

### X-Forwarded-* Headers

#### Add X-Forwarded-*

Append `X-Forwarded-*` headers to existing headers

```yaml
# docker labels
proxy.app1.middlewares.modify_request.add_x_forwarded:

# include file
app1:
  middlewares:
    modify_request:
      add_x_forwarded:
```

#### Set X-Forwarded-*

Replace existing `X-Forwarded-*` headers with `go-proxy` provided headers

```yaml
# docker labels
proxy.app1.middlewares.modify_request.set_x_forwarded:

# include file
app1:
  middlewares:
    modify_request:
      set_x_forwarded:
```

### Forward Authorization header (experimental)

Fields:
- `address`: authentication provider URL _(required)_
- `trust_forward_header`: whether to trust `X-Forwarded-*` headers from upstream proxies _(default: `false`)_
- `auth_response_headers`: list of headers to copy from auth response _(default: empty)_
- `add_auth_cookies_to_response`: list of cookies to add to response _(default: empty)_

```yaml
# docker labels
proxy.app1.middlewares.forward_auth.address: https://auth.example.com
proxy.app1.middlewares.forward_auth.trust_forward_header: true
proxy.app1.middlewares.forward_auth.auth_response_headers: |
  - X-Auth-Token
  - X-Auth-User
proxy.app1.middlewares.forward_auth.add_auth_cookies_to_response: |
  - uid
  - session_id

# include file
app1:
  middlewares:
    forward_authorization:
        address: https://auth.example.com
        trust_forward_header: true
        auth_response_headers:
            - X-Auth-Token
            - X-Auth-User
        add_auth_cookies_to_response:
            - uid
            - session_id
```

Traefik equivalent:
```yaml
# docker labels
traefik.http.middlewares.authentik.forwardauth.address: https://auth.example.com
traefik.http.middlewares.authentik.forwardauth.trustForwardHeader: true
traefik.http.middlewares.authentik.forwardauth.authResponseHeaders: X-Auth-Token, X-Auth-User
traefik.http.middlewares.authentik.forwardauth.addAuthCookiesToResponse: uid, session_id

# standalone
http:
  middlewares:
    forwardAuth:
        address: https://auth.example.com
        trustForwardHeader: true
        authResponseHeaders:
            - X-Auth-Token
            - X-Auth-User
        addAuthCookiesToResponse:
            - uid
            - session_id
```

## Examples

### Authentik

```yaml
# docker compose
services:
  ...
  server:
    ...
    container_name: authentik
    labels:
      proxy.authentik.middlewares.redirect_http:
      proxy.authentik.middlewares.set_x_forwarded:
      proxy.authentik.middlewares.modify_request.add_headers: |
        Strict-Transport-Security: "max-age=63072000" always
```