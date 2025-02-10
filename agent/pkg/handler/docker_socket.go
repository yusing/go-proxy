package handler

import (
	"bufio"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/docker"
	"github.com/yusing/go-proxy/internal/logging"
	godoxyIO "github.com/yusing/go-proxy/internal/utils"
)

func DockerSocketHandler() http.HandlerFunc {
	dockerClient, err := docker.ConnectClient(common.DockerHostFromEnv)
	if err != nil {
		logging.Fatal().Err(err).Msg("failed to connect to docker client")
	}
	dockerDialerCallback := dockerClient.Dialer()

	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := dockerDialerCallback(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer conn.Close()

		// Create a done channel to handle cancellation
		done := make(chan struct{})
		defer close(done)

		closed := false

		// Start a goroutine to monitor context cancellation
		go func() {
			select {
			case <-r.Context().Done():
				closed = true
				conn.Close() // Force close the connection when client disconnects
			case <-done:
			}
		}()

		if err := r.Write(conn); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resp, err := http.ReadResponse(bufio.NewReader(conn), r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		// Set any response headers before writing the status code
		for k, v := range resp.Header {
			w.Header()[k] = v
		}
		w.WriteHeader(resp.StatusCode)

		// For event streams, we need to flush the writer to ensure
		// events are sent immediately
		if f, ok := w.(http.Flusher); ok && strings.HasSuffix(r.URL.Path, "/events") {
			// Copy the body in chunks and flush after each write
			buf := make([]byte, 2048)
			for {
				n, err := resp.Body.Read(buf)
				if n > 0 {
					_, werr := w.Write(buf[:n])
					if werr != nil {
						logging.Error().Err(werr).Msg("error writing docker event response")
						break
					}
					f.Flush()
				}
				if err != nil {
					if !closed && !errors.Is(err, io.EOF) {
						logging.Error().Err(err).Msg("error reading docker event response")
					}
					return
				}
			}
		} else {
			// For non-event streams, just copy the body
			_ = godoxyIO.NewPipe(r.Context(), resp.Body, NopWriteCloser{w}).Start()
		}
	}
}
