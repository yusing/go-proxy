package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/yusing/go-proxy/internal/utils"
)

type GitHubContents struct { //! keep this, may reuse in future
	Type string `json:"type"`
	Path string `json:"path"`
	Name string `json:"name"`
	Sha  string `json:"sha"`
	Size int    `json:"size"`
}

const (
	iconsCachePath = "/tmp/icons_cache.json"
	updateInterval = 1 * time.Hour
)

func ListAvailableIcons() ([]string, error) {
	owner := "walkxcode"
	repo := "dashboard-icons"
	ref := "main"

	var lastUpdate time.Time

	icons := make([]string, 0)
	info, err := os.Stat(iconsCachePath)
	if err == nil {
		lastUpdate = info.ModTime().Local()
	}
	if time.Since(lastUpdate) < updateInterval {
		err := utils.LoadJSON(iconsCachePath, &icons)
		if err == nil {
			return icons, nil
		}
	}

	contents, err := getRepoContents(http.DefaultClient, owner, repo, ref, "")
	if err != nil {
		return nil, err
	}
	for _, content := range contents {
		if content.Type != "dir" {
			icons = append(icons, content.Path)
		}
	}
	err = utils.SaveJSON(iconsCachePath, &icons, 0o644).Error()
	if err != nil {
		log.Print("error saving cache", err)
	}
	return icons, nil
}

func getRepoContents(client *http.Client, owner string, repo string, ref string, path string) ([]GitHubContents, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s", owner, repo, path, ref), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var contents []GitHubContents
	err = json.Unmarshal(body, &contents)
	if err != nil {
		return nil, err
	}

	filesAndDirs := make([]GitHubContents, 0)
	for _, content := range contents {
		if content.Type == "dir" {
			subContents, err := getRepoContents(client, owner, repo, ref, content.Path)
			if err != nil {
				return nil, err
			}
			filesAndDirs = append(filesAndDirs, subContents...)
		} else {
			filesAndDirs = append(filesAndDirs, content)
		}
	}

	return filesAndDirs, nil
}
