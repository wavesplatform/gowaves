package client

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// unix: 	  1540296988
// Timestamp: 1540296988390
// nanos:	  1540296988390586000
func TestNewTimestampFromTime(t *testing.T) {
	now := time.Unix(0, 1540296988390586000)
	assert.Equal(t, uint64(1540296988390), NewTimestampFromTime(now))
}

func TestNewTimestampFromUnixNano(t *testing.T) {
	assert.Equal(t, uint64(1540296988390), NewTimestampFromUnixNano(1540296988390586000))
}
