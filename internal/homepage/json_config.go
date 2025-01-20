package homepage

import (
	"errors"
	"os"
	"sync"

	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/logging"
	"github.com/yusing/go-proxy/internal/utils"
)

type JSONConfig struct {
	DisplayNameOverride     map[string]string   `json:"display_name_override"`
	DisplayCategoryOverride map[string]string   `json:"display_category_override"`
	DisplayOrder            map[string]int      `json:"display_order"` // TODO: implement this
	CategoryNameOverride    map[string]string   `json:"category_name_override"`
	CategoryOrder           map[string]int      `json:"category_order"` // TODO: implement this
	IconOverride            map[string]*IconURL `json:"icon_override"`
	ShowItemOverride        map[string]bool     `json:"show_item_override"`
	mu                      sync.RWMutex
}

var jsonConfigInstance *JSONConfig

func InitOverridesConfig() {
	jsonConfigInstance = &JSONConfig{
		DisplayNameOverride:     make(map[string]string),
		DisplayCategoryOverride: make(map[string]string),
		DisplayOrder:            make(map[string]int),
		CategoryNameOverride:    make(map[string]string),
		CategoryOrder:           make(map[string]int),
		IconOverride:            make(map[string]*IconURL),
		ShowItemOverride:        make(map[string]bool),
	}
	err := utils.LoadJSON(common.HomepageJSONConfigPath, jsonConfigInstance)
	if err != nil && !os.IsNotExist(err) {
		logging.Fatal().Err(err).Msg("failed to load homepage overrides config")
	}
}

func GetJSONConfig() *JSONConfig {
	return jsonConfigInstance
}

func (c *JSONConfig) save() error {
	if common.IsTest {
		return nil
	}
	return utils.SaveJSON(common.HomepageJSONConfigPath, c, 0o644)
}

func (c *JSONConfig) GetDisplayName(item *Item) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if override, ok := c.DisplayNameOverride[item.Alias]; ok {
		return override
	}
	return item.Name
}

func (c *JSONConfig) SetDisplayNameOverride(key, value string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.DisplayNameOverride[key] = value
	return c.save()
}

func (c *JSONConfig) GetCategory(item *Item) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	category := item.Category
	if override, ok := c.DisplayCategoryOverride[item.Alias]; ok {
		category = override
	}
	if override, ok := c.CategoryNameOverride[category]; ok {
		return override
	}
	return category
}

func (c *JSONConfig) SetDisplayCategoryOverride(key, value string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.DisplayCategoryOverride[key] = value
	return c.save()
}

func (c *JSONConfig) SetDisplayOrder(key string, value int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.DisplayOrder[key] = value
	return c.save()
}

func (c *JSONConfig) SetCategoryNameOverride(key, value string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.CategoryNameOverride[key] = value
	return c.save()
}

func (c *JSONConfig) SetCategoryOrder(key string, value int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.CategoryOrder[key] = value
	return c.save()
}

func (c *JSONConfig) GetDisplayIcon(item *Item) *IconURL {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if override, ok := c.IconOverride[item.Alias]; ok {
		return override
	}
	return item.Icon
}

func (c *JSONConfig) SetIconOverride(key, value string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var url IconURL
	if err := url.Parse(value); err != nil {
		return err
	}
	if !url.HasIcon() {
		return errors.New("no such icon")
	}
	c.IconOverride[key] = &url
	return c.save()
}

func (c *JSONConfig) GetShowItem(item *Item) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if override, ok := c.ShowItemOverride[item.Alias]; ok {
		return override
	}
	return true
}

func (c *JSONConfig) SetShowItemOverride(key string, value bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ShowItemOverride[key] = value
	return c.save()
}
