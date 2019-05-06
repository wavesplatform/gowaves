package main

import "github.com/wavesplatform/gowaves/pkg/settings"

func FromArgs(c *Cli) func(s *settings.NodeSettings) {
	return func(s *settings.NodeSettings) {
		s.DeclaredAddr = c.Run.DeclAddr
		s.HttpAddr = c.Run.HttpAddr
		s.WavesNetwork = c.Run.WavesNetwork
		s.Addresses = c.Run.Addresses
	}
}
