package agent

import (
	"bytes"
	"os"
	"path"
	"strconv"
	"text/template"

	_ "embed"

	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/logging"
	"github.com/yusing/go-proxy/internal/utils"
	"gopkg.in/yaml.v3"
)

//go:embed agent.compose.yml
var agentComposeYAML []byte
var agentComposeYAMLTemplate = template.Must(template.New("agent.compose.yml").Parse(string(agentComposeYAML)))

const (
	DockerImageProduction = "ghcr.io/yusing/godoxy-agent:latest"
	DockerImageNightly    = "yusing/godoxy-agent-nightly:latest"
)

type (
	AgentComposeConfig struct {
		Image   string
		Name    string
		Port    int
		CACert  string
		SSLCert string
	}
)

func (c *AgentComposeConfig) Generate() (string, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 1024))
	if err := agentComposeYAMLTemplate.Execute(buf, c); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func pemPairFromFile(path string) (*PEMPair, error) {
	cert, err := os.ReadFile(path + ".crt")
	if err != nil {
		return nil, err
	}
	key, err := os.ReadFile(path + ".key")
	if err != nil {
		return nil, err
	}
	return &PEMPair{
		Cert: cert,
		Key:  key,
	}, nil
}

func rmOldCerts(p string) error {
	files, err := utils.ListFiles(p, 0)
	if err != nil {
		return err
	}
	for _, file := range files {
		if err := os.Remove(path.Join(p, file)); err != nil {
			return err
		}
	}
	return nil
}

type dockerCompose struct {
	Services struct {
		GodoxyAgent struct {
			Environment struct {
				AGENT_NAME string `yaml:"GODOXY_AGENT_NAME"`
				AGENT_PORT string `yaml:"GODOXY_AGENT_PORT"`
			} `yaml:"environment"`
		} `yaml:"godoxy-agent"`
	} `yaml:"services"`
}

// TODO: remove this
func MigrateFromOld() error {
	oldCompose, err := os.ReadFile("/app/compose.yml")
	if err != nil {
		return err
	}
	var compose dockerCompose
	if err := yaml.Unmarshal(oldCompose, &compose); err != nil {
		return err
	}
	ca, err := pemPairFromFile("/app/certs/ca")
	if err != nil {
		return err
	}
	agentCert, err := pemPairFromFile("/app/certs/agent")
	if err != nil {
		return err
	}
	var composeConfig AgentComposeConfig
	composeConfig.Image = DockerImageNightly
	composeConfig.Name = compose.Services.GodoxyAgent.Environment.AGENT_NAME
	composeConfig.Port, err = strconv.Atoi(compose.Services.GodoxyAgent.Environment.AGENT_PORT) // ignore error, empty is fine
	if composeConfig.Port == 0 {
		composeConfig.Port = 8890
	}
	composeConfig.CACert = ca.String()
	composeConfig.SSLCert = agentCert.String()
	composeTemplate, err := composeConfig.Generate()
	if err != nil {
		return E.Wrap(err, "failed to generate new docker compose")
	}

	if err := os.WriteFile("/app/compose.yml", []byte(composeTemplate), 0600); err != nil {
		return E.Wrap(err, "failed to write new docker compose")
	}

	logging.Info().Msg("Migrated from old docker compose:")
	logging.Info().Msg(composeTemplate)
	return nil
}
