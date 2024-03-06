#!/bin/bash
if [ "$1" == "restart" ]; then
    killall go-proxy
fi
if [ "$DEBUG" == "1" ]; then
    /app/go-proxy -v=$VERBOSITY -log_dir=log --stderrthreshold=0 &
    if [ "$1" != "restart" ]; then
        tail -f /dev/null
    fi
else
    /app/go-proxy -v=$VERBOSITY -log_dir=log --stderrthreshold=0 &
fi