FROM debian:stable-slim

RUN apt update && \
    apt install -y netcat-openbsd && \
    rm -rf /var/lib/apt/lists/*

RUN printf '#!/bin/bash\nclear; echo "Netcat UDP server started"; nc -u -l 9999; exit' >> /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
