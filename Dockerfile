FROM alpine:latest

LABEL maintainer="yusing@6uo.me"

RUN apk add --no-cache tzdata
RUN mkdir /app
COPY bin/go-proxy /app/
COPY templates/ /app/templates
COPY schema/ /app/schema

RUN chmod +x /app/go-proxy
ENV DOCKER_HOST unix:///var/run/docker.sock
ENV GOPROXY_DEBUG 0
ENV GOPROXY_REDIRECT_HTTP 1

EXPOSE 80
EXPOSE 8080
EXPOSE 443
EXPOSE 8443

WORKDIR /app
CMD ["/app/go-proxy"]