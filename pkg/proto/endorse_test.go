package proto_test

import (
	"encoding/base64"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestEndorsementMessage(t *testing.T) {
	finalizedIDBase64 := "ZUUlMmISFnQqWlFgZmxAXW4lNUUnWF15YQp+IHd3TnYCIA8BWmRUfh4VNnYREQBXAARPcgkcYQ8FRw87BB4uMw=="
	endorsedIDBase64 := "GR40QGtzGDh+KWwjaUICbnV9Y28CFBwDeHFhXTMEHzxWPGxWe2IMbBtTYH0fE2gSal5CB1s2TRQoQ09PJVghHQ=="
	finalizedHeight := uint32(5)

	finalizedIDBytes, err := base64.StdEncoding.DecodeString(finalizedIDBase64)
	require.NoError(t, err)

	endorsedIDBytes, err := base64.StdEncoding.DecodeString(endorsedIDBase64)
	require.NoError(t, err)

	finalizedID, err := proto.NewBlockIDFromBytes(finalizedIDBytes)
	require.NoError(t, err)

	endorsedID, err := proto.NewBlockIDFromBytes(endorsedIDBytes)
	require.NoError(t, err)

	e := &proto.EndorseBlock{
		FinalizedBlockID:     finalizedID,
		FinalizedBlockHeight: finalizedHeight,
		EndorsedBlockID:      endorsedID,
	}

	got, err := e.EndorsementMessage()
	require.NoError(t, err)

	// Rebuild using the same concatenation as Scala
	expected := make([]byte, 0, len(finalizedIDBytes)+4+len(endorsedIDBytes))
	expected = append(expected, finalizedIDBytes...)
	h := make([]byte, 4)
	binary.BigEndian.PutUint32(h, finalizedHeight)
	expected = append(expected, h...)
	expected = append(expected, endorsedIDBytes...)

	require.Equal(t, expected, got, "endorsement message bytes must match Scala version")

	expectedBase64 := base64.StdEncoding.EncodeToString(expected)
	require.Equal(t,
		"ZUUlMmISFnQqWlFgZmxAXW4lNUUnWF15YQp+IHd3TnYCIA8BWmRUfh4VNnYREQBXAARPcgkcYQ8FRw87BB4uMwAAAAUZHjRAa"+
			"3MYOH4pbCNpQgJudX1jbwIUHAN4cWFdMwQfPFY8bFZ7YgxsG1NgfR8TaBJqXkIHWzZNFChDT08lWCEd",
		expectedBase64,
	)
}
