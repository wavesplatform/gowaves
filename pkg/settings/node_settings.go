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

func FromJavaEnviron(settings *NodeSettings) {
	s, _ := os.LookupEnv("WAVES_OPTS")
	FromJavaEnvironString(settings, s)
}

func ApplySettings(settings *NodeSettings, f ...func(*NodeSettings)) {
	for _, fn := range f {
		fn(settings)
	}
}
