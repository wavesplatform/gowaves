package ride

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNames(t *testing.T) {
	assert.Equal(t, "!", functionNameV2(0))
	assert.Equal(t, "!=", functionNameV3(1))
	assert.Equal(t, "TransferTransaction", functionNameV2(68))
	assert.Equal(t, "BurnTransaction", functionNameV4(182))
}

func TestCheckFunction(t *testing.T) {
	for _, test := range []struct {
		name string
		id   int
	}{
		{"!", 0},
		{"!=", 1},
		{"420", 39},
		{"wavesBalance", 82},
	} {
		id, ok := checkFunctionV2(test.name)
		assert.True(t, ok)
		assert.Equal(t, test.id, int(id))
	}
}

func BenchmarkCheckFunction(b *testing.B) {
	l := len(_functions_V4)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := rand.Intn(l)
		name := functionNameV4(id)
		_, ok := checkFunctionV4(name)
		assert.True(b, ok)
	}
}

func BenchmarkCheckFunctionMap(b *testing.B) {
	l := len(_functions_V4)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := rand.Intn(l)
		name := functionNameV4(id)
		_, ok := CatalogueV4[name]
		assert.True(b, ok)
	}
}
