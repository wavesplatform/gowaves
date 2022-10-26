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
	var rawTypes []*string // mb *string?
	if err := json.Unmarshal(data, &rawTypes); err != nil {
		return errors.Wrap(err, "typeInfos raw types unmarshal")
	}

	typeInfoList := make([]typeInfo, len(rawTypes))
	for i, name := range rawTypes {
		typeInfoList[i] = guessInfoType(*name)
	}

	if err := json.Unmarshal(data, &typeInfoList); err != nil {
		return errors.Wrapf(err, "typeInfoList unmarshal(%s)", data)
	}
	*infos = typeInfoList

	return nil
}

func guessInfoType(typeName string) typeInfo {
	if strings.HasPrefix(typeName, "rideList") {
		return &listTypeInfo{}
	}
	return &simpleTypeInfo{}
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
	var source string
	if err := json.Unmarshal(data, &source); err != nil {
		return errors.Wrap(err, "listTypeInfo unmarshal raw string")
	}

	if !strings.HasPrefix(source, "rideList") {
		return errors.Errorf("'rideList' is missing: %s", source)
	}
	source = strings.ReplaceAll(string(data), " ", "")

	begin, end := strings.Index(source, "["), strings.LastIndex(source, "]")
	if begin == -1 || end == -1 || begin == end {
		return errors.Errorf("bad brace sequence in elements types: %s", source)
	}
	begin++

	typeNames := []string{}
	opened := 0
	for cur := begin; cur < end; cur++ {
		switch source[cur] {
		case '[':
			opened++
		case ']':
			if opened == 0 {
				return errors.New("bad bracket sequence")
			}
			opened--
		case '|':
			if opened == 0 {
				typeNames = append(typeNames, source[begin:cur])
				begin = cur + 1
			}
		}
	}
	typeNames = append(typeNames, source[begin:end])
	if opened != 0 {
		return errors.New("bad bracket sequence")
	}

	var jsonStr strings.Builder
	jsonStr.WriteByte('[')
	for i, name := range typeNames {
		jsonStr.WriteByte('"')
		jsonStr.WriteString(name)
		jsonStr.WriteByte('"')
		if i != len(typeNames)-1 {
			jsonStr.WriteByte(',')
		}
	}
	jsonStr.WriteByte(']')

	return json.Unmarshal([]byte(jsonStr.String()), &info.elementsTypes)
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
