package itests

import (
	"testing"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
)

type SponsorshipTxApiSuite struct {
	f.BaseSuite
}

func TestSponsorshipTxApiSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SponsorshipTxApiSuite))
}
