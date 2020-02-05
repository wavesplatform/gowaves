package ntptime

import (
	"time"

	"github.com/beevik/ntp"
)

type stub struct {
	resp *ntp.Response
	err  error
}

func (a stub) Query(addr string) (*ntp.Response, error) {
	return a.resp, a.err
}

type Stub struct {
}

func (s Stub) Now() time.Time {
	return time.Now()
}
