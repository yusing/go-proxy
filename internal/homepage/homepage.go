package homepage

import (
	"github.com/yusing/go-proxy/internal/utils"
)

type (
	//nolint:recvcheck
	Categories map[string]Category
	Category   []*Item

	ItemConfig struct {
		Show         bool           `json:"show"`
		Name         string         `json:"name"` // display name
		Icon         *IconURL       `json:"icon"`
		Category     string         `json:"category"`
		Description  string         `json:"description" aliases:"desc"`
		SortOrder    int            `json:"sort_order"`
		WidgetConfig map[string]any `json:"widget_config" aliases:"widget"`
		URL          string         `json:"url"` // alias + domain
	}

	Item struct {
		*ItemConfig

		Alias    string `json:"alias"` // proxy alias
		Provider string `json:"provider"`

		IsUnset bool `json:"-"`
	}
)

func init() {
	utils.RegisterDefaultValueFactory(func() *ItemConfig {
		return &ItemConfig{
			Show: true,
		}
	})
}

func NewItem(alias string) *Item {
	return &Item{
		ItemConfig: &ItemConfig{
			Show: true,
		},
		Alias:   alias,
		IsUnset: true,
	}
}

func NewHomePageConfig() Categories {
	return Categories(make(map[string]Category))
}

func (item *Item) IsEmpty() bool {
	return item == nil || item.IsUnset || item.ItemConfig == nil
}

func (item *Item) ApplyOverride() *Item {
	return overrideConfigInstance.ApplyOverride(item)
}

func (c *Categories) Clear() {
	*c = make(Categories)
}

func (c Categories) Add(item *Item) {
	if c[item.Category] == nil {
		c[item.Category] = make(Category, 0)
	}
	c[item.Category] = append(c[item.Category], item)
}
