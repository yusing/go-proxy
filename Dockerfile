FROM alpine:latest

MAINTAINER yusing@6uo.me

COPY bin/go-proxy /usr/bin

RUN chmod +rx /usr/bin/go-proxy
ENV DOCKER_HOST unix:///var/run/docker.sock

EXPOSE 80
EXPOSE 443

CMD ["go-proxy"]
