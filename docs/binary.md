# Getting started with `go-proxy` (binary)

## Setup

1. Install `bash`, `make` and `wget` if not already

2. Run setup script

   To specitfy a version _(optional)_

   ```shell
   export VERSION=latest # will be resolved into real version number
   export VERSION=<version>
   ```

   If you don't need web config editor

   ```shell
   export SETUP_CODEMIRROR=0
   ```

   Setup

   ```shell
   wget -qO- https://github.com/yusing/go-proxy/raw/main/setup-docker.sh | sudo bash
   ```

   What it does:

   - Download source file and binary into /opt/go-proxy/$VERSION
   - Setup `config.yml` and `providers.yml`
   - Setup `template/codemirror` which is a dependency for web config editor
   - Create a systemd service (if available) in `/etc/systemd/system/go-proxy.service`
   - Enable and start `go-proxy` service

3. Start editing config files in `http://<ip>:8080`

4. Check logs / status with `systemctl status go-proxy`

## Setup (alternative method)

1. Download the latest release and extract somewhere

2. Run `make setup` and _(optional) `make setup-codemirror`_

3. Enable HTTPS _(optional)_

   - To use autocert feature

     complete `autocert` in `config/config.yml`

   - To use existing certificate

     Prepare your wildcard (`*.y.z`) SSL cert in `certs/`

     - cert / chain / fullchain: `certs/cert.crt`
     - private key: `certs/priv.key`

4. Run the binary `bin/go-proxy`
