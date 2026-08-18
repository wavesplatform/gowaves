package config

//go:generate go run github.com/dmarkham/enumer@v1.6.1 -type MiningType -json -output miningtype_string.go
type MiningType int

const (
	NoMining MiningType = iota
	GoMining
	ScalaMining
)
