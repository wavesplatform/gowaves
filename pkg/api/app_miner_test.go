package api

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestApp_Miner(t *testing.T) {
	s := Scheduler{
		TimeNow: time.Unix(1559565012, 0),
	}

	bts, err := json.Marshal(s)
	require.NoError(t, err)

	require.Contains(t, string(bts), "2019-06-03T")
}
