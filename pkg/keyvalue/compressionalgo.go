package keyvalue

import "github.com/wavesplatform/goleveldb/leveldb/opt"

//go:generate go run github.com/dmarkham/enumer@v1.6.1 -type CompressionAlgo -trimprefix Compression -text -output compressionalgo_string.go
type CompressionAlgo opt.Compression

const (
	CompressionDefault = CompressionAlgo(opt.DefaultCompression) // = Snappy
	CompressionNone    = CompressionAlgo(opt.NoCompression)
	CompressionSnappy  = CompressionAlgo(opt.SnappyCompression)
)
