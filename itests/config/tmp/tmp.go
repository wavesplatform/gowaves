package main

import "github.com/wavesplatform/gowaves/itests/config"

func main() {
	_, _, err := config.CreateFileConfigs(true)
	if err != nil {
		panic(err.Error())
	}
}
