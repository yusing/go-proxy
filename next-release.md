GoDoxy v0.8.2 expected changes

- **Thanks [polds](https://github.com/polds)**
  Optionally allow a user to specify a “warm-up” endpoint to start the container, returning a 403 if the endpoint isn’t hit and the container has been stopped.

  This can help prevent bots from starting random containers, or allow health check systems to run some probes. Or potentially lock the start endpoints behind a different authentication mechanism, etc.

  Sample service showing this:

  ```yaml
  hello-world:
    image: nginxdemos/hello
    container_name: hello-world
    restart: "no"
    ports:
      - "9100:80"
    labels:
      proxy.aliases: hello-world
      proxy.#1.port: 9100
      proxy.idle_timeout: 45s
      proxy.wake_timeout: 30s
      proxy.stop_method: stop
      proxy.stop_timeout: 10s
      proxy.stop_signal: SIGTERM
      proxy.start_endpoint: "/start"
  ```

  Hitting `/` on this service when the container is down:

  ```curl
  $ curl -sv -X GET -H "Host: hello-world.godoxy.local" http://localhost/
  * Host localhost:80 was resolved.
  * IPv6: ::1
  * IPv4: 127.0.0.1
  *   Trying [::1]:80...
  * Connected to localhost (::1) port 80
  > GET / HTTP/1.1
  > Host: hello-world.godoxy.local
  > User-Agent: curl/8.7.1
  > Accept: */*
  >
  * Request completely sent off
  < HTTP/1.1 403 Forbidden
  < Content-Type: text/plain; charset=utf-8
  < X-Content-Type-Options: nosniff
  < Date: Wed, 08 Jan 2025 02:04:51 GMT
  < Content-Length: 71
  <
  Forbidden: Container can only be started via configured start endpoint
  * Connection #0 to host localhost left intact
  ```

  Hitting `/start` when the container is down:

  ```curl
  curl -sv -X GET -H "Host: hello-world.godoxy.local" -H "X-Goproxy-Check-Redirect: skip" http://localhost/start
  * Host localhost:80 was resolved.
  * IPv6: ::1
  * IPv4: 127.0.0.1
  *   Trying [::1]:80...
  * Connected to localhost (::1) port 80
  > GET /start HTTP/1.1
  > Host: hello-world.godoxy.local
  > User-Agent: curl/8.7.1
  > Accept: */*
  > X-Goproxy-Check-Redirect: skip
  >
  * Request completely sent off
  < HTTP/1.1 200 OK
  < Date: Wed, 08 Jan 2025 02:13:39 GMT
  < Content-Length: 0
  <
  * Connection #0 to host localhost left intact
  ```

- Caddyfile like rules

  ```yaml
  proxy.goaccess.rules: |
    - name: default
      do: |
        rewrite / /index.html
        serve /var/www/goaccess
    - name: ws
      on: |
        header Connection Upgrade
        header Upgrade websocket
      do: bypass # do nothing, pass to reverse proxy

  proxy.app.rules: |
    - name: default
      do: bypass # do nothing, pass to reverse proxy
    - name: block POST and PUT
      on: method POST | method PUT
      do: error 403 Forbidden
  ```

````

- config reload will now cause all servers to fully restart (i.e. proxy, api, prometheus, etc)
- multiline-string as list now treated as YAML list, which requires hyphen prefix `-`, i.e.
  ```yaml
  proxy.app.middlewares.request.hide_headers:
    - X-Header1
    - X-Header2
````
- autocert now supports hot-reload
- middleware compose now supports cross-referencing, e.g.
  ```yaml
  foo:
    - use: RedirectHTTP
  bar: # in the same file or different file
    - use: foo@file
  ```

- Fixes
  - bug: cert renewal failure no longer causes renew schdueler to stuck forever
  - bug: access log writes to closed file after config reload
