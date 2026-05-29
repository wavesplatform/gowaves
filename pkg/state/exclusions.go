package state

import (
	"sync"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

var (
	id1   = crypto.MustDigestFromBase58("Bcg324gLYmiPorzPYceVE6t5Spk2fupmht96jzz9gpnh")
	diff1 = makeDiff1()
)

// AssetID: 3v7zGkeHS6KrsvmTRzEzvCxm5cdzkCtM7z5cM6efcjCB => 	0x2b, 0x53, 0x0e, 0xb5, 0x9d, 0x6c, 0x31, 0x7b, 0xb7, 0xbd, 0xb1, 0x65, 0x74, 0xb1, 0x5d, 0x58, 0x1d, 0xd3, 0x5a, 0xe1
// SenderID: 3P2q6tfVRAhiZUoydGYwDJPybgDhSAtHGRo => 			0x09, 0xdf, 0xc8, 0x5c, 0x44, 0xd4, 0xf1, 0x38, 0x92, 0xbe, 0xef, 0x30, 0x5a, 0xc9, 0x9f, 0x63, 0x88, 0xa4, 0xd8, 0x28
// RecipientID: 3PMLDU7uLs9diGm4vg2tHjG9TocMx5jhDMn => 		0xd4, 0xd3, 0x99, 0x86, 0x79, 0xb9, 0xae, 0x86, 0x47, 0x34, 0xc4, 0xad, 0xfd, 0xf0, 0x53, 0xa3, 0x04, 0x04, 0x37, 0x07
// Amount: -700000
func makeDiff1() txDiff {
	diff := newTxDiff()
	senderKey := assetBalanceKey{
		address: proto.AddressID{0x09, 0xdf, 0xc8, 0x5c, 0x44, 0xd4, 0xf1, 0x38, 0x92, 0xbe, 0xef, 0x30, 0x5a, 0xc9, 0x9f, 0x63, 0x88, 0xa4, 0xd8, 0x28},
		asset:   proto.AssetID{0x2b, 0x53, 0x0e, 0xb5, 0x9d, 0x6c, 0x31, 0x7b, 0xb7, 0xbd, 0xb1, 0x65, 0x74, 0xb1, 0x5d, 0x58, 0x1d, 0xd3, 0x5a, 0xe1},
	}
	if err := diff.appendBalanceDiff(senderKey.bytes(), newBalanceDiff(700000, 0, 0, false)); err != nil {
		panic(err)
	}
	receiverKey := assetBalanceKey{
		address: proto.AddressID{0xd4, 0xd3, 0x99, 0x86, 0x79, 0xb9, 0xae, 0x86, 0x47, 0x34, 0xc4, 0xad, 0xfd, 0xf0, 0x53, 0xa3, 0x04, 0x04, 0x37, 0x07},
		asset:   proto.AssetID{0x2b, 0x53, 0x0e, 0xb5, 0x9d, 0x6c, 0x31, 0x7b, 0xb7, 0xbd, 0xb1, 0x65, 0x74, 0xb1, 0x5d, 0x58, 0x1d, 0xd3, 0x5a, 0xe1},
	}
	if err := diff.appendBalanceDiff(receiverKey.bytes(), newBalanceDiff(-700000, 0, 0, false)); err != nil {
		panic(err)
	}
	return diff
}

func leasesToDisabledAliasesMainnet() []crypto.Digest {
	return []crypto.Digest{
		crypto.MustDigestFromBase58("56unoFHzWTzosYtqW9jd5Dk7P3YNPGPP7TN4bs4tm95W"),
		crypto.MustDigestFromBase58("2F95GigvHjSAfhzpcP5UQSkye4PsVvkEnwww622QWpiU"),
		crypto.MustDigestFromBase58("G7S4thYdMofaE4L33DjMnorsgvUvYBgnhK1yuJDm3Wfe"),
		crypto.MustDigestFromBase58("7MuFHv68dJASmroXUwav2DTk6fsUq3GYC6znnJ8c4A2w"),
		crypto.MustDigestFromBase58("GBJhccWynDovWPpnTzUGSB8mLVg7WRriDSjdMPgajwyP"),
	}
}

const (
	firstAbnormalTxsMainnetHeight        = 5228053 // 7CGkoGMTqJ9hor87qXftUXWABxxoXqEPt4hWS43M34bs
	lastAbnormalTxsMainnetHeight         = 5233108 // 5U9QQ2dwQ7iEAUQFPPm74K7ZT9wpzNMaUDPDvkFpWwsU
	nextHeightAfterLastAbnormalTxMainnet = lastAbnormalTxsMainnetHeight + 1
)

type abnormalTxType struct {
	snapshot                 txSnapshot
	affectedAddressesNoMiner []proto.WavesAddress
}

//nolint:gochecknoglobals // special case
var (
	abnormalTxsMainnetInitializer sync.Once
	abnormalTxsMainnetCleaner     sync.Once
	abnormalTxsMainnet            map[crypto.Digest]abnormalTxType
)

func getAbnormalTxMainnet(txID crypto.Digest) (abnormalTxType, bool) {
	abnormalTxsMainnetInitializer.Do(func() { abnormalTxsMainnet = newAbnormalTxsMainnet() })
	tx, ok := abnormalTxsMainnet[txID]
	return tx, ok
}

func cleanAbnormalTxsMainnet() {
	abnormalTxsMainnetCleaner.Do(func() { abnormalTxsMainnet = nil })
}

//nolint:funlen // abnormal txs generator
func newAbnormalTxsMainnet() map[crypto.Digest]abnormalTxType {
	return map[crypto.Digest]abnormalTxType{
		crypto.MustDigestFromBase58("7CGkoGMTqJ9hor87qXftUXWABxxoXqEPt4hWS43M34bs"): {
			snapshot: txSnapshot{
				regular: []proto.AtomicSnapshot{
					&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PGKyBATDndVEZYYHCNCkEwJNWpdmAPsuHi"),
						Balance: 97000000,
					},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PK3Px9gUSakBV1pqb7kEd8YHYYzxoE7BVu"),
						Balance: 335518911,
					},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PQ9hZ36dyXGcqabcrHXsjP9PaQMqy69yeE"),
						Balance: 163450859222,
					},
				},
			},
			affectedAddressesNoMiner: []proto.WavesAddress{
				proto.MustAddressFromString("3PGKyBATDndVEZYYHCNCkEwJNWpdmAPsuHi"),
				proto.MustAddressFromString("3PK3Px9gUSakBV1pqb7kEd8YHYYzxoE7BVu"),
			},
		},
		crypto.MustDigestFromBase58("75PpouqFkWPFvaHnh8j2b1pSRBqgjS2VYXXww2HT71WR"): {
			snapshot: txSnapshot{
				regular: []proto.AtomicSnapshot{
					&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PGKyBATDndVEZYYHCNCkEwJNWpdmAPsuHi"),
						Balance: 96000000,
					},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PBitcoinAVGJvpSe6jevepPbT3M8SoMZjb"),
						Balance: 10964180000,
					},
				},
			},
			affectedAddressesNoMiner: []proto.WavesAddress{
				proto.MustAddressFromString("3PGKyBATDndVEZYYHCNCkEwJNWpdmAPsuHi"),
			},
		},
		crypto.MustDigestFromBase58("GzeUXkPWzrj5RoGTECW1Myp7Vpx4PdU5ohDWUg74mbdM"): {
			snapshot: txSnapshot{
				regular: []proto.AtomicSnapshot{
					&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
						Balance: 19798000,
					},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PK3Px9gUSakBV1pqb7kEd8YHYYzxoE7BVu"),
						Balance: 6220911,
					},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PA1KvFfq9VuJjg45p2ytGgaNjrgnLSgf4r"),
						Balance: 360738078534,
					},
				},
			},
			affectedAddressesNoMiner: []proto.WavesAddress{
				proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
				proto.MustAddressFromString("3PK3Px9gUSakBV1pqb7kEd8YHYYzxoE7BVu"),
			},
		},
		crypto.MustDigestFromBase58("5mSU5EUrWhUoLK82bavSM6m7YShjYHQwfvKHjMTj1D3t"): {
			snapshot: txSnapshot{
				regular: []proto.AtomicSnapshot{
					&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
						Balance: 1759780,
					},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PQ9hZ36dyXGcqabcrHXsjP9PaQMqy69yeE"),
						Balance: 528759572383,
					},
				},
			},
			affectedAddressesNoMiner: []proto.WavesAddress{
				proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
			},
		},
		crypto.MustDigestFromBase58("2FHrTabz88k3ZPzJdqqr4KNvN1HjnURXvmCyZPy4DFtU"): {
			snapshot: txSnapshot{
				regular: []proto.AtomicSnapshot{
					&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
						Balance: 591901,
					},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PA1KvFfq9VuJjg45p2ytGgaNjrgnLSgf4r"),
						Balance: 436497306522,
					},
				},
			},
			affectedAddressesNoMiner: []proto.WavesAddress{
				proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
			},
		},
		crypto.MustDigestFromBase58("6pjCHjBkeofm9ZkQ7gooDDy63XmcNPdk2zBiSJf1YK9h"): {
			snapshot: txSnapshot{
				regular: []proto.AtomicSnapshot{
					&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
						Balance: 491901,
					},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PA1KvFfq9VuJjg45p2ytGgaNjrgnLSgf4r"),
						Balance: 436499546522,
					},
				},
			},
			affectedAddressesNoMiner: []proto.WavesAddress{
				proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
			},
		},
		crypto.MustDigestFromBase58("Bkp8bU6EEfDanHcHMu97Ncga7ndjUB7NCg4P3TU11dYY"): {
			snapshot: txSnapshot{
				regular: []proto.AtomicSnapshot{
					&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
						Balance: 391901,
					},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PQ9hZ36dyXGcqabcrHXsjP9PaQMqy69yeE"),
						Balance: 589562261126,
					},
				},
			},
			affectedAddressesNoMiner: []proto.WavesAddress{
				proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
			},
		},
		crypto.MustDigestFromBase58("BKSHqtySjV8HEsDSPRSz1e7CsTTkE1542mXpaLpf8Kce"): {
			snapshot: txSnapshot{
				regular: []proto.AtomicSnapshot{
					&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
						Balance: 191901,
					},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PQ9hZ36dyXGcqabcrHXsjP9PaQMqy69yeE"),
						Balance: 589563181126,
					},
				},
			},
			affectedAddressesNoMiner: []proto.WavesAddress{
				proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
			},
		},
		crypto.MustDigestFromBase58("AqwuzGEexbHCTwY2iGG3X3jmNGdWW5HcHT1g5ErbKeRV"): {
			snapshot: txSnapshot{
				regular: []proto.AtomicSnapshot{
					&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PR78LWNk8zUbPrvqo79k1rdcvieF8MYRjB"),
						Balance: 40891067,
					},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PP4nrxNnL3xRkMAaUWXnerryUDVEttAurA"),
						Balance: 57620405078,
					},
				},
			},
			affectedAddressesNoMiner: []proto.WavesAddress{
				proto.MustAddressFromString("3PR78LWNk8zUbPrvqo79k1rdcvieF8MYRjB"),
			},
		},
		crypto.MustDigestFromBase58("EbSCct4oR4QdvwuHzQLAP2JG3PGJ2JeEiyV9DutJ1fcq"): {
			snapshot: txSnapshot{
				regular: []proto.AtomicSnapshot{
					&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PR78LWNk8zUbPrvqo79k1rdcvieF8MYRjB"),
						Balance: 40681067,
					},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PDETXtiaErZncMduS8h9G6aopcjT7wheqj"),
						Balance: 155664879314,
					},
				},
			},
			affectedAddressesNoMiner: []proto.WavesAddress{
				proto.MustAddressFromString("3PR78LWNk8zUbPrvqo79k1rdcvieF8MYRjB"),
			},
		},
		crypto.MustDigestFromBase58("AVVZxXE9w8DWdwGnPCj2KxrcUTcJNVeaLTavghHskQkD"): {
			snapshot: txSnapshot{
				regular: []proto.AtomicSnapshot{
					&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
						Balance: 291900,
					},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PDETXtiaErZncMduS8h9G6aopcjT7wheqj"),
						Balance: 155664919314,
					},
				},
			},
			affectedAddressesNoMiner: []proto.WavesAddress{
				proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
			},
		},
		crypto.MustDigestFromBase58("HPZjkZEA6ysctUCVbAPMK1ftegQzvuQt7ThrANgEFy3F"): {
			snapshot: txSnapshot{
				regular: []proto.AtomicSnapshot{
					&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
						Balance: 491900,
					},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3P3RZeHi4LTjpZdpw7kmkVSbQ84qDfrVy8G"),
						Balance: 48659420157,
					},
				},
			},
			affectedAddressesNoMiner: []proto.WavesAddress{
				proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
			},
		},
		crypto.MustDigestFromBase58("4f5y351nPsz5Ja1sDoxf6GT9mAjs9xC9qRb3u8TckgV7"): {
			snapshot: txSnapshot{
				regular: []proto.AtomicSnapshot{
					&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
						Balance: 391900,
					},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PLp1QsFxukK5nnTBYHAqjz9duWMriDkHeT"),
						Balance: 1182208740935,
					},
				},
			},
			affectedAddressesNoMiner: []proto.WavesAddress{
				proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
			},
		},
		crypto.MustDigestFromBase58("3z7T61bGRbfC31ACb6Zqgnr4VUvhmLf41TUdrgPbisNU"): {
			snapshot: txSnapshot{
				regular: []proto.AtomicSnapshot{
					&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
						Balance: 291900,
					},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PLp1QsFxukK5nnTBYHAqjz9duWMriDkHeT"),
						Balance: 1182208780935,
					},
				},
			},
			affectedAddressesNoMiner: []proto.WavesAddress{
				proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
			},
		},
		crypto.MustDigestFromBase58("6Hubtb3JVECQY5MM7cVUf6v2gKX8yeYABEvfAycwLE1X"): {
			snapshot: txSnapshot{
				regular: []proto.AtomicSnapshot{
					&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
						Balance: 191900,
					},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PLp1QsFxukK5nnTBYHAqjz9duWMriDkHeT"),
						Balance: 1182213540935,
					},
				},
			},
			affectedAddressesNoMiner: []proto.WavesAddress{
				proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
			},
		},
		crypto.MustDigestFromBase58("8jMgaVYRezWqWBQU9gxkfLDyEp3bosqvm1F43emmgzkQ"): {
			snapshot: txSnapshot{
				regular: []proto.AtomicSnapshot{
					&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
						Balance: 91900,
					},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PQ9hZ36dyXGcqabcrHXsjP9PaQMqy69yeE"),
						Balance: 44820360000,
					},
				},
			},
			affectedAddressesNoMiner: []proto.WavesAddress{
				proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
			},
		},
		crypto.MustDigestFromBase58("HGrLmf9izpT3kcXehbxSERuMWNYfimUgj1gLm6Z1NY8e"): {
			snapshot: txSnapshot{
				regular: []proto.AtomicSnapshot{
					&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
						Balance: 891900,
					},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3P9DEDP5VbyXQyKtXDUt2crRPn5B7gs6ujc"),
						Balance: 1660877589117,
					},
				},
			},
			affectedAddressesNoMiner: []proto.WavesAddress{
				proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
			},
		},
		crypto.MustDigestFromBase58("77A9dh2KXQwMonWTDn1qGhBxGvkmgX4Dfi5dGZNJ41nY"): {
			snapshot: txSnapshot{
				regular: []proto.AtomicSnapshot{
					&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
						Balance: 791900,
					},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3P9DEDP5VbyXQyKtXDUt2crRPn5B7gs6ujc"),
						Balance: 1660878069117,
					},
				},
			},
			affectedAddressesNoMiner: []proto.WavesAddress{
				proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
			},
		},
		crypto.MustDigestFromBase58("3rVKoEsNJDAXo3zqpEryK2aScjUW6VHZc4b97niiirHX"): {
			snapshot: txSnapshot{
				regular: []proto.AtomicSnapshot{
					&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
						Balance: 691900,
					},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PKgMeEpfgEbBQ8tPU5Q1MRKxn15MkCUGeM"),
						Balance: 1512746188028,
					},
				},
			},
			affectedAddressesNoMiner: []proto.WavesAddress{
				proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
			},
		},
		crypto.MustDigestFromBase58("5wJuh2TMaqw3MysL5zUBbDoTWDtmd8WhxnhFFRHf5Ep1"): {
			snapshot: txSnapshot{
				regular: []proto.AtomicSnapshot{
					&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
						Balance: 4791869,
					},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3P3RZeHi4LTjpZdpw7kmkVSbQ84qDfrVy8G"),
						Balance: 70904380157,
					},
				},
			},
			affectedAddressesNoMiner: []proto.WavesAddress{
				proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
			},
		},
		crypto.MustDigestFromBase58("3W1Hedza1VbeB5MEvRgp4CPMpuQGLjNzWD7ZcpKYQKn9"): {
			snapshot: txSnapshot{
				regular: []proto.AtomicSnapshot{
					&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
						Balance: 4291869,
					},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3P3RZeHi4LTjpZdpw7kmkVSbQ84qDfrVy8G"),
						Balance: 70908060157,
					},
				},
			},
			affectedAddressesNoMiner: []proto.WavesAddress{
				proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
			},
		},
		crypto.MustDigestFromBase58("8jerqDBCgmGWGHRt9m1q2n8ob4i52Wug3JmwWB2v9Uj1"): {
			snapshot: txSnapshot{
				regular: []proto.AtomicSnapshot{
					&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
						Balance: 3791869,
					},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PLp1QsFxukK5nnTBYHAqjz9duWMriDkHeT"),
						Balance: 1188012800935,
					},
				},
			},
			affectedAddressesNoMiner: []proto.WavesAddress{
				proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
			},
		},
		crypto.MustDigestFromBase58("6uhpdoWPqQiBeWPk7j3ZGSMFAAuy3nz1H74FhTFcU2Ar"): {
			snapshot: txSnapshot{
				regular: []proto.AtomicSnapshot{
					&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
						Balance: 3291869,
					},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PGobRuQzBY9VbeKLaZqrcQtW26wrE9jFm7"),
						Balance: 55326020431,
					},
				},
			},
			affectedAddressesNoMiner: []proto.WavesAddress{
				proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
			},
		},
		crypto.MustDigestFromBase58("tJZUZ9NbkHDiF6V5ERhNNJtPKFds2XcxkqJbEmyNgFi"): {
			snapshot: txSnapshot{
				regular: []proto.AtomicSnapshot{
					&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
						Balance: 2791869,
					},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PLp1QsFxukK5nnTBYHAqjz9duWMriDkHeT"),
						Balance: 1192016160935,
					},
				},
			},
			affectedAddressesNoMiner: []proto.WavesAddress{
				proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
			},
		},
		crypto.MustDigestFromBase58("Ah1BN8G4aqT7iMK6YDh7sMKvDos6u9o7q1AhwXKRNhUa"): {
			snapshot: txSnapshot{
				regular: []proto.AtomicSnapshot{
					&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
						Balance: 2291869,
					},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PLp1QsFxukK5nnTBYHAqjz9duWMriDkHeT"),
						Balance: 1192016360935,
					},
				},
			},
			affectedAddressesNoMiner: []proto.WavesAddress{
				proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
			},
		},
		crypto.MustDigestFromBase58("BKSnQp7Ui8HSu3Nmm7KZTGa4PJD3GFFgpJxTZg8RqmCR"): {
			snapshot: txSnapshot{
				regular: []proto.AtomicSnapshot{
					&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
						Balance: 1791869,
					},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PLp1QsFxukK5nnTBYHAqjz9duWMriDkHeT"),
						Balance: 1192022080935,
					},
				},
			},
			affectedAddressesNoMiner: []proto.WavesAddress{
				proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
			},
		},
		crypto.MustDigestFromBase58("BSL1V4aMDiyWPieA5FkKQZAEzzGMoLSr2dMpiJqsg2mL"): {
			snapshot: txSnapshot{
				regular: []proto.AtomicSnapshot{
					&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
						Balance: 1291869,
					},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PQ9hZ36dyXGcqabcrHXsjP9PaQMqy69yeE"),
						Balance: 83127500000,
					},
				},
			},
			affectedAddressesNoMiner: []proto.WavesAddress{
				proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
			},
		},
		crypto.MustDigestFromBase58("2hjKCk2YQKXfmxMwq3SbaW9jTCiNb6paMbJzV2ksN9UC"): {
			snapshot: txSnapshot{
				regular: []proto.AtomicSnapshot{
					&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
						Balance: 791869,
					},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PQ9hZ36dyXGcqabcrHXsjP9PaQMqy69yeE"),
						Balance: 83127700000,
					},
				},
			},
			affectedAddressesNoMiner: []proto.WavesAddress{
				proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
			},
		},
		crypto.MustDigestFromBase58("GbT8HCcNVvpWozc29t3QSDvtjxKbg9Rzrt6wjgk8Qqhe"): {
			snapshot: txSnapshot{
				regular: []proto.AtomicSnapshot{
					&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
						Balance: 291869,
					},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PQ9hZ36dyXGcqabcrHXsjP9PaQMqy69yeE"),
						Balance: 83128300000,
					},
				},
			},
			affectedAddressesNoMiner: []proto.WavesAddress{
				proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
			},
		},
		crypto.MustDigestFromBase58("H5yZ8TaKDx13pqa8qTNbNhDkzaKo2qUcYos9gz7JtkrC"): {
			snapshot: txSnapshot{
				regular: []proto.AtomicSnapshot{
					&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
						Balance: 2496585,
					},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PP4nrxNnL3xRkMAaUWXnerryUDVEttAurA"),
						Balance: 31880100055,
					},
					&proto.DataEntriesSnapshot{
						Address: proto.MustAddressFromString("3PDXC37iCjkanaENuTXWhvPRQDki6yQcga2"),
						DataEntries: proto.DataEntries{
							&proto.IntegerDataEntry{
								Key:   "counter",
								Value: 4,
							},
						},
					},
				},
			},
			affectedAddressesNoMiner: []proto.WavesAddress{
				proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
				proto.MustAddressFromString("3PDXC37iCjkanaENuTXWhvPRQDki6yQcga2"),
			},
		},
		crypto.MustDigestFromBase58("Whu126znDdAm4bSrGdd6JdKaGUk62Rvt7kLLG1ZA6Kh"): {
			snapshot: txSnapshot{
				regular: []proto.AtomicSnapshot{
					&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
					&proto.AssetBalanceSnapshot{
						Address: proto.MustAddressFromString("3PHJZGJkQHDSTe1J1uoHdshhDoaBhZCzjns"),
						AssetID: crypto.MustDigestFromBase58("4QMfJbtFQ6iKJLMvZ1BbE7Zqb6dho6zh2na8myzUGn1T"),
						Balance: 950000000030,
					},
					&proto.AssetBalanceSnapshot{
						Address: proto.MustAddressFromString("3P9CrEBAmFGg9HYG1ktCWt2t1aNuF5aZt4W"),
						AssetID: crypto.MustDigestFromBase58("4QMfJbtFQ6iKJLMvZ1BbE7Zqb6dho6zh2na8myzUGn1T"),
						Balance: 4999999970,
					},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3P9CrEBAmFGg9HYG1ktCWt2t1aNuF5aZt4W"),
						Balance: 600000,
					},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3P2HNUd5VUPLMQkJmctTPEeeHumiPN2GkTb"),
						Balance: 20156591858,
					},
					&proto.FilledVolumeFeeSnapshot{
						OrderID:      crypto.MustDigestFromBase58("GUg4hL35o9thRdQR4WgsnqCTFoLRsFseX73xziaD6C5s"),
						FilledVolume: 10,
						FilledFee:    300000,
					},
					&proto.FilledVolumeFeeSnapshot{
						OrderID:      crypto.MustDigestFromBase58("9g8PpNxFvBLzFaH1atrssyAJJAc9ajGHM2x6BNadCDyk"),
						FilledVolume: 10,
						FilledFee:    300000,
					},
				},
			},
			affectedAddressesNoMiner: []proto.WavesAddress{
				proto.MustAddressFromString("3PHJZGJkQHDSTe1J1uoHdshhDoaBhZCzjns"),
				proto.MustAddressFromString("3P9CrEBAmFGg9HYG1ktCWt2t1aNuF5aZt4W"),
			},
		},
		crypto.MustDigestFromBase58("6zuQB4ydmTFvKDKKHFAYMkMqY6hn13R8V6CkMy2zpgD2"): {
			snapshot: txSnapshot{
				regular: []proto.AtomicSnapshot{
					&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
						Balance: 2296585,
					},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PFyoZZiRDg92kd25VNoDbRiqtTpj7kCKL1"),
						Balance: 2449802187416,
					},
				},
			},
			affectedAddressesNoMiner: []proto.WavesAddress{
				proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
			},
		},
		crypto.MustDigestFromBase58("5U9QQ2dwQ7iEAUQFPPm74K7ZT9wpzNMaUDPDvkFpWwsU"): {
			snapshot: txSnapshot{
				regular: []proto.AtomicSnapshot{
					&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
						Balance: 2096585,
					},
					&proto.WavesBalanceSnapshot{
						Address: proto.MustAddressFromString("3PLp1QsFxukK5nnTBYHAqjz9duWMriDkHeT"),
						Balance: 1358631653669,
					},
				},
			},
			affectedAddressesNoMiner: []proto.WavesAddress{
				proto.MustAddressFromString("3PNR77UH13UYMCWsXakafUVugvfZmhtvrhT"),
			},
		},
	}
}
