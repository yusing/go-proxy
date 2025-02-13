## GoDoxy v0.10.0

### GoDoxy Agent

Maintain secure connection between main server and agent server by authenticating and encrypting connection with mTLS.

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
- Secure: no one can connect to it except GoDoxy main server because of mTLS, plus connection is encrypted
- Fetch info from agent server, e.g. CPU usage, Memory usage, container list, container logs, etc... (to be ready for beszel and dockge like features in WebUI)

#### How to setup

Prerequisites:

- GoDoxy main server must be running

1. Create a directory for agent server, cd into it
2. Copy `agent.compose.yml` into the directory
3. Modify `agent.compose.yml` to set `REGISTRATION_ALLOWED_HOSTS`
4. Run `docker-compose up -d` to start agent
5. Follow instructions on screen to run command on GoDoxy main server
6. Add config output to GoDoxy main server in `config.yml` under `providers.agents`
   ```yaml
   providers:
     agents:
       - 12.34.5.6:8889
   ```

### How does it work

Setup flow:

```mermaid
flowchart TD
    subgraph Agent Server
        A[Create a directory] -->
        B[Setup agent.compose.yml] -->
        C[Set REGISTRATION_ALLOWED_HOSTS] -->
        D[Run agent] -->
        E[Wait for main server to register]

        F[Respond to main server]
        G[Agent now run in agent mode]
    end
    subgraph Main Server
      E -->
      H[Run register command] -->
      I[Send registration request] --> F -->
      J[Store client certs] -->
      K[Send done request] --> G -->
      L[Add agent to config.yml]
    end
```

Run flow:

```mermaid
flowchart TD
    subgraph Agent HTTPS Server
        aa[Load CA and SSL certs] -->
        ab[Start HTTPS server] -->

        ac[Receive request] -->
        ad[Verify client cert] -->
        ae[Handle request] --> ac
    end
    subgraph Main Server
        ma[Load client certs] -->
        mb[Query agent version] --> ac
        mb --> mc[Check if agent version matches] -->
        md[Query agent info] --> ac
        md --> ae --> me[Store agent info]
    end
```
