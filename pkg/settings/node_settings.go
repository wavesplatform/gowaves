package settings

import (
	"github.com/pkg/errors"
	"os"
	"strings"
)

type NodeSettings struct {
	DeclaredAddr string
	WavesNetwork string
	Addresses    string
	HttpAddr     string
	GrpcAddr     string
}

func (a NodeSettings) Validate() error {
	if len(a.WavesNetwork) == 0 {
		return errors.Errorf("empty WavesNetwork")
	}
	return nil
}

func FromJavaEnvironString(settings *NodeSettings, s string) {
	params := strings.Split(s, " ")

	for _, param := range params {
		if strings.HasPrefix(param, "-Dwaves.network.declared-address=") {
			settings.DeclaredAddr = strings.Replace(param, "-Dwaves.network.declared-address=", "", 1)
		}
	}
}

func FromJavaEnviron(settings *NodeSettings) error {
	s, _ := os.LookupEnv("WAVES_OPTS")
	FromJavaEnvironString(settings, s)
	return nil
}

func ApplySettings(settings *NodeSettings, f ...func(*NodeSettings) error) error {
	for _, fn := range f {
		if err := fn(settings); err != nil {
			return err
		}
	}
	return nil
}
