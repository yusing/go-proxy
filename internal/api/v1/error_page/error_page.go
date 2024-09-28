package error_page

import (
	"context"
	"fmt"
	"os"
	"path"
	"sync"

	api "github.com/yusing/go-proxy/internal/api/v1/utils"
	"github.com/yusing/go-proxy/internal/common"
	U "github.com/yusing/go-proxy/internal/utils"
	F "github.com/yusing/go-proxy/internal/utils/functional"
	W "github.com/yusing/go-proxy/internal/watcher"
	"github.com/yusing/go-proxy/internal/watcher/events"
)

const errPagesBasePath = common.ErrorPagesBasePath

var setup = sync.OnceFunc(func() {
	dirWatcher = W.NewDirectoryWatcher(context.Background(), errPagesBasePath)
	loadContent()
	go watchDir()
})

func GetStaticFile(filename string) ([]byte, bool) {
	return fileContentMap.Load(filename)
}

// try <statusCode>.html -> 404.html -> not ok
func GetErrorPageByStatus(statusCode int) (content []byte, ok bool) {
	content, ok = fileContentMap.Load(fmt.Sprintf("%d.html", statusCode))
	if !ok && statusCode != 404 {
		return fileContentMap.Load("404.html")
	}
	return
}

func loadContent() {
	files, err := U.ListFiles(errPagesBasePath, 0)
	if err != nil {
		api.Logger.Error(err)
		return
	}
	for _, file := range files {
		if fileContentMap.Has(file) {
			continue
		}
		content, err := os.ReadFile(file)
		if err != nil {
			api.Logger.Errorf("failed to read error page resource %s: %s", file, err)
			continue
		}
		file = path.Base(file)
		api.Logger.Infof("error page resource %s loaded", file)
		fileContentMap.Store(file, content)
	}
}

func watchDir() {
	eventCh, errCh := dirWatcher.Events(context.Background())
	for {
		select {
		case event, ok := <-eventCh:
			if !ok {
				return
			}
			filename := event.ActorName
			switch event.Action {
			case events.ActionFileWritten:
				fileContentMap.Delete(filename)
				loadContent()
			case events.ActionFileDeleted:
				fileContentMap.Delete(filename)
				api.Logger.Infof("error page resource %s deleted", filename)
			case events.ActionFileRenamed:
				api.Logger.Infof("error page resource %s deleted", filename)
				fileContentMap.Delete(filename)
				loadContent()
			}
		case err := <-errCh:
			api.Logger.Errorf("error watching error page directory: %s", err)
		}
	}
}

var dirWatcher W.Watcher
var fileContentMap = F.NewMapOf[string, []byte]()
