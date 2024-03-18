FROM alpine:latest

LABEL maintainer="yusing@6uo.me"

RUN apk add --no-cache bash tzdata
RUN mkdir /app
COPY bin/go-proxy entrypoint.sh /app/
COPY templates/ /app/templates
COPY config.default.yml /app/config.yml

RUN chmod +x /app/go-proxy /app/entrypoint.sh
ENV DOCKER_HOST unix:///var/run/docker.sock
ENV GOPROXY_DEBUG 0
ENV GOPROXY_REDIRECT_HTTP 1

EXPOSE 80
EXPOSE 443
EXPOSE 8443

WORKDIR /app
ENTRYPOINT /app/entrypoint.sh