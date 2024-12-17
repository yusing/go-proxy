package types

type (
	AutoCertConfig struct {
		Email       string              `json:"email,omitempty" validate:"email"`
		Domains     []string            `json:"domains,omitempty"`
		CertPath    string              `json:"cert_path,omitempty" validate:"omitempty,filepath"`
		KeyPath     string              `json:"key_path,omitempty" validate:"omitempty,filepath"`
		ACMEKeyPath string              `json:"acme_key_path,omitempty" validate:"omitempty,filepath"`
		Provider    string              `json:"provider,omitempty"`
		Options     AutocertProviderOpt `json:"options,omitempty"`
	}
	AutocertProviderOpt map[string]any
)
