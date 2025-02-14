package v1

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	_ "embed"

	"github.com/yusing/go-proxy/agent/pkg/agent"
	"github.com/yusing/go-proxy/agent/pkg/certs"
	U "github.com/yusing/go-proxy/internal/api/v1/utils"
	config "github.com/yusing/go-proxy/internal/config/types"
	"github.com/yusing/go-proxy/internal/utils/strutils"
)

func NewAgent(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	name := q.Get("name")
	if name == "" {
		U.RespondError(w, U.ErrMissingKey("name"))
		return
	}
	host := q.Get("host")
	if host == "" {
		U.RespondError(w, U.ErrMissingKey("host"))
		return
	}
	portStr := q.Get("port")
	if portStr == "" {
		U.RespondError(w, U.ErrMissingKey("port"))
		return
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		U.RespondError(w, U.ErrInvalidKey("port"))
		return
	}
	hostport := fmt.Sprintf("%s:%d", host, port)
	if _, ok := config.GetInstance().GetAgent(hostport); ok {
		U.RespondError(w, U.ErrAlreadyExists("agent", hostport), http.StatusConflict)
		return
	}
	t := q.Get("type")
	switch t {
	case "docker":
		break
	case "system":
		U.RespondError(w, U.Errorf("system agent is not supported yet"), http.StatusNotImplemented)
		return
	case "":
		U.RespondError(w, U.ErrMissingKey("type"))
		return
	default:
		U.RespondError(w, U.ErrInvalidKey("type"))
		return
	}

	nightly := strutils.ParseBool(q.Get("nightly"))
	var image string
	if nightly {
		image = agent.DockerImageNightly
	} else {
		image = agent.DockerImageProduction
	}

	ca, srv, client, err := agent.NewAgent()
	if err != nil {
		U.HandleErr(w, r, err)
		return
	}

	cfg := agent.AgentComposeConfig{
		Image:   image,
		Name:    name,
		Port:    port,
		CACert:  ca.String(),
		SSLCert: srv.String(),
	}

	template, err := cfg.Generate()
	if err != nil {
		U.HandleErr(w, r, err)
		return
	}

	U.RespondJSON(w, r, map[string]any{
		"compose": template,
		"ca":      ca,
		"client":  client,
	})
}

func AddAgent(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	clientPEMData, err := io.ReadAll(r.Body)
	if err != nil {
		U.HandleErr(w, r, err)
		return
	}

	var data struct {
		Host   string        `json:"host"`
		CA     agent.PEMPair `json:"ca"`
		Client agent.PEMPair `json:"client"`
	}

	if err := json.Unmarshal(clientPEMData, &data); err != nil {
		U.RespondError(w, err, http.StatusBadRequest)
		return
	}

	nRoutesAdded, err := config.GetInstance().AddAgent(data.Host, data.CA, data.Client)
	if err != nil {
		U.RespondError(w, err)
		return
	}

	zip, err := certs.ZipCert(data.CA.Cert, data.Client.Cert, data.Client.Key)
	if err != nil {
		U.HandleErr(w, r, err)
		return
	}

	if err := os.WriteFile(certs.AgentCertsFilename(data.Host), zip, 0600); err != nil {
		U.HandleErr(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Added %d routes", nRoutesAdded)))
}
