## Docker Socket Proxy

For docker client on other machine, set this up, then add `name: tcp://<machine_ip>:2375` to `config.yml` under `docker` section

```yml
# compose.yml on remote machine (e.g. server1)
docker-proxy:
  container_name: docker-proxy
  image: tecnativa/docker-socket-proxy
  privileged: true
  environment:
    - ALLOW_START=1
    - ALLOW_STOP=1
    - ALLOW_RESTARTS=1
    - CONTAINERS=1
    - EVENTS=1
    - PING=1
    - POST=1
    - VERSION=1
  volumes:
    - /var/run/docker.sock:/var/run/docker.sock
  restart: always
  ports:
    - 2375:2375
    # or more secure
    - <machine_ip>:2375:2375
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
