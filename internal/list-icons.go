package internal

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/logging"
	"github.com/yusing/go-proxy/internal/utils"
)

type GitHubContents struct { //! keep this, may reuse in future
	Type string `json:"type"`
	Path string `json:"path"`
	Name string `json:"name"`
	Sha  string `json:"sha"`
	Size int    `json:"size"`
}

type (
	IconsMap map[string]map[string]struct{}
	Cache    struct {
		WalkxCode, Selfhst IconsMap
		DisplayNames       ReferenceDisplayNameMap
	}
	ReferenceDisplayNameMap map[string]string
)

func (icons *Cache) isEmpty() bool {
	return len(icons.WalkxCode) == 0 && len(icons.Selfhst) == 0
}

func (icons *Cache) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"walkxcode": icons.WalkxCode,
		"selfhst":   icons.Selfhst,
	})
}

const updateInterval = 1 * time.Hour

var (
	iconsCache   *Cache
	iconsCahceMu sync.Mutex
	lastUpdate   time.Time
)

const (
	walkxcodeIcons = "https://cdn.jsdelivr.net/gh/walkxcode/dashboard-icons@master/tree.json"
	selfhstIcons   = "https://cdn.selfh.st/directory/icons.json"
)

func InitIconListCache() {
	iconsCache = &Cache{
		WalkxCode:    make(IconsMap),
		Selfhst:      make(IconsMap),
		DisplayNames: make(ReferenceDisplayNameMap),
	}
	err := utils.LoadJSON(common.IconListCachePath, iconsCache)
	if err != nil && !os.IsNotExist(err) {
		logging.Fatal().Err(err).Msg("failed to load icon list cache config")
	} else if err == nil {
		if stats, err := os.Stat(common.IconListCachePath); err != nil {
			logging.Fatal().Err(err).Msg("failed to load icon list cache config")
		} else {
			lastUpdate = stats.ModTime()
		}
	}
}

func ListAvailableIcons() (*Cache, error) {
	iconsCahceMu.Lock()
	defer iconsCahceMu.Unlock()

	if time.Since(lastUpdate) < updateInterval {
		if !iconsCache.isEmpty() {
			return iconsCache, nil
		}
	}

	icons, err := fetchIconData()
	if err != nil {
		return nil, err
	}

	iconsCache = icons
	lastUpdate = time.Now()

	err = utils.SaveJSON(common.IconListCachePath, iconsCache, 0o644)
	if err != nil {
		logging.Warn().Err(err).Msg("failed to save icon list cache")
	}
	return icons, nil
}

func HasWalkxCodeIcon(name string, filetype string) bool {
	icons, err := ListAvailableIcons()
	if err != nil {
		logging.Error().Err(err).Msg("failed to list icons")
		return false
	}
	if _, ok := icons.WalkxCode[filetype]; !ok {
		return false
	}
	_, ok := icons.WalkxCode[filetype][name+"."+filetype]
	return ok
}

func HasSelfhstIcon(name string, filetype string) bool {
	icons, err := ListAvailableIcons()
	if err != nil {
		logging.Error().Err(err).Msg("failed to list icons")
		return false
	}
	if _, ok := icons.Selfhst[filetype]; !ok {
		return false
	}
	_, ok := icons.Selfhst[filetype][name+"."+filetype]
	return ok
}

func GetDisplayName(reference string) (string, bool) {
	icons, err := ListAvailableIcons()
	if err != nil {
		logging.Error().Err(err).Msg("failed to list icons")
		return "", false
	}
	displayName, ok := icons.DisplayNames[reference]
	return displayName, ok
}

func fetchIconData() (*Cache, error) {
	walkxCodeIcons, err := fetchWalkxCodeIcons()
	if err != nil {
		return nil, err
	}

	selfhstIcons, referenceToNames, err := fetchSelfhstIcons()
	if err != nil {
		return nil, err
	}

	return &Cache{
		WalkxCode:    walkxCodeIcons,
		Selfhst:      selfhstIcons,
		DisplayNames: referenceToNames,
	}, nil
}

/*
format:

	{
		"png": [
			"*.png",
		],
		"svg": [
			"*.svg",
		],
		"webp": [
			"*.webp",
		]
	}
*/
func fetchWalkxCodeIcons() (IconsMap, error) {
	req, err := http.NewRequest(http.MethodGet, walkxcodeIcons, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	data := make(map[string][]string)
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}
	icons := make(IconsMap, len(data))
	for fileType, files := range data {
		icons[fileType] = make(map[string]struct{}, len(files))
		for _, icon := range files {
			icons[fileType][icon] = struct{}{}
		}
	}
	return icons, nil
}

/*
format:

	{
			"Name": "2FAuth",
			"Reference": "2fauth",
			"SVG": "Yes",
			"PNG": "Yes",
			"WebP": "Yes",
			"Light": "Yes",
			"Category": "Self-Hosted",
			"CreatedAt": "2024-08-16 00:27:23+00:00"
	}
*/
func fetchSelfhstIcons() (IconsMap, ReferenceDisplayNameMap, error) {
	type SelfhStIcon struct {
		Name      string `json:"Name"`
		Reference string `json:"Reference"`
		SVG       string `json:"SVG"`
		PNG       string `json:"PNG"`
		WebP      string `json:"WebP"`
		// Light          string
		// Category       string
		// CreatedAt      string
	}

	req, err := http.NewRequest(http.MethodGet, selfhstIcons, nil)
	if err != nil {
		return nil, nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	data := make([]SelfhStIcon, 0, 2000)
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, nil, err
	}

	icons := make(IconsMap)
	icons["svg"] = make(map[string]struct{}, len(data))
	icons["png"] = make(map[string]struct{}, len(data))
	icons["webp"] = make(map[string]struct{}, len(data))

	referenceToNames := make(ReferenceDisplayNameMap, len(data))

	for _, item := range data {
		if item.SVG == "Yes" {
			icons["svg"][item.Reference+".svg"] = struct{}{}
		}
		if item.PNG == "Yes" {
			icons["png"][item.Reference+".png"] = struct{}{}
		}
		if item.WebP == "Yes" {
			icons["webp"][item.Reference+".webp"] = struct{}{}
		}
		referenceToNames[item.Reference] = item.Name
	}

	return icons, referenceToNames, nil
}
