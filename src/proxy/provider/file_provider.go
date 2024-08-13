package provider

import (
	"os"
	"path"

	"github.com/yusing/go-proxy/common"
	E "github.com/yusing/go-proxy/error"
	M "github.com/yusing/go-proxy/models"
	U "github.com/yusing/go-proxy/utils"
	W "github.com/yusing/go-proxy/watcher"
)

type FileProvider struct {
	fileName string
	path     string
}

func FileProviderImpl(filename string) ProviderImpl {
	return &FileProvider{
		fileName: filename,
		path:     path.Join(common.ConfigBasePath, filename),
	}
}

func Validate(data []byte) E.NestedError {
	return U.ValidateYaml(U.GetSchema(common.ProvidersSchemaPath), data)
}

func (p *FileProvider) String() string {
	return p.fileName
}

func (p *FileProvider) GetProxyEntries() (M.ProxyEntries, E.NestedError) {
	entries := M.NewProxyEntries()
	data, err := E.Check(os.ReadFile(p.path))
	if err.IsNotNil() {
		return entries, E.Failure("read file").Subject(p).With(err)
	}
	ne := E.Failure("validation").Subject(p)
	if !common.NoSchemaValidation {
		if err = Validate(data); err.IsNotNil() {
			return entries, ne.With(err)
		}
	}
	if err = entries.UnmarshalFromYAML(data); err.IsNotNil() {
		return entries, ne.With(err)
	}
	return entries, E.Nil()
}

func (p *FileProvider) NewWatcher() W.Watcher {
	return W.NewFileWatcher(p.fileName)
}
