package internal

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"

	. "github.com/yusing/go-proxy/internal/common"
)

var branch = GetEnv("GOPROXY_BRANCH", "v0.5")
var baseUrl = fmt.Sprintf("https://github.com/yusing/go-proxy/raw/%s", branch)
var requiredConfigs = []Config{
	{ConfigBasePath, true, false, ""},
	{ComposeFileName, false, true, ComposeExampleFileName},
	{path.Join(ConfigBasePath, ConfigFileName), false, true, ConfigExampleFileName},
}

type Config struct {
	Pathname         string
	IsDir            bool
	NeedDownload     bool
	DownloadFileName string
}

func Setup() {
	log.Println("setting up go-proxy")
	log.Println("branch:", branch)

	os.Chdir("/setup")

	for _, config := range requiredConfigs {
		config.setup()
	}

	log.Println("done")
}

func (c *Config) setup() {
	if c.IsDir {
		mkdir(c.Pathname)
		return
	}
	if !c.NeedDownload {
		touch(c.Pathname)
		return
	}

	fetch(c.DownloadFileName, c.Pathname)
}

func hasFileOrDir(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func mkdir(pathname string) {
	_, err := os.Stat(pathname)
	if err != nil && os.IsNotExist(err) {
		log.Printf("creating directory %q\n", pathname)
		err := os.MkdirAll(pathname, 0o755)
		if err != nil {
			log.Fatalf("failed: %s\n", err)
		}
		return
	}
	if err != nil {
		log.Fatalf("failed: %s\n", err)
	}
}

func touch(pathname string) {
	if hasFileOrDir(pathname) {
		return
	}
	log.Printf("creating file %q\n", pathname)
	_, err := os.Create(pathname)
	if err != nil {
		log.Fatalf("failed: %s\n", err)
	}
}
func fetch(remoteFilename string, outFileName string) {
	if hasFileOrDir(outFileName) {
		return
	}
	log.Printf("downloading %q\n", remoteFilename)

	url, err := url.JoinPath(baseUrl, remoteFilename)
	if err != nil {
		log.Fatalf("unexpected error: %s\n", err)
	}

	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("http request failed: %s\n", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("error reading response body: %s\n", err)
	}

	err = os.WriteFile(outFileName, body, 0o644)
	if err != nil {
		log.Fatalf("failed to write to file: %s\n", err)
	}

	log.Printf("downloaded %q\n", outFileName)
}
