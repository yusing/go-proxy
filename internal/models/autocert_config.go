package model

type (
	AutoCertConfig struct {
		Email    string              `json:"email"`
		Domains  []string            `yaml:",flow" json:"domains"`
		CertPath string              `yaml:"cert_path" json:"cert_path"`
		KeyPath  string              `yaml:"key_path" json:"key_path"`
		Provider string              `json:"provider"`
		Options  AutocertProviderOpt `yaml:",flow" json:"options"`
	}
	AutocertProviderOpt map[string]any
)
