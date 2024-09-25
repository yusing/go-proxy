package provider

import (
	"errors"
	"os"
	"path"

	"github.com/yusing/go-proxy/common"
	E "github.com/yusing/go-proxy/error"
	M "github.com/yusing/go-proxy/models"
	R "github.com/yusing/go-proxy/route"
	U "github.com/yusing/go-proxy/utils"
	W "github.com/yusing/go-proxy/watcher"
)

type FileProvider struct {
	fileName string
	path     string
}

func FileProviderImpl(filename string) (ProviderImpl, E.NestedError) {
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

func Validate(data []byte) E.NestedError {
	return U.ValidateYaml(U.GetSchema(common.FileProviderSchemaPath), data)
}

func (p FileProvider) String() string {
	return p.fileName
}

func (p FileProvider) OnEvent(event W.Event, routes R.Routes) (res EventResult) {
	b := E.NewBuilder("event %s error", event)
	defer b.To(&res.err)

	newRoutes, err := p.LoadRoutesImpl()
	if err.HasError() {
		b.Add(err)
		return
	}

	routes.RangeAll(func(_ string, v R.Route) {
		b.Add(v.Stop())
	})
	routes.Clear()

	newRoutes.RangeAll(func(_ string, v R.Route) {
		b.Add(v.Start())
	})

	routes.MergeFrom(newRoutes)
	return
}

func (p *FileProvider) LoadRoutesImpl() (routes R.Routes, res E.NestedError) {
	routes = R.NewRoutes()

	b := E.NewBuilder("file %q validation failure", p.fileName)
	defer b.To(&res)

	entries := M.NewProxyEntries()

	data, err := E.Check(os.ReadFile(p.path))
	if err.HasError() {
		b.Add(E.FailWith("read file", err))
		return
	}

	if !common.NoSchemaValidation {
		if err = Validate(data); err.HasError() {
			b.Add(err)
			return
		}
	}
	if err = entries.UnmarshalFromYAML(data); err.HasError() {
		b.Add(err)
		return
	}

	return R.FromEntries(entries)
}

func (p *FileProvider) NewWatcher() W.Watcher {
	return W.NewFileWatcher(p.fileName)
}
