# Getting started with `go-proxy` docker container

## Setup

1. Install `wget` if not already

2. Run setup script

   `bash <(wget -qO- https://6uo.me/go-proxy-setup-docker)`

   What it does:

   - Create required directories
   - Setup `config.yml` and `compose.yml`

3. Verify folder structure

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
