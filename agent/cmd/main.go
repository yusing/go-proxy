package main

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net"
	"os"

	"github.com/rs/zerolog"
	"github.com/yusing/go-proxy/agent/pkg/certs"
	"github.com/yusing/go-proxy/agent/pkg/env"
	"github.com/yusing/go-proxy/agent/pkg/server"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/logging"
	"github.com/yusing/go-proxy/internal/logging/memlogger"
	"github.com/yusing/go-proxy/internal/task"
	"github.com/yusing/go-proxy/internal/utils"
	"github.com/yusing/go-proxy/pkg"
	"gopkg.in/yaml.v3"
)

func init() {
	logging.InitLogger(zerolog.MultiLevelWriter(os.Stderr, memlogger.GetMemLogger()))
}

func printNewClientHelp(ca *tls.Certificate) {
	crt, key, err := certs.NewClientCert(ca)
	if err != nil {
		E.LogFatal("init SSL error", err)
	}
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca.Certificate[0]})
	ip := machineIP()
	host := fmt.Sprintf("%s:%d", ip, env.AgentPort)
	cfgYAML, _ := yaml.Marshal(map[string]any{
		"providers": map[string]any{
			"agents": host,
		},
	})

	certsData, err := certs.ZipCert(caPEM, crt, key)
	if err != nil {
		E.LogFatal("marshal certs error", err)
	}

	fmt.Printf("Add this host (%s) to main server config like below:\n", host)
	fmt.Println(string(cfgYAML))
	fmt.Printf("On main server, run:\ngodoxy new-agent '%s' '%s'\n", host, base64.StdEncoding.EncodeToString(certsData))
}

func machineIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "<machine-ip>"
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "<machine-ip>"
}

func main() {
	args := pkg.GetArgs(agentCommandValidator{})

	ca, srv, isNew, err := certs.InitCerts()
	if err != nil {
		E.LogFatal("init CA error", err)
	}

	if args.Command == CommandNewClient {
		printNewClientHelp(ca)
		return
	}

	logging.Info().Msgf("GoDoxy Agent version %s", pkg.GetVersion())
	logging.Info().Msgf("Agent name: %s", env.AgentName)

	if isNew {
		logging.Info().Msg("Initialization complete.")
		logging.Info().Msg("New client cert created")
		printNewClientHelp(ca)
		logging.Info().Msg("Exiting... Clear the screen and start agent again")
		logging.Info().Msg("To create more client certs, run `godoxy-agent new-client`")
		return
	}

	server.StartAgentServer(task.RootTask("agent", false), server.Options{
		CACert:     ca,
		ServerCert: srv,
		Port:       env.AgentPort,
	})

	utils.WaitExit(3)
}
