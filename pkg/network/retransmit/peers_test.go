package retransmit

import (
	"github.com/magiconair/properties/assert"
	"github.com/segmentio/objconv/json"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"io/ioutil"
	"testing"
)

func TestNewKnownPeersFileBased(t *testing.T) {

	pathToFile := "/path/to/file"

	fs := afero.NewMemMapFs()
	p, err := NewKnownPeersFileBased(fs, pathToFile)
	require.NoError(t, err)

	p.Add("127.0.0.1:6868", proto.Version{})
	p.Stop()

	f, err := fs.Open(pathToFile)
	require.NoError(t, err)

	bts, err := ioutil.ReadAll(f)
	require.NoError(t, err)

	var rows []JsonKnowPeerRow
	err = json.Unmarshal(bts, &rows)
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1:6868", rows[0].Addr)
}
