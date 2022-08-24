package internal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

const configPath = "/generate/ride_objects.json"

type actionField struct {
	Name  string   `json:"name"`
	Types []string `json:"types"`
	Order int      `json:"order"`
}

type actionsObject struct {
	Name   string        `json:"name"`
	Fields []actionField `json:"fields"`

	StructName string `json:"struct_name"`
	SetProofs  bool   `json:"set_proofs"`
}

type rideObjects struct {
	Actions []actionsObject `json:"actions"`
}

func parseConfig() (*rideObjects, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	configPath := filepath.Clean(filepath.Join(pwd, configPath))
	f, err := os.Open(configPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open file")
	}
	jsonParser := json.NewDecoder(f)
	s := &rideObjects{}
	if err = jsonParser.Decode(s); err != nil {
		return nil, errors.Wrap(err, "failed to decode ride objects config")
	}
	for i := 0; i < len(s.Actions); i++ {
		if s.Actions[i].StructName == "" {
			s.Actions[i].StructName = strings.ToUpper(string(s.Actions[i].Name[0])) + s.Actions[i].Name[1:]
		}
	}
	return s, nil
}
