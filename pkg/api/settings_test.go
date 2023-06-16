package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateLimiterOptions(t *testing.T) {
	for _, test := range []struct {
		s    string
		fail bool
		opts *RateLimiterOptions
		err  string
	}{
		{"", false, &RateLimiterOptions{DefaultRateLimiterCacheSize, DefaultRateLimiterRPS, DefaultRateLimiterBurst}, ""},
		{"cache=12345&rps=67890&burst=13579", false, &RateLimiterOptions{12345, 67890, 13579}, ""},
		{"  cache=12345 &rps=67890 &burst=13579   ", false, &RateLimiterOptions{12345, 67890, 13579}, ""},
		{"rps=67890&burst=13579", false, &RateLimiterOptions{DefaultRateLimiterCacheSize, 67890, 13579}, ""},
		{"rps=67890", false, &RateLimiterOptions{DefaultRateLimiterCacheSize, 67890, DefaultRateLimiterBurst}, ""},
		{"rps=-1", false, &RateLimiterOptions{DefaultRateLimiterCacheSize, -1, DefaultRateLimiterBurst}, ""},
		{"cache=xxx&rps=67890&burst=13579", true, nil, "invalid rate limiter options: invalid value for key 'cache': strconv.ParseInt: parsing \"xxx\": invalid syntax"},
		{"cache=&rps=67890&burst=13579", true, nil, "invalid rate limiter options: invalid value for key 'cache': strconv.ParseInt: parsing \"\": invalid syntax"},
		{"cache=&rps=67890&burst=13579", true, nil, "invalid rate limiter options: invalid value for key 'cache': strconv.ParseInt: parsing \"\": invalid syntax"},
		{"shmesh=&RPS=67890", false, &RateLimiterOptions{DefaultRateLimiterCacheSize, DefaultRateLimiterBurst, DefaultRateLimiterBurst}, ""},
	} {
		opts, err := NewRateLimiterOptionsFromString(test.s)
		if test.fail {
			require.Error(t, err)
			assert.EqualError(t, err, test.err)
		} else {
			require.NoError(t, err)
			assert.NotNil(t, opts)
			assert.Equal(t, test.opts, opts)
		}
	}
}
