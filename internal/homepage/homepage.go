package homepage

type (
	//nolint:recvcheck
	Config   map[string]Category
	Category []*Item

	Item struct {
		Show         bool           `json:"show"`
		Name         string         `json:"name"` // display name
		Icon         *IconURL       `json:"icon"`
		URL          string         `json:"url"` // alias + domain
		Category     string         `json:"category"`
		Description  string         `json:"description" aliases:"desc"`
		WidgetConfig map[string]any `json:"widget_config" aliases:"widget"`

		Alias      string `json:"alias"` // proxy alias
		SourceType string `json:"source_type"`
		AltURL     string `json:"alt_url"` // original proxy target
	}
)

func (item *Item) IsEmpty() bool {
	return item == nil || (item.Name == "" &&
		item.Icon == nil &&
		item.URL == "" &&
		item.Category == "" &&
		item.Description == "" &&
		len(item.WidgetConfig) == 0)
}

func NewHomePageConfig() Config {
	return Config(make(map[string]Category))
}

func (c *Config) Clear() {
	*c = make(Config)
}

func (c Config) Add(item *Item) {
	if c[item.Category] == nil {
		c[item.Category] = make(Category, 0)
	}
	c[item.Category] = append(c[item.Category], item)
}
