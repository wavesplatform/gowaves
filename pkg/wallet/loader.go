package wallet

import (
	"io/ioutil"
	"os/user"
	"path"
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
		home, err := userHomeDir()
		if err != nil {
			return nil, err
		}
		return ioutil.ReadFile(path.Join(home, ".waves"))
	}
}

func userHomeDir() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return u.HomeDir, nil
}
