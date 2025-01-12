package internal

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/yusing/go-proxy/internal/logging"
)

type GitHubContents struct { //! keep this, may reuse in future
	Type string `json:"type"`
	Path string `json:"path"`
	Name string `json:"name"`
	Sha  string `json:"sha"`
	Size int    `json:"size"`
}

type Icons map[string]map[string]struct{}

// no longer cache for `godoxy ls-icons`

const updateInterval = 1 * time.Hour

var (
	iconsCache   = make(Icons)
	iconsCahceMu sync.Mutex
	lastUpdate   time.Time
)

const walkxcodeIcons = "https://cdn.jsdelivr.net/gh/walkxcode/dashboard-icons@master/tree.json"

func ListAvailableIcons() (Icons, error) {
	iconsCahceMu.Lock()
	defer iconsCahceMu.Unlock()

	if time.Since(lastUpdate) < updateInterval {
		if len(iconsCache) > 0 {
			return iconsCache, nil
		}
	}

	icons, err := getIcons()
	if err != nil {
		return nil, err
	}

	iconsCache = icons
	lastUpdate = time.Now()
	return icons, nil
}

func HasIcon(name string, filetype string) bool {
	icons, err := ListAvailableIcons()
	if err != nil {
		logging.Error().Err(err).Msg("failed to list icons")
		return false
	}
	if _, ok := icons[filetype]; !ok {
		return false
	}
	_, ok := icons[filetype][name+"."+filetype]
	return ok
}

/*
format:

	{
		"png": [
			"*.png",
		],
		"svg": [
			"*.svg",
		]
	}
*/
func getIcons() (Icons, error) {
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
	icons := make(Icons, len(data))
	for fileType, files := range data {
		icons[fileType] = make(map[string]struct{}, len(files))
		for _, icon := range files {
			icons[fileType][icon] = struct{}{}
		}
	}
	return icons, nil
}
