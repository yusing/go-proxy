package homepage

type (
	HomePageConfig   map[string]HomePageCategory
	HomePageCategory []*HomePageItem

	HomePageItem struct {
		Show         bool           `yaml:"show" json:"show"`
		Name         string         `yaml:"name" json:"name"`
		Icon         string         `yaml:"icon" json:"icon"`
		URL          string         `yaml:"url" json:"url"` // URL or unicodes
		Category     string         `yaml:"category" json:"category"`
		SourceType   string         `yaml:"source_type" json:"source_type"`
		Description  string         `yaml:"description" json:"description"`
		WidgetConfig map[string]any `yaml:",flow" json:"widget_config"`

		Initialized bool `yaml:"-" json:"-"`
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