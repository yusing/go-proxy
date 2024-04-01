# Getting started with `go-proxy` docker container

## Setup

1. Install `wget` if not already

2. Run setup script

   `bash <(wget -qO- https://6uo.me/go-proxy-setup-docker)`

   What it does:

   - Create required directories
   - Setup `config.yml` and `compose.yml`

3. Verify folder structure and then `cd go-proxy`

   ```plain
   go-proxy
   ├── certs
   ├── compose.yml
   └── config
       ├── config.yml
       └── providers.yml
   ```

4. Enable HTTPs _(optional)_

   - To use autocert feature

     - completing `autocert` section in `config/config.yml`
     - mount `certs/` to `/app/certs` to store obtained certs

   - To use existing certificate

     mount your wildcard (`*.y.z`) SSL cert

     - cert / chain / fullchain -> `/app/certs/cert.crt`
     - private key -> `/app/certs/priv.key`

5. Modify `compose.yml` fit your needs

   Add networks to make sure it is in the same network with other containers, or make sure `proxy.<alias>.host` is reachable

6. Run `docker compose up -d` to start the container

7. Start editing config files in `http://<ip>:8080`

## Troubleshooting

- Firewall issues

  If you are using `ufw` with vpn that drop all inbound traffic except vpn, run below:

  `sudo ufw allow from 172.16.0.0/16 to 100.64.0.0/10`

  Explaination:

  Docker network is usually `172.16.0.0/16`

  Tailscale is used as an example, `100.64.0.0/10` will be the CIDR

  You can also list CIDRs of all docker bridge networks by:

  `docker network inspect $(docker network ls | awk '$3 == "bridge" { print $1}') | jq -r '.[] | .Name + " " + .IPAM.Config[0].Subnet' -`

## Docker compose example (bridge network)

```yaml
volumes:
  adg-work:
  adg-conf:
  mc-data:
  palworld:
  nginx:
services:
  adg:
    image: adguard/adguardhome
    restart: unless-stopped
    labels:
      - proxy.aliases=adg,adg-dns,adg-setup
      - proxy.adg.port=80
      - proxy.adg-setup.port=3000
      - proxy.adg-dns.scheme=udp
      - proxy.adg-dns.port=20000:dns
    volumes:
      - adg-work:/opt/adguardhome/work
      - adg-conf:/opt/adguardhome/conf
  mc:
    image: itzg/minecraft-server
    tty: true
    stdin_open: true
    container_name: mc
    restart: unless-stopped
    labels:
      - proxy.mc.scheme=tcp
      - proxy.mc.port=20001:25565
    environment:
      - EULA=TRUE
    volumes:
      - mc-data:/data
  palworld:
    image: thijsvanloef/palworld-server-docker:latest
    restart: unless-stopped
    container_name: pal
    stop_grace_period: 30s
    labels:
      - proxy.aliases=pal1,pal2
      - proxy.*.scheme=udp
      - proxy.pal1.port=20002:8211
      - proxy.pal2.port=20003:27015
    environment: ...
    volumes:
      - palworld:/palworld
  nginx:
    image: nginx
    container_name: nginx
    volumes:
      - nginx:/usr/share/nginx/html
  go-proxy:
    image: ghcr.io/yusing/go-proxy
    container_name: go-proxy
    restart: always
    ports:
      - 80:80 # http
      - 443:443 # optional, https
      - 8080:8080 # http panel
      - 8443:8443 # optional, https panel

      - 53:20000/udp # adguardhome
      - 25565:20001/tcp # minecraft
      - 8211:20002/udp # palworld
      - 27015:20003/udp # palworld
    volumes:
      - ./config:/app/config
      - /var/run/docker.sock:/var/run/docker.sock:ro
    labels:
      - proxy.aliases=gp
      - proxy.panel.port=8080
```

### Services URLs

- `gp.yourdomain.com`: go-proxy web panel
- `adg-setup.yourdomain.com`: adguard setup (first time setup)
- `adg.yourdomain.com`: adguard dashboard
- `nginx.yourdomain.com`: nginx
- `yourdomain.com:53`: adguard dns
- `yourdomain.com:25565`: minecraft server
- `yourdomain.com:8211`: palworld server

## Docker compose example (host network)

### Notice

When `go-proxy` is running in `host` network mode, you must:

- set `GOPROXY_HOST_NETWORK=1`
- map ports to host explicitly
- add an asterisk sign **(*)** before `port` number under `labels`

```yaml
volumes:
  adg-work:
  adg-conf:
  mc-data:
  palworld:
  nginx:
services:
  adg:
    image: adguard/adguardhome
    restart: unless-stopped
    ports: # map random ports to container ports
      - 80
      - 3000
      - 53/udp
      - 53/tcp
    labels:
      - proxy.aliases=adg,adg-dns,adg-setup
      # add an asterisk (*) before to find host port automatically
      - proxy.adg.port=*80
      - proxy.adg-setup.port=*3000
      - proxy.adg-dns.scheme=udp
      - proxy.adg-dns.port=*53
    volumes:
      - adg-work:/opt/adguardhome/work
      - adg-conf:/opt/adguardhome/conf
  mc:
    image: itzg/minecraft-server
    tty: true
    stdin_open: true
    container_name: mc
    restart: unless-stopped
    ports:
      - 25565
    labels:
      - proxy.mc.scheme=tcp
      - proxy.mc.port=*25565
    environment:
      - EULA=TRUE
    volumes:
      - mc-data:/data
  palworld:
    image: thijsvanloef/palworld-server-docker:latest
    restart: unless-stopped
    container_name: pal
    stop_grace_period: 30s
    ports:
      - 8211/udp
      - 27015/udp
    labels:
      - proxy.aliases=pal1,pal2
      - proxy.*.scheme=udp
      - proxy.pal1.port=*8211
      - proxy.pal2.port=*27015
    environment: ...
    volumes:
      - palworld:/palworld
  nginx:
    image: nginx
    container_name: nginx
    # for single port container, host port will be found automatically
    ports:
      - 80
    volumes:
      - nginx:/usr/share/nginx/html
  go-proxy:
    image: ghcr.io/yusing/go-proxy
    container_name: go-proxy
    restart: always
    network_mode: host # no port mapping needed for host network mode
    environment:
      - GOPROXY_HOST_NETWORK=1 # required for host network mode
    volumes:
      - ./config:/app/config
      - /var/run/docker.sock:/var/run/docker.sock:ro
    labels:
      - proxy.aliases=gp
      - proxy.panel.port=808
```

**Same services URLs as [`bridge`](#services-urls) example!**
