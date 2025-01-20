package provider

import (
	"os"
	"path"
	"strings"

	"github.com/rs/zerolog"
	"github.com/yusing/go-proxy/internal/common"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/logging"
	"github.com/yusing/go-proxy/internal/route"
	"github.com/yusing/go-proxy/internal/utils"
	W "github.com/yusing/go-proxy/internal/watcher"
)

type FileProvider struct {
	fileName string
	path     string
	l        zerolog.Logger
}

func FileProviderImpl(filename string) (ProviderImpl, error) {
	impl := &FileProvider{
		fileName: filename,
		path:     path.Join(common.ConfigBasePath, filename),
		l:        logging.With().Str("type", "file").Str("name", filename).Logger(),
	}
	_, err := os.Stat(impl.path)
	if err != nil {
		return nil, err
	}
	return impl, nil
}

func validate(provider string, data []byte) (route.Routes, E.Error) {
	entries, err := utils.DeserializeYAMLMap[*route.RawEntry](data)
	if err != nil {
		return route.NewRoutes(), err
	}
	return route.FromEntries(provider, entries)
}

func Validate(data []byte) (err E.Error) {
	_, err = validate("", data)
	return
}

func (p *FileProvider) String() string {
	return p.fileName
}

func (p *FileProvider) ShortName() string {
	return strings.Split(p.fileName, ".")[0]
}

func (p *FileProvider) IsExplicitOnly() bool {
	return false
}

func (p *FileProvider) Logger() *zerolog.Logger {
	return &p.l
}

func (p *FileProvider) loadRoutesImpl() (route.Routes, E.Error) {
	routes := route.NewRoutes()

	data, err := os.ReadFile(p.path)
	if err != nil {
		return routes, E.From(err)
	}

	return validate(p.ShortName(), data)
}

func (p *FileProvider) NewWatcher() W.Watcher {
	return W.NewConfigFileWatcher(p.fileName)
}
