package bytespool

type Pool interface {
	Get() []byte
	Put([]byte)
	BytesLen() int
}
