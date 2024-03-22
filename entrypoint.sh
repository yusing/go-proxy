#!/bin/bash
if [ "$1" == "restart" ]; then
    echo "restarting"
    killall go-proxy
fi
if [ "$GOPROXY_DEBUG" == "1" ]; then
    /app/go-proxy 2> log/go-proxy.log &
    tail -f /dev/null
else
    /app/go-proxy
fi