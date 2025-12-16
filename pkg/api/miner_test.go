package api

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestApp_Miner(t *testing.T) {
	s := Scheduler{
		TimeNow: time.Unix(1559565012, 0),
	}

	bts, err := json.Marshal(s)
	require.NoError(t, err)

	require.Contains(t, string(bts), "2019-06-03T")
}
