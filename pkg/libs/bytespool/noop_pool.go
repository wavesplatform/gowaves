package bytespool

type NoOpBytesPool struct {
	bytesLen int
}

func (a *NoOpBytesPool) Get() []byte {
	return make([]byte, a.bytesLen)
}

func (a *NoOpBytesPool) Put([]byte) {
	// just skip
}

func (a *NoOpBytesPool) BytesLen() int {
	return a.bytesLen
}

// poolSize is the maximum number of elements can be stored in pool
// bytesLength is the size of byte array, like cap([]byte{})
func NewNoOpBytesPool(bytesLength int) *NoOpBytesPool {
	if bytesLength < 1 {
		panic("bytesLen should be positive")
	}

	return &NoOpBytesPool{
		bytesLen: bytesLength,
	}
}
