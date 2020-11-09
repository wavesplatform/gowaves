package rideTypes

import "github.com/wavesplatform/gowaves/pkg/ride"

type Tree interface {
	HasVerifier() bool
	IsDApp() bool
	GetTree() *ride.Tree
}