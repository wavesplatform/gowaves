package gcradupchecker

import (
	"fmt"

	"github.com/throttled/throttled/v2"
	"github.com/throttled/throttled/v2/store/memstore"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

/*
	TODO(artemreyt): add description to this duplicator

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

var (
	lightWeightMessageList = proto.PeerMessageIDs{
		// TODO(artemreyt): fill
	}
	middleWeightMessageList = proto.PeerMessageIDs{
		// TODO(artemreyt): fill
	}
	heavyWeightMessageList = proto.PeerMessageIDs{
		// TODO(artemreyt): fill
	}
)

const separator = "|"

const defaultMaxMsgs = 1000

type DuplicateChecker struct {
	limiters [messageWeightTotal]throttled.RateLimiter
	settings *Settings
}

func NewDuplicateChecker(settings Settings) (*DuplicateChecker, error) {
	store, err := memstore.New(settings.MaxMsgs)
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
	return heavyMessageWeight // TODO(artemreyt): implement
}

// settings set limits for all message types
// and also separates messages by type
type Settings struct {
	Quotes   [messageWeightTotal]throttled.RateQuota
	MsgTypes map[proto.PeerMessageID]messageWeight
	MaxMsgs  int // max messages in store
}

func DefaultSettings() Settings {
	return Settings{
		Quotes:   defaultQuotes(),
		MsgTypes: defautMessageTypes(),
		MaxMsgs:  defaultMaxMsgs,
	}
}

func defaultQuotes() [messageWeightTotal]throttled.RateQuota {
	var quotes [messageWeightTotal]throttled.RateQuota

	quotes[lightMessageWeight] = throttled.RateQuota{
		MaxRate:  throttled.PerMin(2),
		MaxBurst: 0,
	}
	quotes[middleMessageWeight] = throttled.RateQuota{
		MaxRate:  throttled.PerMin(4),
		MaxBurst: 0,
	}
	quotes[heavyMessageWeight] = throttled.RateQuota{
		MaxRate:  throttled.PerMin(6),
		MaxBurst: 0,
	}

	return quotes
}

func defautMessageTypes() map[proto.PeerMessageID]messageWeight {
	msgTypes := make(map[proto.PeerMessageID]messageWeight)

	for _, msgs := range []struct {
		ids    proto.PeerMessageIDs
		weight messageWeight
	}{
		{ids: lightWeightMessageList, weight: lightMessageWeight},
		{ids: middleWeightMessageList, weight: middleMessageWeight},
		{ids: heavyWeightMessageList, weight: heavyMessageWeight},
	} {
		for _, id := range msgs.ids {
			msgTypes[id] = msgs.weight
		}
	}
	return msgTypes
}

// // TODO(artemreyt): implement
// func init() {

// }
