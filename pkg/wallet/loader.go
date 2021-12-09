package wallet

import (
	"io/ioutil"
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
		return ioutil.ReadFile(a.path)
	} else {
		u, err := user.Current()
		if err != nil {
			return nil, err
		}
		return ioutil.ReadFile(filepath.Join(u.HomeDir, ".waves"))
	}
}
