package state

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestRewardVotesPackKey(t *testing.T) {
	tests := []struct {
		height   proto.Height
		keyBytes []byte
	}{
		{0, []byte{rewardVotesKeyPrefix, 0, 0, 0, 0, 0, 0, 0, 0}},
		{999, []byte{rewardVotesKeyPrefix, 0, 0, 0, 0, 0, 0, 0, 0}},
		{1000, []byte{rewardVotesKeyPrefix, 0, 0, 0, 0, 0, 0, 0, 1}},
		{1001, []byte{rewardVotesKeyPrefix, 0, 0, 0, 0, 0, 0, 0, 1}},
		{1500, []byte{rewardVotesKeyPrefix, 0, 0, 0, 0, 0, 0, 0, 1}},
		{2048, []byte{rewardVotesKeyPrefix, 0, 0, 0, 0, 0, 0, 0, 2}},
	}
	for i, tc := range tests {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			key := rewardVotesPackKey{tc.height}
			keyBytes := key.bytes()
			assert.Equal(t, tc.keyBytes, keyBytes)
		})
	}
}
