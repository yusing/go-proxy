package homepage

type (
	HomePageConfig   map[string]HomePageCategory
	HomePageCategory []*HomePageItem

	HomePageItem struct {
		Name         string         `yaml:"name" json:"name"`
		Icon         string         `yaml:"icon" json:"icon,omitempty"`
		URL          string         `yaml:"url" json:"url,omitempty"` // URL or unicodes
		Category     string         `yaml:"category" json:"category"`
		SourceType   string         `yaml:"source_type" json:"source_type,omitempty"`
		Description  string         `yaml:"description" json:"description,omitempty"`
		WidgetConfig map[string]any `yaml:",flow" json:"widget_config,omitempty"`
	}
)

func NewHomePageConfig() HomePageConfig {
	return HomePageConfig(make(map[string]HomePageCategory))
}

func (c *HomePageConfig) Clear() {
	*c = make(HomePageConfig)
}

func (c HomePageConfig) Add(item *HomePageItem) {
	if c[item.Category] == nil {
		c[item.Category] = make(HomePageCategory, 0)
	}
	c[item.Category] = append(c[item.Category], item)
}
