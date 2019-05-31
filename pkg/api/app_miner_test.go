package api

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestApp_Miner(t *testing.T) {

	s := Scheduler{
		TimeNow: time.Now(),
	}

	bts, err := json.Marshal(s)
	require.NoError(t, err)

	require.Equal(t, 1, string(bts))

}
