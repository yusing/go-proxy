#!/bin/sh
set -e
REPO_URL=https://github.com/yusing/go-proxy
BIN_URL="${REPO_URL}/releases/download/${VERSION}/go-proxy"
SRC_URL="${REPO_URL}/archive/refs/tags/${VERSION}.tar.gz"
APP_ROOT="/opt/go-proxy/${VERSION}"

if [ -z "$VERSION" ]; then
    echo "You must specify a version"
    exit 1
fi

if [ -d "$APP_ROOT" ]; then
    echo "$APP_ROOT already exists"
    exit 1
fi

# check if wget exists
if ! [ -x "$(command -v wget)" ]; then
    echo "wget is not installed"
    exit 1
fi

# check if make exists
if ! [ -x "$(command -v make)" ]; then
    echo "make is not installed"
    exit 1
fi

dl_source() {
    cd /tmp
    echo "Downloading go-proxy source ${VERSION}"
    wget -c "${SRC_URL}" -O go-proxy.tar.gz 2>&1
    echo "Done"
    if [ $? -gt 0 ]; then
        echo "Source download failed, check your internet connection and version number"
        exit 1
    fi
    echo "Extracting go-proxy source ${VERSION}"
    tar xzf go-proxy.tar.gz 2>&1
    if [ $? -gt 0 ]; then
        echo "failed to untar go-proxy.tar.gz"
        exit 1
    fi
    rm go-proxy.tar.gz
    mkdir -p $(dirname $APP_ROOT)
    mv "go-proxy-${VERSION}" $APP_ROOT
    cd $APP_ROOT
    echo "Done"
}
dl_binary() {
    mkdir -p bin
    echo "Downloading go-proxy binary ${VERSION}"
    wget -c "${BIN_URL}" -O bin/go-proxy 2>&1
    if [ $? -gt 0 ]; then
        echo "Binary download failed, check your internet connection and version number"
        exit 1
    fi
    chmod +x bin/go-proxy
    echo "Done"
}
setup() {
    make setup
    if [ $? -gt 0 ]; then
        echo "make setup failed"
        exit 1
    fi
    # SETUP_CODEMIRROR = 1
    if [ "$SETUP_CODEMIRROR" = "1" ]; then
        make setup-codemirror || echo "make setup-codemirror failed, ignored"
    fi
}

dl_source
dl_binary
setup

# setup systemd

# check if systemctl exists
if ! command -v systemctl is-system-running > /dev/null 2>&1; then
    echo "systemctl not found, skipping systemd setup"
    exit 0
fi
systemctl-failed() {
    echo "Failed to enable and start go-proxy"
    systemctl status go-proxy
    exit 1
}
mkdir -p /etc/systemd/system
cat <<EOF > /etc/systemd/system/go-proxy.service
[Unit]
Description=go-proxy reverse proxy
After=network.target
[Service]
Type=simple
ExecStart=${APP_ROOT}/bin/go-proxy
WorkingDirectory=${APP_ROOT}
[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload || systemctl-failed
systemctl enable --now go-proxy || systemctl-failed
echo "Setup complete"