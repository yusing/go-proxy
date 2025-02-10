## GoDoxy v0.10.0

### Agent Mode

listen only on Agent API server, authenticate with mTLS. Maintain secure connection between GoDoxy main and GoDoxy agent server

Main benefits:

- No more exposing docker socket: drops the need of `docker-socket-proxy`
- No more exposing app ports: fewer attack surface
  ```yaml
  services:
    app:
      ...
      # ports: # this part is not needed on agent server
      #  - 6789
  ```
- Secure: no one can connect to it except GoDoxy main server because of mTLS
- Fetch info from agent server, e.g. CPU usage, Memory usage, container list, container logs, etc... (to be ready for beszel and dockge like features in WebUI)

### How to setup

1. Agent server generates CA cert, SSL certificate and Client certificate on first run.
2. Follow the output on screen to run `godoxy new-agent <ip>:<port> ...` on GoDoxy main server to store generated certs
3. Add config output to GoDoxy main server in `config.yml` under `providers.agents`
   ```yaml
   providers:
     agents:
       - 12.34.5.6:8889
   ```

### How does it work

1. Main server and agent server negotiate mTLS
2. Agent server verify main server's client cert and check if server version matches agent version
3. Agent server now acts as a http proxy and docker socket proxy
