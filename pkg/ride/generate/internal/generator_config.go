package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
)

type actionField struct {
	Name             string `json:"name"`
	Type             string `json:"type"`
	Order            int    `json:"order"`            // order for string representation
	ConstructorOrder int    `json:"constructorOrder"` // order for constructor
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
