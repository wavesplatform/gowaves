package state

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	GENESIS_SIGNATURE = "FSH8eAAzZNqnG8xgTZtz5xuLqXySsXgAjmFEC25hXMbEufiGjqWPnGCZFt6gLiVLJny16ipxRNAkkzjjhqTjBE2"
)

var (
	signatures = []string{
		"2DVtfgXjpMeFf2PQCqvwxAiaGbiDsxDjSdNQkc5JQ74eWxjWFYgwvqzC4dn7iB1AhuM32WxEiVi1SGijsBtYQwn8",
		"2TsxPS216SsZJAiep7HrjZ3stHERVkeZWjMPFcvMotrdGpFa6UCCmoFiBGNizx83Ks8DnP3qdwtJ8WFcN9J4exa3",
		"3gF8LFjhnZdgEVjP7P6o1rvwapqdgxn7GCykCo8boEQRwxCufhrgqXwdYKEg29jyPWthLF5cFyYcKbAeFvhtRNTc",
		"5hjSPLDyqic7otvtTJgVv73H3o6GxgTBqFMTY2PqAFzw2GHAnoQddC4EgWWFrAiYrtPadMBUkoepnwFHV1yR6u6g",
		"ivP1MzTd28yuhJPkJsiurn2rH2hovXqxr7ybHZWoRGUYKazkfaL9MYoTUym4sFgwW7WB5V252QfeFTsM6Uiz3DM",
		"29gnRjk8urzqc9kvqaxAfr6niQTuTZnq7LXDAbd77nydHkvrTA4oepoMLsiPkJ8wj2SeFB5KXASSPmbScvBbfLiV",
	}
	recipients = []string{
		"3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ",
		"3P8JdJGYc7vaLu4UXUZc1iRLdzrkGtdCyJM",
		"3PAGPDPqnGkyhcihyjMHe9v36Y4hkAh9yDy",
		"3P9o3ZYwtHkaU1KxsKkFjJqJKS3dLHLC9oF",
		"3PJaDyprvekvPXPuAtxrapacuDJopgJRaU3",
		"3PBWXDFUc86N2EQxKJmW8eFco65xTyMZx6J",
	}
	ids = []string{
		"2DVtfgXjpMeFf2PQCqvwxAiaGbiDsxDjSdNQkc5JQ74eWxjWFYgwvqzC4dn7iB1AhuM32WxEiVi1SGijsBtYQwn8",
		"2TsxPS216SsZJAiep7HrjZ3stHERVkeZWjMPFcvMotrdGpFa6UCCmoFiBGNizx83Ks8DnP3qdwtJ8WFcN9J4exa3",
		"3gF8LFjhnZdgEVjP7P6o1rvwapqdgxn7GCykCo8boEQRwxCufhrgqXwdYKEg29jyPWthLF5cFyYcKbAeFvhtRNTc",
		"5hjSPLDyqic7otvtTJgVv73H3o6GxgTBqFMTY2PqAFzw2GHAnoQddC4EgWWFrAiYrtPadMBUkoepnwFHV1yR6u6g",
		"ivP1MzTd28yuhJPkJsiurn2rH2hovXqxr7ybHZWoRGUYKazkfaL9MYoTUym4sFgwW7WB5V252QfeFTsM6Uiz3DM",
		"29gnRjk8urzqc9kvqaxAfr6niQTuTZnq7LXDAbd77nydHkvrTA4oepoMLsiPkJ8wj2SeFB5KXASSPmbScvBbfLiV",
	}
	genesisTransactions = []proto.Genesis{
		{
			Type:      1,
			Timestamp: 1465742577614,
			Amount:    9999999500000000,
		}, {
			Type:      1,
			Timestamp: 1465742577614,
			Amount:    100000000,
		}, {
			Type:      1,
			Timestamp: 1465742577614,
			Amount:    100000000,
		}, {
			Type:      1,
			Timestamp: 1465742577614,
			Amount:    100000000,
		}, {
			Type:      1,
			Timestamp: 1465742577614,
			Amount:    100000000,
		}, {
			Type:      1,
			Timestamp: 1465742577614,
			Amount:    100000000,
		},
	}
)

func generateGenesisTransactions() ([]proto.Genesis, error) {
	res := make([]proto.Genesis, len(genesisTransactions), len(genesisTransactions))
	for i := 0; i < len(genesisTransactions); i++ {
		txFrame := genesisTransactions[i]
		id, err := crypto.NewSignatureFromBase58(ids[i])
		if err != nil {
			return nil, err
		}
		txFrame.ID = &id
		sig, err := crypto.NewSignatureFromBase58(signatures[i])
		if err != nil {
			return nil, err
		}
		txFrame.Signature = &sig
		recipient, err := proto.NewAddressFromString(recipients[i])
		if err != nil {
			return nil, err
		}
		txFrame.Recipient = recipient
		res[i] = txFrame
	}
	return res, nil
}
