package genesis_generator

import (
	"github.com/stretchr/testify/require"
	. "github.com/wavesplatform/gowaves/pkg/proto"
	"testing"
)

func TestGenerate(t *testing.T) {

	require.Equal(t, 1,
		Generate(1558516864282, 'W',
			NewKeyPair([]byte("test")), 9000000000000000))

}
