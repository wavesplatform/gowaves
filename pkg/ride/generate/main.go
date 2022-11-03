package main

import (
	"github.com/wavesplatform/gowaves/pkg/ride/generate/internal"
)

func main() {
	internal.GenerateConstants("constants.gen.go")
	internal.GenerateObjects("objects.gen.go")
	internal.GenerateConstructors("constructors.gen.go")
	// internal.GenerateFunctions must be invoked after internal.GenerateConstructors
	internal.GenerateFunctions("functions.gen.go")
	internal.GenerateFunctionFamilies("function_families.gen.go")
	internal.GenerateObjects("objects.gen.go")
	internal.GenerateTuples("tuples.gen.go")
}
