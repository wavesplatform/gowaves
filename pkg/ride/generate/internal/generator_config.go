package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

const configPath = "/generate/ride_objects.json"

type actionField struct {
	Name             string    `json:"name"`
	Types            typeInfos `json:"types"`
	Order            int       `json:"order"`            // order for string representation
	ConstructorOrder int       `json:"constructorOrder"` // order for constructor
}

type typeInfos []typeInfo

func (infos *typeInfos) UnmarshalJSON(data []byte) error {
	var rawTypes []string // mb *string?
	if err := json.Unmarshal(data, &rawTypes); err != nil {
		return errors.Wrap(err, "typeInfos raw types unmarshal")
	}

	typeInfoList := make([]typeInfo, len(rawTypes))
	for i, name := range rawTypes {
		typeInfoList[i] = guessInfoType(name)
	}

	if err := json.Unmarshal(data, &typeInfoList); err != nil {
		return errors.Wrap(err, "typeInfoList unmarshal")
	}
	*infos = typeInfoList

	return nil
}

func guessInfoType(typeName string) typeInfo {
	switch typeName {
	case "rideList":
		return &listTypeInfo{}
	default:
		return &simpleTypeInfo{}
	}
}

type typeInfo interface {
	fmt.Stringer
	json.Unmarshaler
}

type simpleTypeInfo struct {
	name string
}

func (info *simpleTypeInfo) String() string {
	return info.name
}

func (info *simpleTypeInfo) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &info.name); err != nil {
		return errors.Wrap(err, "unmarshal type name")
	}
	return nil
}

type listTypeInfo struct {
	elementsTypes typeInfos
}

func (info *listTypeInfo) String() string {
	return "rideList"
}

func (info *listTypeInfo) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &info.elementsTypes); err != nil {
		return errors.Wrap(err, "elementsTypes list unmarshal")
	}
	return nil
}

func (info listTypeInfo) ElementTypes() typeInfos {
	return info.elementsTypes
}

type actionsObject struct {
	Name   string        `json:"name"`
	Fields []actionField `json:"fields"`

	StructName     string `json:"struct_name"`
	SetProofs      bool   `json:"set_proofs"`
	GenConstructor bool   `json:"generateConstructor"`
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
