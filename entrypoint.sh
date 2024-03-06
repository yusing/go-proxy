#!/bin/bash
if [ "$1" == "restart" ]; then
    echo "restarting"
    killall go-proxy
fi
if [ -z "$VERBOSITY" ]; then
    VERBOSITY=1
fi
echo "starting with verbosity $VERBOSITY" > log/go-proxy.log
if [ "$DEBUG" == "1" ]; then
    /app/go-proxy -v=$VERBOSITY -log_dir=log --stderrthreshold=0 2>> log/go-proxy.log &
    tail -f /dev/null
else
    /app/go-proxy -v=$VERBOSITY -log_dir=log --stderrthreshold=0
fi