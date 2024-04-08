#!/bin/bash
set -e
REPO_URL=https://github.com/yusing/go-proxy
BIN_URL="${REPO_URL}/releases/download/${VERSION}/go-proxy"
SRC_URL="${REPO_URL}/archive/refs/tags/${VERSION}.tar.gz"
APP_ROOT="/opt/go-proxy/${VERSION}"
LOG_FILE="/tmp/go-proxy-setup.log"

if [ -z "$VERSION" ] || [ "$VERSION" = "latest" ]; then
    VERSION_URL="${REPO_URL}/raw/main/version.txt"
    VERSION=$(wget -qO- "$VERSION_URL")
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
    wget -c "${SRC_URL}" -O go-proxy.tar.gz &> $LOG_FILE
    if [ $? -gt 0 ]; then
        echo "Source download failed, check your internet connection and version number"
        exit 1
    fi
    echo "Done"
    echo "Extracting go-proxy source ${VERSION}"
    tar xzf go-proxy.tar.gz &> $LOG_FILE
    if [ $? -gt 0 ]; then
        echo "failed to untar go-proxy.tar.gz"
        exit 1
    fi
    rm go-proxy.tar.gz
    mkdir -p "$(dirname "${APP_ROOT}")"
    mv "go-proxy-${VERSION}" "$APP_ROOT"
    cd "$APP_ROOT"
    echo "Done"
}
dl_binary() {
    mkdir -p bin
    echo "Downloading go-proxy binary ${VERSION}"
    wget -c "${BIN_URL}" -O bin/go-proxy &> $LOG_FILE
    if [ $? -gt 0 ]; then
        echo "Binary download failed, check your internet connection and version number"
        exit 1
    fi
    chmod +x bin/go-proxy
    echo "Done"
}
setup() {
    make setup &> $LOG_FILE
    if [ $? -gt 0 ]; then
        echo "make setup failed"
        exit 1
    fi
    # SETUP_CODEMIRROR = 1
    if [ "$SETUP_CODEMIRROR" != "0" ]; then
        make setup-codemirror &> $LOG_FILE || echo "make setup-codemirror failed, ignored"
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
systemctl_failed() {
    echo "Failed to enable and start go-proxy"
    systemctl status go-proxy
    exit 1
}
echo "Setting up systemd service"
cat <<EOF > /etc/systemd/system/go-proxy.service
[Unit]
Description=go-proxy reverse proxy
After=network-online.target
Wants=network-online.target systemd-networkd-wait-online.service
[Service]
Type=simple
ExecStart=${APP_ROOT}/bin/go-proxy
WorkingDirectory=${APP_ROOT}
Environment="GOPROXY_IS_SYSTEMD=1"
Restart=on-failure
RestartSec=1s
KillMode=process
KillSignal=SIGINT
TimeoutStartSec=5s
TimeoutStopSec=5s
[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload &>$LOG_FILE || systemctl_failed
systemctl enable --now go-proxy &>$LOG_FILE || systemctl_failed
echo "Done"
echo "Setup complete"