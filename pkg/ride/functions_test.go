package ride

import (
	"math/rand"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNames(t *testing.T) {
	assert.Equal(t, "!", functionNameV2(0))
	assert.Equal(t, "!=", functionNameV3(1))
	assert.Equal(t, "wavesBalance", functionNameV2(68))
	assert.Equal(t, "DeleteEntry", functionNameV4(182))
}

func TestCheckFunction(t *testing.T) {
	for _, test := range []struct {
		name string
		id   int
	}{
		{"!", 0},
		{"!=", 1},
		{"420", 39},
		{"wavesBalance", 68},
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

func TestInvokeCallComplexityV5Constant(t *testing.T) {
	const (
		invokeFunctionID          = 1020
		reentrantInvokeFunctionID = 1021
	)
	catalogues := &[...]map[string]int{
		CatalogueV5,
		EvaluationCatalogueV5EvaluatorV1,
		EvaluationCatalogueV5EvaluatorV2,
	}
	for _, catalogue := range catalogues {
		assert.Equal(t, catalogue[strconv.Itoa(invokeFunctionID)], invokeCallComplexityV5)
		assert.Equal(t, catalogue[strconv.Itoa(reentrantInvokeFunctionID)], invokeCallComplexityV5)
	}
}
