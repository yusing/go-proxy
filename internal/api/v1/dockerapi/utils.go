package dockerapi

import (
	"encoding/json"
	"net/http"

	config "github.com/yusing/go-proxy/internal/config/types"
	"github.com/yusing/go-proxy/internal/docker"
	"github.com/yusing/go-proxy/internal/gperr"
)

// getDockerClients returns a map of docker clients for the current config.
//
// Returns a map of docker clients by server name and an error if any.
//
// Even if there are errors, the map of docker clients might not be empty.
func getDockerClients() (map[string]*docker.SharedClient, gperr.Error) {
	cfg := config.GetInstance()

	dockerHosts := cfg.Value().Providers.Docker
	dockerClients := make(map[string]*docker.SharedClient)

	connErrs := gperr.NewBuilder("failed to connect to docker")

	for name, host := range dockerHosts {
		dockerClient, err := docker.ConnectClient(host)
		if err != nil {
			connErrs.Add(err)
			continue
		}
		dockerClients[name] = dockerClient
	}

	for _, agent := range cfg.ListAgents() {
		dockerClient, err := docker.ConnectClient(agent.FakeDockerHost())
		if err != nil {
			connErrs.Add(err)
			continue
		}
		dockerClients[agent.Name()] = dockerClient
	}

	return dockerClients, connErrs.Error()
}

// getDockerClientsWithErrHandling returns a map of docker clients for the current config.
//
// Returns a map of docker clients by server name and a boolean indicating if http handler should stop/
func getDockerClientsWithErrHandling(w http.ResponseWriter) (map[string]*docker.SharedClient, bool) {
	dockerClients, err := getDockerClients()
	if err != nil {
		gperr.LogError("failed to get docker clients", err)
		if len(dockerClients) == 0 {
			http.Error(w, "no docker hosts connected successfully", http.StatusInternalServerError)
			return nil, false
		}
	}
	return dockerClients, true
}

func getDockerClient(w http.ResponseWriter, server string) (*docker.SharedClient, bool, error) {
	cfg := config.GetInstance()
	var host string
	for name, h := range cfg.Value().Providers.Docker {
		if name == server {
			host = h
			break
		}
	}
	for _, agent := range cfg.ListAgents() {
		if agent.Name() == server {
			host = agent.FakeDockerHost()
			break
		}
	}
	if host == "" {
		return nil, false, nil
	}
	dockerClient, err := docker.ConnectClient(host)
	if err != nil {
		return nil, false, err
	}
	return dockerClient, true, nil
}

// closeAllClients closes all docker clients after a delay.
//
// This is used to ensure that all docker clients are closed after the http handler returns.
func closeAllClients(dockerClients map[string]*docker.SharedClient) {
	for _, dockerClient := range dockerClients {
		dockerClient.Close()
	}
}

func handleResult[T any](w http.ResponseWriter, errs error, result []T) {
	if errs != nil {
		gperr.LogError("docker errors", errs)
		if len(result) == 0 {
			http.Error(w, "docker errors", http.StatusInternalServerError)
			return
		}
	}
	json.NewEncoder(w).Encode(result)
}
