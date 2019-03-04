package internal

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"math"
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestPeerKeyBinaryRoundTrip(t *testing.T) {
	tests := []struct {
		ip    net.IP
		nonce uint64
	}{
		{net.IPv4(127, 0, 0, 1), 1234567890},
		{net.IPv4(8, 8, 8, 8), 0},
		{net.IPv4(1, 2, 3, 4), math.MaxUint64},
	}

	for _, tc := range tests {
		k := peerKey{ip: tc.ip, nonce: tc.nonce}
		b := k.bytes()
		var ak peerKey
		if err := ak.fromByte(b); assert.NoError(t, err) {
			assert.Equal(t, k.ip.To4(), ak.ip.To4())
			assert.Equal(t, k.nonce, ak.nonce)
		}
	}
}

func TestPeerInfoBinaryRoundTrip(t *testing.T) {
	tests := []struct {
		port    uint16
		name    string
		version string
		last    uint64
	}{
		{12345, "super node", "1.2.3", 1234567890},
		{0, "", "0.0.0", 1234567890},
	}
	for _, tc := range tests {
		v, err := proto.NewVersionFromString(tc.version)
		require.NoError(t, err)
		i := peerInfo{port: tc.port, name: tc.name, version: *v, last: tc.last}
		b := i.bytes()
		var ai peerInfo
		if err := ai.fromBytes(b); assert.NoError(t, err) {
			assert.Equal(t, tc.port, ai.port)
			assert.Equal(t, tc.name, ai.name)
			assert.Equal(t, *v, ai.version)
			assert.Equal(t, tc.last, ai.last)
		}
	}
}

func TestOneFork(t *testing.T) {
	db, closeDB := openDB(t, "fd-1-fork")
	defer closeDB()

	peer1 := PeerDesignation{Address: net.IPv4(1, 2, 3, 4).To4(), Nonce: 12345}
	peer2 := PeerDesignation{Address: net.IPv4(5, 6, 7, 8).To4(), Nonce: 67890}

	gs, err := crypto.NewSignatureFromBase58("FSH8eAAzZNqnG8xgTZtz5xuLqXySsXgAjmFEC25hXMbEufiGjqWPnGCZFt6gLiVLJny16ipxRNAkkzjjhqTjBE2")
	require.NoError(t, err)
	gb := proto.Block{BlockHeader: proto.BlockHeader{Parent: zeroSignature, BlockSignature: gs}}
	bs2, err := crypto.NewSignatureFromBase58("62ruZoatk3Wvs1pkWH1VB2utacPSyYdCfdAiMaYygJn6jUFGyGVg9F5i1SqDjimJvGhi8FhyT7LuRQusbHRXMnjp")
	require.NoError(t, err)
	b2 := proto.Block{BlockHeader: proto.BlockHeader{Parent: gs, BlockSignature: bs2}}
	bs3, err := crypto.NewSignatureFromBase58("54dS4KPRoKzccHv6sXYPaDNnxLL9n91LiQkDA6BkUi9tc7SJaUAXiaWZpsjrnTDYESDuqjtxjsqd34D6acyAMw7a")
	require.NoError(t, err)
	b3 := proto.Block{BlockHeader: proto.BlockHeader{Parent: bs2, BlockSignature: bs3}}
	bs4, err := crypto.NewSignatureFromBase58("2y2S3cBtcECKWPbREL3W4zoQz6nM1sHPyYggC926dfYPhADykDHHP4wPxQZHJsJEPRnbhBKDcpUcsvnYYvixMqFh")
	require.NoError(t, err)
	b4 := proto.Block{BlockHeader: proto.BlockHeader{Parent: bs3, BlockSignature: bs4}}

	storage := storage{db: db, genesis: gs}

	if err = storage.handleBlock(gb, peer1); assert.NoError(t, err) {
		if fs, err := storage.parentedForks(); assert.NoError(t, err) {
			assert.Equal(t, 1, len(fs))
			assert.Equal(t, 1, len(fs[0].Peers))
			assert.ElementsMatch(t, []PeerForkInfo{{peer1, 0}}, fs[0].Peers)
			assert.True(t, fs[0].Longest)
			assert.Equal(t, 1, fs[0].Length)
			assert.Equal(t, 1, fs[0].Height)
			assert.Equal(t, gs, fs[0].HeadBlock)
			assert.Equal(t, gs, fs[0].CommonBlock)
		}
	}
	if err = storage.handleBlock(b2, peer1); assert.NoError(t, err) {
		if fs, err := storage.parentedForks(); assert.NoError(t, err) {
			assert.Equal(t, 1, len(fs))
			assert.Equal(t, 1, len(fs[0].Peers))
			assert.ElementsMatch(t, []PeerForkInfo{{peer1, 0}}, fs[0].Peers)
			assert.True(t, fs[0].Longest)
			assert.Equal(t, 2, fs[0].Length)
			assert.Equal(t, 2, fs[0].Height)
			assert.Equal(t, bs2, fs[0].HeadBlock)
			assert.Equal(t, gs, fs[0].CommonBlock)
		}
	}
	if err = storage.handleBlock(b3, peer1); assert.NoError(t, err) {
		if fs, err := storage.parentedForks(); assert.NoError(t, err) {
			assert.Equal(t, 1, len(fs))
			assert.Equal(t, 1, len(fs[0].Peers))
			assert.ElementsMatch(t, []PeerForkInfo{{peer1, 0}}, fs[0].Peers)
			assert.True(t, fs[0].Longest)
			assert.Equal(t, 3, fs[0].Length)
			assert.Equal(t, 3, fs[0].Height)
			assert.Equal(t, bs3, fs[0].HeadBlock)
			assert.Equal(t, gs, fs[0].CommonBlock)
		}
	}
	if err = storage.handleBlock(b4, peer1); assert.NoError(t, err) {
		if fs, err := storage.parentedForks(); assert.NoError(t, err) {
			assert.Equal(t, 1, len(fs))
			assert.Equal(t, 1, len(fs[0].Peers))
			assert.ElementsMatch(t, []PeerForkInfo{{peer1, 0}}, fs[0].Peers)
			assert.True(t, fs[0].Longest)
			assert.Equal(t, 4, fs[0].Length)
			assert.Equal(t, 4, fs[0].Height)
			assert.Equal(t, bs4, fs[0].HeadBlock)
			assert.Equal(t, gs, fs[0].CommonBlock)
		}
	}
	if err = storage.handleBlock(gb, peer2); assert.NoError(t, err) {
		if fs, err := storage.parentedForks(); assert.NoError(t, err) {
			assert.Equal(t, 1, len(fs))
			assert.Equal(t, 2, len(fs[0].Peers))
			assert.ElementsMatch(t, []PeerForkInfo{{peer1, 0}, {peer2, 3}}, fs[0].Peers)
			assert.True(t, fs[0].Longest)
			assert.Equal(t, 4, fs[0].Length)
			assert.Equal(t, 4, fs[0].Height)
			assert.Equal(t, bs4, fs[0].HeadBlock)
			assert.Equal(t, gs, fs[0].CommonBlock)
		}
	}
	if err = storage.handleBlock(b2, peer2); assert.NoError(t, err) {
		if fs, err := storage.parentedForks(); assert.NoError(t, err) {
			assert.Equal(t, 1, len(fs))
			assert.Equal(t, 2, len(fs[0].Peers))
			assert.ElementsMatch(t, []PeerForkInfo{{peer1, 0}, {peer2, 2}}, fs[0].Peers)
			assert.True(t, fs[0].Longest)
			assert.Equal(t, 4, fs[0].Length)
			assert.Equal(t, 4, fs[0].Height)
			assert.Equal(t, bs4, fs[0].HeadBlock)
			assert.Equal(t, gs, fs[0].CommonBlock)
		}
	}
	if err = storage.handleBlock(b3, peer2); assert.NoError(t, err) {
		if fs, err := storage.parentedForks(); assert.NoError(t, err) {
			assert.Equal(t, 1, len(fs))
			assert.Equal(t, 2, len(fs[0].Peers))
			assert.ElementsMatch(t, []PeerForkInfo{{peer1, 0}, {peer2, 1}}, fs[0].Peers)
			assert.True(t, fs[0].Longest)
			assert.Equal(t, 4, fs[0].Length)
			assert.Equal(t, 4, fs[0].Height)
			assert.Equal(t, bs4, fs[0].HeadBlock)
			assert.Equal(t, gs, fs[0].CommonBlock)
		}
	}
}

func TestTwoForks(t *testing.T) {
	db, closeDB := openDB(t, "fd-2-fork")
	defer closeDB()

	peer1 := PeerDesignation{Address: net.IPv4(1, 2, 3, 4).To4(), Nonce: 12345}
	peer2 := PeerDesignation{Address: net.IPv4(5, 6, 7, 8).To4(), Nonce: 67890}

	gs, err := crypto.NewSignatureFromBase58("FSH8eAAzZNqnG8xgTZtz5xuLqXySsXgAjmFEC25hXMbEufiGjqWPnGCZFt6gLiVLJny16ipxRNAkkzjjhqTjBE2")
	require.NoError(t, err)
	gb := proto.Block{BlockHeader: proto.BlockHeader{Parent: zeroSignature, BlockSignature: gs}}
	bs2, err := crypto.NewSignatureFromBase58("62ruZoatk3Wvs1pkWH1VB2utacPSyYdCfdAiMaYygJn6jUFGyGVg9F5i1SqDjimJvGhi8FhyT7LuRQusbHRXMnjp")
	require.NoError(t, err)
	b2 := proto.Block{BlockHeader: proto.BlockHeader{Parent: gs, BlockSignature: bs2}}
	bs31, err := crypto.NewSignatureFromBase58("54dS4KPRoKzccHv6sXYPaDNnxLL9n91LiQkDA6BkUi9tc7SJaUAXiaWZpsjrnTDYESDuqjtxjsqd34D6acyAMw7a")
	require.NoError(t, err)
	b31 := proto.Block{BlockHeader: proto.BlockHeader{Parent: bs2, BlockSignature: bs31}}
	bs41, err := crypto.NewSignatureFromBase58("2y2S3cBtcECKWPbREL3W4zoQz6nM1sHPyYggC926dfYPhADykDHHP4wPxQZHJsJEPRnbhBKDcpUcsvnYYvixMqFh")
	require.NoError(t, err)
	b41 := proto.Block{BlockHeader: proto.BlockHeader{Parent: bs31, BlockSignature: bs41}}
	bs32, err := crypto.NewSignatureFromBase58("3w11ByGnPqjY1t3xF867JH1QWpQfJN9X532jXBMbfBhRa48itrb5QfLQGbxKjTyvogZ29yh3xvGCQoeFgqfoR2Xs")
	require.NoError(t, err)
	b32 := proto.Block{BlockHeader: proto.BlockHeader{Parent: bs2, BlockSignature: bs32}}
	//bs42, err := crypto.NewSignatureFromBase58("4dqeSQwy63YM5krVJV96ypdVJCjTrhrHLZBH8d38fvNAQhLfLWJ4JTk31tDxRQbgWL2QqnLqGHenPZwGwmE4WAku")
	//require.NoError(t, err)
	//b42 := proto.Block{BlockHeader: proto.BlockHeader{Parent: bs32, BlockSignature: bs42}}

	storage := storage{db: db, genesis: gs}

	if err = storage.handleBlock(gb, peer1); assert.NoError(t, err) {
		if fs, err := storage.parentedForks(); assert.NoError(t, err) {
			assert.Equal(t, 1, len(fs))
			assert.Equal(t, 1, len(fs[0].Peers))
			assert.ElementsMatch(t, []PeerForkInfo{{peer1, 0}}, fs[0].Peers)
			assert.True(t, fs[0].Longest)
			assert.Equal(t, 1, fs[0].Length)
			assert.Equal(t, 1, fs[0].Height)
			assert.Equal(t, gs, fs[0].HeadBlock)
			assert.Equal(t, gs, fs[0].CommonBlock)
		}
	}
	if err = storage.handleBlock(b2, peer1); assert.NoError(t, err) {
		if fs, err := storage.parentedForks(); assert.NoError(t, err) {
			assert.Equal(t, 1, len(fs))
			assert.Equal(t, 1, len(fs[0].Peers))
			assert.ElementsMatch(t, []PeerForkInfo{{peer1, 0}}, fs[0].Peers)
			assert.True(t, fs[0].Longest)
			assert.Equal(t, 2, fs[0].Length)
			assert.Equal(t, 2, fs[0].Height)
			assert.Equal(t, bs2, fs[0].HeadBlock)
			assert.Equal(t, gs, fs[0].CommonBlock)
		}
	}
	if err = storage.handleBlock(b31, peer1); assert.NoError(t, err) {
		if fs, err := storage.parentedForks(); assert.NoError(t, err) {
			assert.Equal(t, 1, len(fs))
			assert.Equal(t, 1, len(fs[0].Peers))
			assert.ElementsMatch(t, []PeerForkInfo{{peer1, 0}}, fs[0].Peers)
			assert.True(t, fs[0].Longest)
			assert.Equal(t, 3, fs[0].Length)
			assert.Equal(t, 3, fs[0].Height)
			assert.Equal(t, bs31, fs[0].HeadBlock)
			assert.Equal(t, gs, fs[0].CommonBlock)
		}
	}
	if err = storage.handleBlock(b41, peer1); assert.NoError(t, err) {
		if fs, err := storage.parentedForks(); assert.NoError(t, err) {
			assert.Equal(t, 1, len(fs))
			assert.Equal(t, 1, len(fs[0].Peers))
			assert.ElementsMatch(t, []PeerForkInfo{{peer1, 0}}, fs[0].Peers)
			assert.True(t, fs[0].Longest)
			assert.Equal(t, 4, fs[0].Length)
			assert.Equal(t, 4, fs[0].Height)
			assert.Equal(t, bs41, fs[0].HeadBlock)
			assert.Equal(t, gs, fs[0].CommonBlock)
		}
	}
	if err = storage.handleBlock(gb, peer2); assert.NoError(t, err) {
		if fs, err := storage.parentedForks(); assert.NoError(t, err) {
			assert.Equal(t, 1, len(fs))
			assert.Equal(t, 2, len(fs[0].Peers))
			assert.ElementsMatch(t, []PeerForkInfo{{peer1, 0}, {peer2, 3}}, fs[0].Peers)
			assert.True(t, fs[0].Longest)
			assert.Equal(t, 4, fs[0].Length)
			assert.Equal(t, 4, fs[0].Height)
			assert.Equal(t, bs41, fs[0].HeadBlock)
			assert.Equal(t, gs, fs[0].CommonBlock)
		}
	}
	if err = storage.handleBlock(b2, peer2); assert.NoError(t, err) {
		if fs, err := storage.parentedForks(); assert.NoError(t, err) {
			assert.Equal(t, 1, len(fs))
			assert.Equal(t, 2, len(fs[0].Peers))
			assert.ElementsMatch(t, []PeerForkInfo{{peer1, 0}, {peer2, 2}}, fs[0].Peers)
			assert.True(t, fs[0].Longest)
			assert.Equal(t, 4, fs[0].Length)
			assert.Equal(t, 4, fs[0].Height)
			assert.Equal(t, bs41, fs[0].HeadBlock)
			assert.Equal(t, gs, fs[0].CommonBlock)
		}
	}
	if err = storage.handleBlock(b32, peer2); assert.NoError(t, err) {
		if fs, err := storage.parentedForks(); assert.NoError(t, err) {
			assert.Equal(t, 2, len(fs))
			assert.Equal(t, 1, len(fs[0].Peers))
			assert.Equal(t, 1, len(fs[1].Peers))
			assert.ElementsMatch(t, []PeerForkInfo{{peer1, 0}}, fs[0].Peers)
			assert.ElementsMatch(t, []PeerForkInfo{{peer2, 0}}, fs[1].Peers)
			assert.True(t, fs[0].Longest)
			assert.Equal(t, 4, fs[0].Length)
			assert.Equal(t, 4, fs[0].Height)
			assert.Equal(t, bs41, fs[0].HeadBlock)
			assert.Equal(t, gs, fs[0].CommonBlock)
			assert.False(t, fs[1].Longest)
			assert.Equal(t, 1, fs[1].Length)
			assert.Equal(t, 3, fs[1].Height)
			assert.Equal(t, bs32, fs[1].HeadBlock)
			assert.Equal(t, bs2, fs[1].CommonBlock)
		}
	}
}

func TestMultipleForksAndSwitching(t *testing.T) {
	db, closeDB := openDB(t, "fd-3-fork")
	defer closeDB()

	peer1 := PeerDesignation{Address: net.IPv4(1, 1, 1, 1).To4(), Nonce: 11111}
	peer2 := PeerDesignation{Address: net.IPv4(2, 2, 2, 2).To4(), Nonce: 22222}
	peer3 := PeerDesignation{Address: net.IPv4(3, 3, 3, 3).To4(), Nonce: 33333}

	gs, err := crypto.NewSignatureFromBase58("FSH8eAAzZNqnG8xgTZtz5xuLqXySsXgAjmFEC25hXMbEufiGjqWPnGCZFt6gLiVLJny16ipxRNAkkzjjhqTjBE2")
	require.NoError(t, err)
	gb := proto.Block{BlockHeader: proto.BlockHeader{Parent: zeroSignature, BlockSignature: gs}}
	bs2, err := crypto.NewSignatureFromBase58("62ruZoatk3Wvs1pkWH1VB2utacPSyYdCfdAiMaYygJn6jUFGyGVg9F5i1SqDjimJvGhi8FhyT7LuRQusbHRXMnjp")
	require.NoError(t, err)
	b2 := proto.Block{BlockHeader: proto.BlockHeader{Parent: gs, BlockSignature: bs2}}
	bs31, err := crypto.NewSignatureFromBase58("54dS4KPRoKzccHv6sXYPaDNnxLL9n91LiQkDA6BkUi9tc7SJaUAXiaWZpsjrnTDYESDuqjtxjsqd34D6acyAMw7a")
	require.NoError(t, err)
	b31 := proto.Block{BlockHeader: proto.BlockHeader{Parent: bs2, BlockSignature: bs31}}
	bs41, err := crypto.NewSignatureFromBase58("2y2S3cBtcECKWPbREL3W4zoQz6nM1sHPyYggC926dfYPhADykDHHP4wPxQZHJsJEPRnbhBKDcpUcsvnYYvixMqFh")
	require.NoError(t, err)
	b41 := proto.Block{BlockHeader: proto.BlockHeader{Parent: bs31, BlockSignature: bs41}}
	bs32, err := crypto.NewSignatureFromBase58("3w11ByGnPqjY1t3xF867JH1QWpQfJN9X532jXBMbfBhRa48itrb5QfLQGbxKjTyvogZ29yh3xvGCQoeFgqfoR2Xs")
	require.NoError(t, err)
	b32 := proto.Block{BlockHeader: proto.BlockHeader{Parent: bs2, BlockSignature: bs32}}
	bs33, err := crypto.NewSignatureFromBase58("SxqNDcFRWhCaErKA2w74zE3eXFp9uEuosd8HTGd7pozyZFMuCumyppFZuP7UNrtzEvm713ZmKjzEiNtNoit7mAC")
	require.NoError(t, err)
	b33 := proto.Block{BlockHeader: proto.BlockHeader{Parent: bs2, BlockSignature: bs33}}
	bs42, err := crypto.NewSignatureFromBase58("4dqeSQwy63YM5krVJV96ypdVJCjTrhrHLZBH8d38fvNAQhLfLWJ4JTk31tDxRQbgWL2QqnLqGHenPZwGwmE4WAku")
	require.NoError(t, err)
	b42 := proto.Block{BlockHeader: proto.BlockHeader{Parent: bs32, BlockSignature: bs42}}
	bs51, err := crypto.NewSignatureFromBase58("4qTNZkFRBw1Pt2bz2x3WqVivTkd3h1fNd3riCgeGmg9s4vvF3Y1zbHqHdYoj7XdtCfZHhdPJ4ZRuiCUYqD1MVTHb")
	require.NoError(t, err)
	b51 := proto.Block{BlockHeader: proto.BlockHeader{Parent: bs41, BlockSignature: bs51}}

	storage := storage{db: db, genesis: gs}

	if err = storage.handleBlock(gb, peer1); assert.NoError(t, err) {
		if fs, err := storage.parentedForks(); assert.NoError(t, err) {
			assert.Equal(t, 1, len(fs))
			assert.Equal(t, 1, len(fs[0].Peers))
			assert.ElementsMatch(t, []PeerForkInfo{{peer1, 0}}, fs[0].Peers)
			assert.True(t, fs[0].Longest)
			assert.Equal(t, 1, fs[0].Length)
			assert.Equal(t, 1, fs[0].Height)
			assert.Equal(t, gs, fs[0].HeadBlock)
			assert.Equal(t, gs, fs[0].CommonBlock)
		}
	}
	if err = storage.handleBlock(gb, peer2); assert.NoError(t, err) {
		if fs, err := storage.parentedForks(); assert.NoError(t, err) {
			assert.Equal(t, 1, len(fs))
			assert.Equal(t, 2, len(fs[0].Peers))
			assert.ElementsMatch(t, []PeerForkInfo{{peer1, 0}, {peer2, 0}}, fs[0].Peers)
			assert.True(t, fs[0].Longest)
			assert.Equal(t, 1, fs[0].Length)
			assert.Equal(t, 1, fs[0].Height)
			assert.Equal(t, gs, fs[0].HeadBlock)
			assert.Equal(t, gs, fs[0].CommonBlock)
		}
	}
	if err = storage.handleBlock(gb, peer3); assert.NoError(t, err) {
		if fs, err := storage.parentedForks(); assert.NoError(t, err) {
			assert.Equal(t, 1, len(fs))
			assert.Equal(t, 3, len(fs[0].Peers))
			assert.ElementsMatch(t, []PeerForkInfo{{peer1, 0}, {peer2, 0}, {peer3, 0}}, fs[0].Peers)
			assert.True(t, fs[0].Longest)
			assert.Equal(t, 1, fs[0].Length)
			assert.Equal(t, 1, fs[0].Height)
			assert.Equal(t, gs, fs[0].HeadBlock)
			assert.Equal(t, gs, fs[0].CommonBlock)
		}
	}
	if err = storage.handleBlock(b2, peer1); assert.NoError(t, err) {
		if fs, err := storage.parentedForks(); assert.NoError(t, err) {
			assert.Equal(t, 1, len(fs))
			assert.Equal(t, 3, len(fs[0].Peers))
			assert.ElementsMatch(t, []PeerForkInfo{{peer1, 0}, {peer2, 1}, {peer3, 1}}, fs[0].Peers)
			assert.True(t, fs[0].Longest)
			assert.Equal(t, 2, fs[0].Length)
			assert.Equal(t, 2, fs[0].Height)
			assert.Equal(t, bs2, fs[0].HeadBlock)
			assert.Equal(t, gs, fs[0].CommonBlock)
		}
	}
	if err = storage.handleBlock(b31, peer1); assert.NoError(t, err) {
		if fs, err := storage.parentedForks(); assert.NoError(t, err) {
			assert.Equal(t, 1, len(fs))
			assert.Equal(t, 3, len(fs[0].Peers))
			assert.ElementsMatch(t, []PeerForkInfo{{peer1, 0}, {peer2, 2}, {peer3, 2}}, fs[0].Peers)
			assert.True(t, fs[0].Longest)
			assert.Equal(t, 3, fs[0].Length)
			assert.Equal(t, 3, fs[0].Height)
			assert.Equal(t, bs31, fs[0].HeadBlock)
			assert.Equal(t, gs, fs[0].CommonBlock)
		}
	}
	if err = storage.handleBlock(b2, peer3); assert.NoError(t, err) {
		if fs, err := storage.parentedForks(); assert.NoError(t, err) {
			assert.Equal(t, 1, len(fs))
			assert.Equal(t, 3, len(fs[0].Peers))
			assert.ElementsMatch(t, []PeerForkInfo{{peer1, 0}, {peer2, 2}, {peer3, 1}}, fs[0].Peers)
			assert.True(t, fs[0].Longest)
			assert.Equal(t, 3, fs[0].Length)
			assert.Equal(t, 3, fs[0].Height)
			assert.Equal(t, bs31, fs[0].HeadBlock)
			assert.Equal(t, gs, fs[0].CommonBlock)
		}
	}
	if err = storage.handleBlock(b2, peer2); assert.NoError(t, err) {
		if fs, err := storage.parentedForks(); assert.NoError(t, err) {
			assert.Equal(t, 1, len(fs))
			assert.Equal(t, 3, len(fs[0].Peers))
			assert.ElementsMatch(t, []PeerForkInfo{{peer1, 0}, {peer2, 1}, {peer3, 1}}, fs[0].Peers)
			assert.True(t, fs[0].Longest)
			assert.Equal(t, 3, fs[0].Length)
			assert.Equal(t, 3, fs[0].Height)
			assert.Equal(t, bs31, fs[0].HeadBlock)
			assert.Equal(t, gs, fs[0].CommonBlock)
		}
	}
	if err = storage.handleBlock(b32, peer2); assert.NoError(t, err) {
		if fs, err := storage.parentedForks(); assert.NoError(t, err) {
			assert.Equal(t, 2, len(fs))
			assert.Equal(t, 2, len(fs[0].Peers))
			assert.Equal(t, 1, len(fs[1].Peers))
			assert.ElementsMatch(t, []PeerForkInfo{{peer1, 0}, {peer3, 1}}, fs[0].Peers)
			assert.ElementsMatch(t, []PeerForkInfo{{peer2, 0}}, fs[1].Peers)
			assert.True(t, fs[0].Longest)
			assert.False(t, fs[1].Longest)
			assert.Equal(t, 3, fs[0].Length)
			assert.Equal(t, 3, fs[0].Height)
			assert.Equal(t, 1, fs[1].Length)
			assert.Equal(t, 3, fs[1].Height)
			assert.Equal(t, bs31, fs[0].HeadBlock)
			assert.Equal(t, gs, fs[0].CommonBlock)
			assert.Equal(t, bs32, fs[1].HeadBlock)
			assert.Equal(t, bs2, fs[1].CommonBlock)
		}
	}
	if err = storage.handleBlock(b41, peer1); assert.NoError(t, err) {
		if fs, err := storage.parentedForks(); assert.NoError(t, err) {
			assert.Equal(t, 2, len(fs))
			assert.Equal(t, 2, len(fs[0].Peers))
			assert.Equal(t, 1, len(fs[1].Peers))
			assert.ElementsMatch(t, []PeerForkInfo{{peer1, 0}, {peer3, 2}}, fs[0].Peers)
			assert.ElementsMatch(t, []PeerForkInfo{{peer2, 0}}, fs[1].Peers)
			assert.True(t, fs[0].Longest)
			assert.False(t, fs[1].Longest)
			assert.Equal(t, 4, fs[0].Length)
			assert.Equal(t, 4, fs[0].Height)
			assert.Equal(t, 1, fs[1].Length)
			assert.Equal(t, 3, fs[1].Height)
			assert.Equal(t, bs41, fs[0].HeadBlock)
			assert.Equal(t, gs, fs[0].CommonBlock)
			assert.Equal(t, bs32, fs[1].HeadBlock)
			assert.Equal(t, bs2, fs[1].CommonBlock)
		}
	}
	if err = storage.handleBlock(b33, peer3); assert.NoError(t, err) {
		if fs, err := storage.parentedForks(); assert.NoError(t, err) {
			assert.Equal(t, 3, len(fs))
			assert.Equal(t, 1, len(fs[0].Peers))
			assert.Equal(t, 1, len(fs[1].Peers))
			assert.Equal(t, 1, len(fs[2].Peers))
			assert.ElementsMatch(t, []PeerForkInfo{{peer1, 0}}, fs[0].Peers)
			assert.True(t, fs[0].Longest)
			assert.False(t, fs[1].Longest)
			assert.False(t, fs[2].Longest)
			assert.Equal(t, 4, fs[0].Length)
			assert.Equal(t, 4, fs[0].Height)
			assert.Equal(t, 1, fs[1].Length)
			assert.Equal(t, 3, fs[1].Height)
			assert.Equal(t, 1, fs[2].Length)
			assert.Equal(t, 3, fs[2].Height)
			assert.Equal(t, bs41, fs[0].HeadBlock)
			assert.Equal(t, gs, fs[0].CommonBlock)
			if bs32 == fs[1].HeadBlock {
				assert.Equal(t, bs32, fs[1].HeadBlock)
				assert.Equal(t, bs2, fs[1].CommonBlock)
				assert.Equal(t, bs33, fs[2].HeadBlock)
				assert.Equal(t, bs2, fs[2].CommonBlock)
			} else {
				assert.Equal(t, bs32, fs[2].HeadBlock)
				assert.Equal(t, bs2, fs[2].CommonBlock)
				assert.Equal(t, bs33, fs[1].HeadBlock)
				assert.Equal(t, bs2, fs[1].CommonBlock)
			}
		}
	}
	if err = storage.handleBlock(b42, peer2); assert.NoError(t, err) {
		if fs, err := storage.parentedForks(); assert.NoError(t, err) {
			assert.Equal(t, 3, len(fs))
			assert.Equal(t, 1, len(fs[0].Peers))
			assert.Equal(t, 1, len(fs[1].Peers))
			assert.Equal(t, 1, len(fs[2].Peers))
			if fs[0].Peers[0].Peer.Address.Equal(peer3.Address) {
				assert.Fail(t, "unexpected peer in longest fork")
			}
			assert.True(t, fs[0].Longest)
			assert.False(t, fs[1].Longest)
			assert.False(t, fs[2].Longest)
			assert.Equal(t, 4, fs[0].Length)
			assert.Equal(t, 4, fs[0].Height)
			assert.Equal(t, bs41, fs[0].HeadBlock)
			assert.Equal(t, gs, fs[0].CommonBlock)
			assert.Equal(t, 2, fs[1].Length)
			assert.Equal(t, 4, fs[1].Height)
			assert.Equal(t, bs42, fs[1].HeadBlock)
			assert.Equal(t, bs2, fs[1].CommonBlock)
			assert.Equal(t, 1, fs[2].Length)
			assert.Equal(t, 3, fs[2].Height)
			assert.Equal(t, bs33, fs[2].HeadBlock)
			assert.Equal(t, bs2, fs[2].CommonBlock)
		}
	}
	if err = storage.handleBlock(b31, peer3); assert.NoError(t, err) {
		if fs, err := storage.parentedForks(); assert.NoError(t, err) {
			assert.Equal(t, 2, len(fs))
			assert.Equal(t, 2, len(fs[0].Peers))
			assert.Equal(t, 1, len(fs[1].Peers))
			assert.ElementsMatch(t, []PeerForkInfo{{peer1, 0}, {peer3, 1}}, fs[0].Peers)
			assert.ElementsMatch(t, []PeerForkInfo{{peer2, 0}}, fs[1].Peers)
			assert.True(t, fs[0].Longest)
			assert.False(t, fs[1].Longest)
			assert.Equal(t, 4, fs[0].Length)
			assert.Equal(t, 4, fs[0].Height)
			assert.Equal(t, 2, fs[1].Length)
			assert.Equal(t, 4, fs[1].Height)
			assert.Equal(t, bs41, fs[0].HeadBlock)
			assert.Equal(t, gs, fs[0].CommonBlock)
			assert.Equal(t, bs42, fs[1].HeadBlock)
			assert.Equal(t, bs2, fs[1].CommonBlock)
		}
	}
	if err = storage.handleBlock(b41, peer3); assert.NoError(t, err) {
		if fs, err := storage.parentedForks(); assert.NoError(t, err) {
			assert.Equal(t, 2, len(fs))
			assert.Equal(t, 2, len(fs[0].Peers))
			assert.Equal(t, 1, len(fs[1].Peers))
			assert.ElementsMatch(t, []PeerForkInfo{{peer1, 0}, {peer3, 0}}, fs[0].Peers)
			assert.ElementsMatch(t, []PeerForkInfo{{peer2, 0}}, fs[1].Peers)
			assert.True(t, fs[0].Longest)
			assert.False(t, fs[1].Longest)
			assert.Equal(t, 4, fs[0].Length)
			assert.Equal(t, 4, fs[0].Height)
			assert.Equal(t, 2, fs[1].Length)
			assert.Equal(t, 4, fs[1].Height)
			assert.Equal(t, bs41, fs[0].HeadBlock)
			assert.Equal(t, gs, fs[0].CommonBlock)
			assert.Equal(t, bs42, fs[1].HeadBlock)
			assert.Equal(t, bs2, fs[1].CommonBlock)
		}
	}
	if err = storage.handleBlock(b51, peer1); assert.NoError(t, err) {
		if fs, err := storage.parentedForks(); assert.NoError(t, err) {
			assert.Equal(t, 2, len(fs))
			assert.Equal(t, 2, len(fs[0].Peers))
			assert.Equal(t, 1, len(fs[1].Peers))
			assert.ElementsMatch(t, []PeerForkInfo{{peer1, 0}, {peer3, 1}}, fs[0].Peers)
			assert.ElementsMatch(t, []PeerForkInfo{{peer2, 0}}, fs[1].Peers)
			assert.True(t, fs[0].Longest)
			assert.False(t, fs[1].Longest)
			assert.Equal(t, 5, fs[0].Length)
			assert.Equal(t, 5, fs[0].Height)
			assert.Equal(t, 2, fs[1].Length)
			assert.Equal(t, 4, fs[1].Height)
			assert.Equal(t, bs51, fs[0].HeadBlock)
			assert.Equal(t, gs, fs[0].CommonBlock)
			assert.Equal(t, bs42, fs[1].HeadBlock)
			assert.Equal(t, bs2, fs[1].CommonBlock)
		}
	}
	if err = storage.handleBlock(b51, peer3); assert.NoError(t, err) {
		if fs, err := storage.parentedForks(); assert.NoError(t, err) {
			assert.Equal(t, 2, len(fs))
			assert.Equal(t, 2, len(fs[0].Peers))
			assert.Equal(t, 1, len(fs[1].Peers))
			assert.ElementsMatch(t, []PeerForkInfo{{peer1, 0}, {peer3, 0}}, fs[0].Peers)
			assert.ElementsMatch(t, []PeerForkInfo{{peer2, 0}}, fs[1].Peers)
			assert.True(t, fs[0].Longest)
			assert.False(t, fs[1].Longest)
			assert.Equal(t, 5, fs[0].Length)
			assert.Equal(t, 5, fs[0].Height)
			assert.Equal(t, 2, fs[1].Length)
			assert.Equal(t, 4, fs[1].Height)
			assert.Equal(t, bs51, fs[0].HeadBlock)
			assert.Equal(t, gs, fs[0].CommonBlock)
			assert.Equal(t, bs42, fs[1].HeadBlock)
			assert.Equal(t, bs2, fs[1].CommonBlock)
		}
	}
}

func openDB(t *testing.T, name string) (*leveldb.DB, func()) {
	path := filepath.Join(os.TempDir(), name)
	opts := opt.Options{ErrorIfExist: true}
	db, err := leveldb.OpenFile(path, &opts)
	assert.NoError(t, err)
	return db, func() {
		err = db.Close()
		assert.NoError(t, err)
		err = os.RemoveAll(path)
		assert.NoError(t, err)
	}
}
