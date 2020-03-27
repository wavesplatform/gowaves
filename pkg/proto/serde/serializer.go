package serde

type Serializer interface {
	MarshalBinary() ([]byte, error)
}
