package clients

//go:generate stringer -type Implementation -trimprefix Node
type Implementation byte

const (
	NodeGo Implementation = iota
	NodeScala
)
