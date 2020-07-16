package fride

type Program struct {
	Code      []byte
	Constants []rideType
	Functions map[string]rideFunction
}
