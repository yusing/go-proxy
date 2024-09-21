## Docker Socket Proxy

For docker client on other machine, set this up, then add `name: tcp://<machine_ip>:2375` to `config.yml` under `docker` section

```yml
# compose.yml on remote machine (e.g. server1)
services:
  docker-proxy:
    container_name: docker-proxy
    image: ghcr.io/linuxserver/socket-proxy
    environment:
      - ALLOW_START=1 #optional
      - ALLOW_STOP=1 #optional
      - ALLOW_RESTARTS=0 #optional
      - AUTH=0 #optional
      - BUILD=0 #optional
      - COMMIT=0 #optional
      - CONFIGS=0 #optional
      - CONTAINERS=1 #optional
      - DISABLE_IPV6=1 #optional
      - DISTRIBUTION=0 #optional
      - EVENTS=1 #optional
      - EXEC=0 #optional
      - IMAGES=0 #optional
      - INFO=0 #optional
      - NETWORKS=0 #optional
      - NODES=0 #optional
      - PING=1 #optional
      - POST=1 #optional
      - PLUGINS=0 #optional
      - SECRETS=0 #optional
      - SERVICES=0 #optional
      - SESSION=0 #optional
      - SWARM=0 #optional
      - SYSTEM=0 #optional
      - TASKS=0 #optional
      - VERSION=1 #optional
      - VOLUMES=0 #optional
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    restart: always
    tmpfs:
      - /run
    ports:
      - 2375:2375
```

```yml
# config.yml on go-proxy machine
autocert:
    ... # your config

providers:
    include:
        ...
    docker:
        ...
        server1: tcp://<machine_ip>:2375
```
