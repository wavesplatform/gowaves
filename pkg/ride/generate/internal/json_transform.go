package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
)

type actionFieldOld struct {
	Name             string   `json:"name"`
	Types            []string `json:"types"`
	Order            int      `json:"order"`
	ConstructorOrder int      `json:"constructorOrder"`
}

type actionsObjectOld struct {
	Name   string           `json:"name"`
	Fields []actionFieldOld `json:"fields"`

	StructName string `json:"struct_name"`
	SetProofs  bool   `json:"set_proofs"`
}

type rideObjectsOld struct {
	Actions []actionsObjectOld `json:"actions"`
}

type actionsObjectNew struct {
	LibVersion ast.LibraryVersion  `json:"version"`
	Deleted    *ast.LibraryVersion `json:"deleted,omitempty"`
	Fields     []actionFieldOld    `json:"fields"`

	StructName string `json:"struct_name,omitempty"`
	SetProofs  bool   `json:"set_proofs,omitempty"`
}

type rideObjectNew struct {
	Name    string             `json:"name"`
	Actions []actionsObjectNew `json:"actions"`
}

type rideObjectsNew struct {
	Objects []rideObjectNew `json:"objects"`
}

func parseConfigOld(fname string) (*rideObjectsOld, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	configPath := filepath.Clean(filepath.Join(pwd, fname))
	f, err := os.Open(configPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open file")
	}
	jsonParser := json.NewDecoder(f)
	s := &rideObjectsOld{}
	if err = jsonParser.Decode(s); err != nil {
		return nil, errors.Wrap(err, "failed to decode ride objects config")
	}

	return s, nil
}

func TransfromOldConfig(oldPath, newPath string) {
	old, err := parseConfigOld(oldPath)
	if err != nil {
		panic(err)
	}

	newConfig := rideObjectsNew{
		Objects: make([]rideObjectNew, 0),
	}

	objs := map[string]int{}

	for _, act := range old.Actions {
		var rideObj *rideObjectNew
		if _, ok := objs[act.Name]; !ok {
			newConfig.Objects = append(newConfig.Objects, rideObjectNew{
				Name:    strings.ToUpper(string(act.Name[0])) + act.Name[1:],
				Actions: []actionsObjectNew{},
			})
			objs[act.Name] = len(newConfig.Objects) - 1
		} else {
			fmt.Printf("else one %s found\n", act.Name)
		}

		rideObj = &newConfig.Objects[objs[act.Name]]
		rideObj.Actions = append(rideObj.Actions, actionsObjectNew{
			LibVersion: ast.LibV1,
			Fields:     act.Fields,
			SetProofs:  act.SetProofs,
		})
	}

	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	if jsonBytes, err := json.MarshalIndent(newConfig, "", "\t"); err == nil {
		os.WriteFile(filepath.Clean(filepath.Join(pwd, newPath)), jsonBytes, 0600)
	}
}
