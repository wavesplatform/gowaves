package main

import "github.com/wavesplatform/gowaves/pkg/ride/generate/internal"

const configPath = "/generate/ride_objects_new.json"
const configPathOld = "/generate/ride_objects.json"

func main() {
	internal.TransfromOldConfig(configPathOld, configPath)
}
