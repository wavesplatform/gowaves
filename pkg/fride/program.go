package fride

type Program struct {
	LibVersion int
	EntryPoint int
	Code       []byte
	Constants  []rideType
	Functions  map[string]int
}
