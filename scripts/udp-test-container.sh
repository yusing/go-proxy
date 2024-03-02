#!/bin/sh
docker run -it --tty --rm \
    -p 9999:9999/udp \
    --label proxy.test-udp.scheme=udp \
    --label proxy.test-udp.port=20003:9999 \
    --network data_default \
    --name test-udp \
    debian:stable-slim \
    /bin/bash -c \
    "apt update && apt install -y netcat-openbsd && echo 'nc -u -l 9999' >> ~/.bashrc && bash"
