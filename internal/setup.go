package internal

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"

	"github.com/yusing/go-proxy/internal/common"
)

var (
	branch          = common.GetEnv("GOPROXY_BRANCH", "v0.6")
	baseURL         = "https://github.com/yusing/go-proxy/raw/" + branch
	requiredConfigs = []Config{
		{common.ConfigBasePath, true, false, ""},
		{common.ComposeFileName, false, true, common.ComposeExampleFileName},
		{path.Join(common.ConfigBasePath, common.ConfigFileName), false, true, common.ConfigExampleFileName},
	}
)

type Config struct {
	Pathname         string
	IsDir            bool
	NeedDownload     bool
	DownloadFileName string
}

func Setup() {
	log.Println("setting up go-proxy")
	log.Println("branch:", branch)

	if err := os.Chdir("/setup"); err != nil {
		log.Fatalf("failed: %s\n", err)
	}

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
		if remoteFilename == outFileName {
			log.Printf("%q already exists, not overwriting\n", outFileName)
			return
		}
		log.Printf("%q already exists, downloading to %q\n", outFileName, remoteFilename)
		outFileName = remoteFilename
	}
	log.Printf("downloading %q\n", remoteFilename)

	url, err := url.JoinPath(baseURL, remoteFilename)
	if err != nil {
		log.Fatalf("unexpected error: %s\n", err)
	}

	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("http request failed: %s\n", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		resp.Body.Close()
		log.Fatalf("error reading response body: %s\n", err)
	}

	err = os.WriteFile(outFileName, body, 0o644)
	if err != nil {
		resp.Body.Close()
		log.Fatalf("failed to write to file: %s\n", err)
	}

	log.Printf("downloaded to %q\n", outFileName)

	resp.Body.Close()
}
