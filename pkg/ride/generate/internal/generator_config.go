package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
)

type actionField struct {
	Name             string    `json:"name"`
	Types            typeInfos `json:"types"`
	Order            int       `json:"order"`            // order for string representation
	ConstructorOrder int       `json:"constructorOrder"` // order for constructor
}

type typeInfo interface {
	fmt.Stringer
	json.Unmarshaler
}

type typeInfos []typeInfo

func (infos *typeInfos) UnmarshalJSON(data []byte) error {
	var rawTypes []string
	if err := json.Unmarshal(data, &rawTypes); err != nil {
		return errors.Wrap(err, "typeInfos raw types unmarshal")
	}

	typeInfoList := make([]typeInfo, len(rawTypes))
	for i, name := range rawTypes {
		typeInfoList[i] = guessInfoType(name)
	}

	if err := json.Unmarshal(data, &typeInfoList); err != nil {
		return errors.Wrapf(err, "typeInfoList unmarshal(%s)", strings.Join(rawTypes, ","))
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

	if !strings.HasPrefix(source, info.String()) {
		return errors.Errorf("'%s' is missing: %s", info.String(), source)
	}
	source = strings.ReplaceAll(string(data), " ", "")

	begin, end := strings.Index(source, "["), strings.LastIndex(source, "]")
	if begin == -1 || end == -1 || begin == end {
		return errors.Errorf("bad brace sequence in elements types: %s", source)
	}
	begin++

	var typeNames []string
	opened := 0
	for cur := begin; cur < end; cur++ {
		switch source[cur] {
		case '[':
			opened++
		case ']':
			if opened == 0 {
				return errors.Errorf("bad bracket sequence: %s", source)
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
		return errors.Errorf("bad bracket sequence:%s", source)
	}

	var jsonStr strings.Builder
	jsonStr.WriteByte('[')
	for i, name := range typeNames {
		jsonStr.WriteString(strconv.Quote(name))
		if i != len(typeNames)-1 {
			jsonStr.WriteByte(',')
		}
	}
	jsonStr.WriteByte(']')

	return json.Unmarshal([]byte(jsonStr.String()), &info.elementsTypes)
}

func (info *listTypeInfo) ElementTypes() typeInfos {
	return info.elementsTypes
}

type actionsObject struct {
	LibVersion ast.LibraryVersion  `json:"version"`
	Deleted    *ast.LibraryVersion `json:"deleted,omitempty"`
	Fields     []actionField       `json:"fields"`

	StructName string `json:"struct_name,omitempty"`
	SetProofs  bool   `json:"set_proofs,omitempty"`
}

type rideObject struct {
	Name            string          `json:"name"`
	Actions         []actionsObject `json:"actions"`
	SkipConstructor bool            `json:"skip_constructor"`
}

type rideObjects struct {
	Objects []rideObject `json:"objects"`
}

func fillRideObjectStructNames(obj rideObject) error {
	if obj.Name == "" {
		return errors.New("empty name of object")
	}

	if len(obj.Actions) == 1 {
		obj.Actions[0].StructName = obj.Name
		return nil
	}

	// check versions duplicates
	versions := map[ast.LibraryVersion]struct{}{}
	for _, act := range obj.Actions {
		if _, ok := versions[act.LibVersion]; ok {
			return errors.Errorf("duplicated version (%d) for %s", act.LibVersion, obj.Name)
		}
		versions[act.LibVersion] = struct{}{}
	}

	sort.Slice(obj.Actions, func(i, j int) bool {
		return obj.Actions[i].LibVersion < obj.Actions[j].LibVersion
	})

	for i := 0; i < len(obj.Actions); i++ {
		obj.Actions[i].StructName = fmt.Sprintf("%sV%d", obj.Name, obj.Actions[i].LibVersion)
	}
	return nil
}

func parseConfig(configPath string) (_ *rideObjects, err error) {
	f, err := os.Open(configPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open file")
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			if err != nil {
				err = errors.Wrapf(err, "failed to close file: %v", closeErr)
			} else {
				err = closeErr
			}
		}
	}()
	jsonParser := json.NewDecoder(f)
	s := &rideObjects{}
	if err = jsonParser.Decode(s); err != nil {
		return nil, errors.Wrap(err, "failed to decode ride objects config")
	}

	for _, obj := range s.Objects {
		if err := fillRideObjectStructNames(obj); err != nil {
			return nil, err
		}
	}
	return s, nil
}
