package internal

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net"
	"testing"
)

func TestPeerDesignationString(t *testing.T) {
	pd := PeerDesignation{Address: net.IPv4(1, 2, 3, 4), Nonce: 1234567890}
	assert.Equal(t, "1.2.3.4-1234567890", pd.String())
	pd = PeerDesignation{Address:net.IPv4bcast, Nonce:0}
	assert.Equal(t, "255.255.255.255-0", pd.String())
}

func TestPeerDesignationMarshalJSON(t *testing.T) {
	pd := PeerDesignation{Address:net.IPv4(1, 2, 3, 4).To4(), Nonce: 567890}
	js, err := json.Marshal(pd)
	require.NoError(t, err)
	assert.Equal(t, "\"1.2.3.4-567890\"", string(js))
	pd = PeerDesignation{Address:net.IPv4bcast, Nonce:0}
	js, err = json.Marshal(pd)
	require.NoError(t, err)
	assert.Equal(t, "\"255.255.255.255-0\"", string(js))
}