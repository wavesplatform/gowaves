package main

import (
	"flag"
	"path/filepath"

	"github.com/wavesplatform/gowaves/pkg/ride/generate/internal"
)

var (
	rideObjectsPath = flag.String("ride-objects-path", "generate/ride_objects.json", "Path to ride objects JSON file.")
	outputDir       = flag.String("output-dir", ".", "Directory where generated files will be placed.")
)

func filePath(fn string) string {
	return filepath.Join(*outputDir, fn)
}

func main() {
	flag.Parse()
	rideObjectsPath := *rideObjectsPath

	internal.GenerateConstants(filePath("constants.gen.go"))
	internal.GenerateObjects(rideObjectsPath, filePath("objects.gen.go"))
	internal.GenerateConstructors(rideObjectsPath, filePath("constructors.gen.go"))
	// internal.GenerateFunctions must be invoked after internal.GenerateConstructors
	internal.GenerateFunctions(filePath("functions.gen.go"))
	internal.GenerateFunctionFamilies(filePath("function_families.gen.go"))
	internal.GenerateTuples(filePath("tuples.gen.go"))
}
