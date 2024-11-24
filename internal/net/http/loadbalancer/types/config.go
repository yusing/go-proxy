package types

type Config struct {
	Link    string         `json:"link" yaml:"link"`
	Mode    Mode           `json:"mode" yaml:"mode"`
	Weight  Weight         `json:"weight" yaml:"weight"`
	Options map[string]any `json:"options,omitempty" yaml:"options,omitempty"`
}
