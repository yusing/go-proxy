#!/bin/sh
echo "Running as PUID: ${PUID}, PGID: ${PGID}"
echo "Creating user"
addgroup -S -g "${PGID}" godoxyg
adduser -S -D -H -s /bin/false -u "${PUID}" -g "${PGID}" godoxy

echo "Setting up permissions"
chown -R godoxy:godoxyg /app
setcap CAP_NET_BIND_SERVICE=+eip /app/godoxy

# fork docker socket if exists
if test -e /var/run/docker.sock; then
	echo "Proxying docker socket"
	socat -v "UNIX-LISTEN:${SOCKET_FORK}",fork UNIX-CONNECT:/var/run/docker.sock >/dev/null 2>&1 &
	# wait for socket to be ready
	while [ ! -S "${SOCKET_FORK}" ]; do
		sleep 0.1
	done
	chmod 660 "${SOCKET_FORK}"
	chown godoxy:godoxyg "${SOCKET_FORK}"
fi

echo "Done"

runuser -u godoxy -g godoxyg -- /app/godoxy
