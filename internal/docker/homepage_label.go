package docker

type (
	HomePageConfig   struct{ m map[string]HomePageCategory }
	HomePageCategory []HomePageItem

	HomePageItem struct {
		Name         string
		Icon         string
		Category     string
		Description  string
		WidgetConfig map[string]any
	}
)

func NewHomePageConfig() *HomePageConfig {
	return &HomePageConfig{m: make(map[string]HomePageCategory)}
}

func NewHomePageItem() *HomePageItem {
	return &HomePageItem{}
}

func (c *HomePageConfig) Clear() {
	c.m = make(map[string]HomePageCategory)
}

func (c *HomePageConfig) Add(item HomePageItem) {
	c.m[item.Category] = HomePageCategory{item}
}

const NSHomePage = "homepage"
