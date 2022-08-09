package wallet

import (
	"os"
	"os/user"
	"path/filepath"
)

type Loader interface {
	Load() ([]byte, error)
}

type LoaderImpl struct {
	path string
}

func NewLoader(path string) LoaderImpl {
	return LoaderImpl{path: path}
}

func (a LoaderImpl) Load() ([]byte, error) {
	if a.path != "" {
		return os.ReadFile(a.path)
	} else {
		u, err := user.Current()
		if err != nil {
			return nil, err
		}
		return os.ReadFile(filepath.Join(u.HomeDir, ".waves"))
	}
}
