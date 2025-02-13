package handler

import (
	"net/http"
	"net/url"

	"github.com/docker/docker/client"
	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/docker"
	"github.com/yusing/go-proxy/internal/logging"
	"github.com/yusing/go-proxy/internal/net/http/reverseproxy"
	"github.com/yusing/go-proxy/internal/net/types"
)

func DockerSocketHandler() http.HandlerFunc {
	dockerClient, err := docker.ConnectClient(common.DockerHostFromEnv)
	if err != nil {
		logging.Fatal().Err(err).Msg("failed to connect to docker client")
	}
	rp := reverseproxy.NewReverseProxy("docker", types.NewURL(&url.URL{
		Scheme: "http",
		Host:   client.DummyHost,
	}), dockerClient.HTTPClient().Transport)

	return rp.ServeHTTP
}
