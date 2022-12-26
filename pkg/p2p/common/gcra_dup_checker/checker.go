package gcradupchecker

import (
	"fmt"

	"github.com/throttled/throttled/v2"
	"github.com/throttled/throttled/v2/store/memstore"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

/*
	Map for RateLimiters "heavy", "middle", "light" with different settings.
	RateLimiter key = peerId + sep + fastHash(messageBytes)
*/

type messageWeight byte

const (
	lightMessageWeight messageWeight = iota
	middleMessageWeight
	heavyMessageWeight
	messageWeightTotal
)

const separator = "|"

type DuplicateChecker struct {
	limiters [messageWeightTotal]throttled.RateLimiter
}

func NewDuplicateChecker(maxMessages int) (*DuplicateChecker, error) {
	store, err := memstore.New(maxMessages)
	if err != nil {
		return nil, err
	}

	dc := new(DuplicateChecker)
	for i := range dc.limiters {
		limiter, err := throttled.NewGCRARateLimiter(
			store,
			throttled.RateQuota{
				MaxRate:  throttled.PerMin(2), // TODO(artemreyt): add settings
				MaxBurst: 0,
			},
		)
		if err != nil {
			return nil, err
		}
		dc.limiters[i] = limiter
	}
	return dc, nil
}

func (dc *DuplicateChecker) Add(peerID string, message []byte) bool {
	messageID, err := proto.UnmarshalMessageID(message)
	if err != nil {
		return false // TODO(artemreyt): think how to handle maybe
	}

	weight := messageWeightByID(messageID)

	key := fmt.Sprintf("%s%s%s", peerID, separator, crypto.MustFastHash(message)) // TODO(artemreyt): check how to concat it better

	limiter := dc.limiters[weight]

	ok, _, err := limiter.RateLimit(key, 1)
	if err != nil {
		return false
	}
	return ok
}

func messageWeightByID(id proto.PeerMessageID) messageWeight {
	return lightMessageWeight // TODO(artemreyt): implement
}

// TODO(artemreyt): implement
func init() {

}
