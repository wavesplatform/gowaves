package utils

import (
	"github.com/spf13/afero"
	"io/ioutil"
	"os"
)

type Storage interface {
	Save([]byte) error
	Read() ([]byte, error)
	Close()
}

type FileBasedStorage struct {
	f afero.File
}

func NewFileBasedStorage(fs afero.Fs, pathToFile string) (*FileBasedStorage, error) {
	f, err := fs.OpenFile(pathToFile, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}

	return &FileBasedStorage{
		f: f,
	}, nil
}

func (a *FileBasedStorage) Save(b []byte) error {
	err := a.f.Truncate(0)
	if err != nil {
		return err
	}
	_, err = a.f.Seek(0, 0)
	if err != nil {
		return err
	}

	_, err = a.f.Write(b)
	if err != nil {
		return err
	}
	return nil
}

func (a *FileBasedStorage) Read() ([]byte, error) {
	_, err := a.f.Seek(0, 0)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(a.f)
}

func (a *FileBasedStorage) Close() {
	_ = a.f.Close()
}

type NoOnStorage struct{}

func (a NoOnStorage) Read() ([]byte, error) {
	return []byte{}, nil
}

func (a NoOnStorage) Save(b []byte) error {
	return nil
}

func (a NoOnStorage) Close() {}
