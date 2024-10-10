package types

type (
	AutoCertConfig struct {
		Email    string              `json:"email,omitempty" yaml:"email"`
		Domains  []string            `json:"domains,omitempty" yaml:",flow"`
		CertPath string              `json:"cert_path,omitempty" yaml:"cert_path"`
		KeyPath  string              `json:"key_path,omitempty" yaml:"key_path"`
		Provider string              `json:"provider,omitempty" yaml:"provider"`
		Options  AutocertProviderOpt `json:"options,omitempty" yaml:",flow"`
	}
	AutocertProviderOpt map[string]any
)
