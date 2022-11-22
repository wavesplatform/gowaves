package stdlib

import (
	"embed"
	"encoding/json"
)

//go:embed vars.json
var embedVars embed.FS

type variableJson struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type Variable struct {
	Name string `json:"name"`
	Type Type
}

type StdVars struct {
	Vars []VarsInVersion `json:"vars"`
}

type VarsInVersion struct {
	AppendJson []variableJson `json:"append"`
	Remove     []string       `json:"remove"`
	Append     []Variable
}

var Vars = MustLoadVars()

func MustLoadVars() *StdVars {
	f, err := embedVars.ReadFile("vars.json")
	if err != nil {
		panic(err)
	}
	s := &StdVars{}
	if err = json.Unmarshal(f, s); err != nil {
		panic(err)
	}
	for i, ver := range s.Vars {
		s.Vars[i].Append = make([]Variable, len(ver.AppendJson))
		for j, variable := range ver.AppendJson {
			s.Vars[i].Append[j] = Variable{
				Name: variable.Name,
				Type: ParseType(variable.Type),
			}
		}
	}
	return s
}
