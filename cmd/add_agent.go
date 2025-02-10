package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/yusing/go-proxy/agent/pkg/certs"
	"github.com/yusing/go-proxy/internal/logging"
)

func AddAgent(args []string) {
	if len(args) != 1 {
		logging.Fatal().Msgf("invalid arguments: %v, expect host", args)
	}
	host := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "http://"+host, nil)
	if err != nil {
		logging.Fatal().Err(err).Msg("failed to create request")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logging.Fatal().Err(err).Msg("failed to send request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg, err := io.ReadAll(resp.Body)
		if err != nil {
			msg = []byte("unknown error")
		}
		logging.Fatal().Int("status", resp.StatusCode).Str("msg", string(msg)).Msg("failed to add agent")
	}

	zip, err := io.ReadAll(resp.Body)
	if err != nil {
		logging.Fatal().Err(err).Msg("failed to read response body")
	}

	f, err := os.OpenFile(certs.AgentCertsFilename(host), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		logging.Fatal().Err(err).Msg("failed to create client certs file")
	}
	defer f.Close()

	if _, err := f.Write(zip); err != nil {
		logging.Fatal().Err(err).Msg("failed to save client certs")
	}

	logging.Info().Msgf("agent %s added, certs saved to %s", host, certs.AgentCertsFilename(host))

	req, err = http.NewRequestWithContext(ctx, "GET", "http://"+host+"/done", nil)
	if err != nil {
		logging.Fatal().Err(err).Msg("failed to create done request")
	}

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		logging.Fatal().Err(err).Msg("failed to send done request")
	}
	defer resp.Body.Close()
}
