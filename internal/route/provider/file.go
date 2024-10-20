package provider

import (
	"errors"
	"os"
	"path"

	"github.com/sirupsen/logrus"
	"github.com/yusing/go-proxy/internal/common"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/proxy/entry"
	R "github.com/yusing/go-proxy/internal/route"
	U "github.com/yusing/go-proxy/internal/utils"
	W "github.com/yusing/go-proxy/internal/watcher"
)

type FileProvider struct {
	fileName string
	path     string
}

func FileProviderImpl(filename string) (ProviderImpl, E.Error) {
	impl := &FileProvider{
		fileName: filename,
		path:     path.Join(common.ConfigBasePath, filename),
	}
	_, err := os.Stat(impl.path)
	switch {
	case err == nil:
		return impl, nil
	case errors.Is(err, os.ErrNotExist):
		return nil, E.NotExist("file", impl.path)
	default:
		return nil, E.UnexpectedError(err)
	}
}

func Validate(data []byte) E.Error {
	return U.ValidateYaml(U.GetSchema(common.FileProviderSchemaPath), data)
}

func (p FileProvider) String() string {
	return p.fileName
}

func (p *FileProvider) LoadRoutesImpl() (routes R.Routes, res E.Error) {
	routes = R.NewRoutes()

	b := E.NewBuilder("validation failure")
	defer b.To(&res)

	entries := entry.NewProxyEntries()

	data, err := E.Check(os.ReadFile(p.path))
	if err != nil {
		b.Add(E.FailWith("read file", err))
		return
	}

	if err = entries.UnmarshalFromYAML(data); err != nil {
		b.Add(err)
		return
	}

	if err := Validate(data); err != nil {
		logrus.Warn(err)
	}

	return R.FromEntries(entries)
}

func (p *FileProvider) NewWatcher() W.Watcher {
	return W.NewConfigFileWatcher(p.fileName)
}
