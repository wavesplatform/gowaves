package storage

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestMarshalUnmarshalCborFromFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_marshal_to_cbor_*.cbor")
	require.NoError(t, err)
	defer func() {
		filename := tmpFile.Name()
		assert.NoError(t, tmpFile.Close())
		require.NoError(t, os.Remove(filename))
	}()

	expected := SuspendedPeer{
		IP:                      IPFromString("13.3.4.1"),
		RestrictTimestampMillis: time.Now().UnixMilli(),
		RestrictDuration:        time.Minute * 5,
		Reason:                  "some reason",
	}

	err = marshalToCborAndSyncToFile(tmpFile.Name(), &expected)
	require.NoError(t, err)
	err = tmpFile.Sync()
	require.NoError(t, err)

	// nickeskov: check marshalling
	actual := SuspendedPeer{}
	err = cbor.NewDecoder(tmpFile).Decode(&actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)

	// nickeskov: check unmarshalling
	err = unmarshalCborFromFile(tmpFile.Name(), &actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)

	// nickeskov: check error when file not exist
	err = unmarshalCborFromFile(tmpFile.Name()+"_not_exist_", &actual)
	assert.Error(t, err)
}

func TestUnmarshalCborFromEmptyFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "peers_storage_test_*.cbor")
	require.NoError(t, err)
	defer func() {
		filename := tmpFile.Name()
		assert.NoError(t, tmpFile.Close())
		require.NoError(t, os.Remove(filename))
	}()

	dummy := SuspendedPeer{}
	err = unmarshalCborFromFile(tmpFile.Name(), &dummy)
	assert.Equal(t, io.EOF, err)
}

func TestCborMarshalUnmarshalWithEmptyData(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "*.cbor")
	require.NoError(t, err)
	defer func() {
		filename := tmpFile.Name()
		assert.NoError(t, tmpFile.Close())
		require.NoError(t, os.Remove(filename))
	}()

	dummy := suspendedPeers{}
	err = marshalToCborAndSyncToFile(tmpFile.Name(), dummy)
	require.NoError(t, err)

	err = unmarshalCborFromFile(tmpFile.Name(), &dummy)
	require.NoError(t, err)
}

type binaryStorageCborSuite struct {
	suite.Suite
	storage *CBORStorage
	now     time.Time
}

func (s *binaryStorageCborSuite) SetupTest() {
	tmpdir := s.T().TempDir()
	now := time.Now()
	storage, err := newCBORStorageInDir(tmpdir, now, peersStorageCurrentVersion)
	require.NoError(s.T(), err)

	s.storage = storage
	s.now = now
}

func (s *binaryStorageCborSuite) TearDownTest() {
	s.storage = nil
}

func TestBinaryStorageCborTestSuite(t *testing.T) {
	suite.Run(t, new(binaryStorageCborSuite))
}

func (s *binaryStorageCborSuite) TestCBORStorageKnown() {
	knownList := []KnownPeer{
		KnownPeer(proto.NewIpPortFromTcpAddr(proto.NewTCPAddrFromString("13.3.4.1:2345"))),
		KnownPeer(proto.NewIpPortFromTcpAddr(proto.NewTCPAddrFromString("3.54.1.9:1454"))),
		KnownPeer(proto.NewIpPortFromTcpAddr(proto.NewTCPAddrFromString("23.43.7.43:4234"))),
		KnownPeer(proto.NewIpPortFromTcpAddr(proto.NewTCPAddrFromString("42.54.1.6:54356"))),
	}

	setPeersTs := func(m knownPeers, list []KnownPeer, ts int64) {
		for _, v := range list {
			m[v] = ts
		}
	}

	initKnownMap := func(list []KnownPeer, ts int64) knownPeers {
		knownMap := make(knownPeers)
		setPeersTs(knownMap, list, ts)
		return knownMap
	}

	check := func(known knownPeers) {
		var unmarshalled knownPeers
		require.NoError(s.T(), unmarshalCborFromFile(s.storage.knownFilePath, &unmarshalled))
		assert.Equal(s.T(), len(known), len(unmarshalled))
		// nickeskov: check that all marshaled data saved in file
		for expectedPeer, expectedTs := range known {
			ts, in := unmarshalled[expectedPeer]
			require.True(s.T(), in)
			require.Equal(s.T(), expectedTs, ts)
		}

		// nickeskov: check that all data saved in cache
		cachedKnown := make(knownPeers)
		for _, k := range s.storage.Known(10) {
			cachedKnown[k] = 0
		}

		for k := range cachedKnown {
			_, in := unmarshalled[k]
			require.True(s.T(), in)
		}
	}

	s.Run("add and get known peers", func() {
		// nickeskov: check empty input
		require.NoError(s.T(), s.storage.AddOrUpdateKnown(nil, time.Time{}))

		knownMap := initKnownMap(knownList, 0)

		ts := time.Now()
		setPeersTs(knownMap, knownList, ts.UnixNano())
		require.NoError(s.T(), s.storage.AddOrUpdateKnown(knownList, ts))
		check(knownMap)

		// check input with same addresses and new timestamps
		newTs := time.Now()
		setPeersTs(knownMap, knownList, newTs.UnixNano())
		err := s.storage.AddOrUpdateKnown(knownList, newTs)
		require.NoError(s.T(), err)
		check(knownMap)

		// clean known peers storage to eliminate unexpected side effects
		require.NoError(s.T(), s.storage.DropKnown())
	})

	s.Run("delete and get known peers", func() {
		// nickeskov: check empty input
		require.NoError(s.T(), s.storage.DeleteKnown(nil))

		// fill known in storage
		ts := time.Now()
		knownMap := initKnownMap(knownList, ts.UnixNano())
		require.NoError(s.T(), s.storage.AddOrUpdateKnown(knownList, ts))

		// nickeskov: remove first entry
		err := s.storage.DeleteKnown(knownList[:1])
		require.NoError(s.T(), err)

		delete(knownMap, knownList[0])
		check(knownMap)
		require.NoError(s.T(), s.storage.DropKnown())

		// clean known peers storage to eliminate unexpected side effects
		require.NoError(s.T(), s.storage.DropKnown())
	})

	s.Run("unsafe sync known peers bad storage file", func() {
		defer func(knownStorageFile string) {
			require.NoError(s.T(), os.Remove(s.storage.knownFilePath))
			s.storage.knownFilePath = knownStorageFile
		}(s.storage.knownFilePath)

		badFilePath := filepath.Join(s.storage.storageDir, "test_invalid_known_storage_file")
		f, err := os.OpenFile(badFilePath, os.O_CREATE, 0100)
		require.NoError(s.T(), err)
		defer func() {
			require.NoError(s.T(), f.Chmod(0644))
			require.NoError(s.T(), f.Close())
		}()

		s.storage.knownFilePath = badFilePath
		err = s.storage.unsafeSyncKnown(nil, nil)
		require.Error(s.T(), err)
	})
}

func (s *binaryStorageCborSuite) TestCBORStorageSuspended() {
	suspendDuration := time.Minute * 5
	now := s.now.Truncate(time.Millisecond)
	suspended := []SuspendedPeer{
		{
			IP:                      IPFromString("13.3.4.1"),
			RestrictTimestampMillis: now.UnixMilli(),
			RestrictDuration:        suspendDuration,
			Reason:                  "some reason #1",
		},
		{
			IP:                      IPFromString("3.54.1.9"),
			RestrictTimestampMillis: now.UnixMilli(),
			RestrictDuration:        suspendDuration,
			Reason:                  "some reason #2",
		},
		{
			IP:                      IPFromString("23.43.7.43"),
			RestrictTimestampMillis: now.UnixMilli(),
			RestrictDuration:        suspendDuration + time.Minute*2,
			Reason:                  "some reason #3",
		},
		{
			IP:                      IPFromString("42.54.1.6"),
			RestrictTimestampMillis: now.UnixMilli(),
			RestrictDuration:        suspendDuration + time.Minute*2,
			Reason:                  "some reason #4",
		},
	}

	check := func(suspended []SuspendedPeer) {
		var unmarshalled suspendedPeers
		require.NoError(s.T(), unmarshalCborFromFile(s.storage.suspendedFilePath, &unmarshalled))
		assert.Equal(s.T(), len(suspended), len(unmarshalled))
		// nickeskov: check that all marshaled data saved in file
		for _, expected := range suspended {
			_, in := unmarshalled[expected.IP]
			require.True(s.T(), in)
		}

		// nickeskov: check that all data saved in cache
		cachedSuspended := make(suspendedPeers)
		for _, suspendedPeer := range s.storage.Suspended(now) {
			cachedSuspended[suspendedPeer.IP] = suspendedPeer
		}

		for k := range cachedSuspended {
			_, in := unmarshalled[k]
			require.True(s.T(), in)
		}
	}

	s.Run("add and get suspended peers", func() {
		// nickeskov: check empty input
		require.NoError(s.T(), s.storage.AddSuspended(nil))

		err := s.storage.AddSuspended(suspended)
		require.NoError(s.T(), err)
		check(suspended)
	})

	s.Run("ip is suspended", func() {
		for _, peer := range suspended {
			require.True(s.T(), s.storage.IsSuspendedIP(peer.IP, now))
		}
	})

	s.Run("ips is suspended", func() {
		// nickeskov: check empty input
		empty := s.storage.IsSuspendedIPs(nil, now)
		assert.Empty(s.T(), empty)

		ips := make([]IP, 0, len(suspended))
		for _, peer := range suspended {
			ips = append(ips, peer.IP)
		}
		res := s.storage.IsSuspendedIPs(ips, now.Add(suspendDuration))
		assert.False(s.T(), res[0])
		assert.False(s.T(), res[1])
		assert.True(s.T(), res[2])
		assert.True(s.T(), res[3])
	})

	s.Run("delete and get suspended peers", func() {
		defer func() {
			// nickeskov: set previous values
			require.NoError(s.T(), s.storage.AddSuspended(suspended))
		}()

		// nickeskov: check empty input
		require.NoError(s.T(), s.storage.DeleteSuspendedByIP(nil))

		// nickeskov: remove first entry
		err := s.storage.DeleteSuspendedByIP(suspended[:1])
		require.NoError(s.T(), err)
		check(suspended[1:])
	})

	s.Run("refresh suspended peers", func() {
		err := s.storage.RefreshSuspended(now.Add(suspendDuration))
		require.NoError(s.T(), err)
		check(suspended[2:])
	})

	s.Run("new cbor storage with suspended refreshing", func() {
		defer func() {
			// nickeskov: set previous values
			require.NoError(s.T(), s.storage.AddSuspended(suspended))
		}()

		newNow := now.Add(suspendDuration)
		storage, err := newCBORStorageInDir(s.storage.storageDir, newNow, peersStorageCurrentVersion)
		require.NoError(s.T(), err)
		s.storage = storage

		testMap := make(suspendedPeers)
		for _, peer := range s.storage.Suspended(newNow) {
			testMap[peer.IP] = peer
		}

		for _, peer := range suspended[2:] {
			inMapPeer, in := testMap[peer.IP]
			assert.True(s.T(), in)
			assert.Equal(s.T(), peer, inMapPeer)
		}
	})

	s.Run("unsafe sync suspended peers bad storage file", func() {
		defer func(suspendedStorageFile string) {
			require.NoError(s.T(), os.Remove(s.storage.suspendedFilePath))
			s.storage.suspendedFilePath = suspendedStorageFile
		}(s.storage.suspendedFilePath)

		badFilePath := filepath.Join(s.storage.storageDir, "test_invalid_suspended_storage_file")
		f, err := os.OpenFile(badFilePath, os.O_CREATE, 0100)
		require.NoError(s.T(), err)
		defer func() {
			require.NoError(s.T(), f.Chmod(0644))
			require.NoError(s.T(), f.Close())
		}()

		s.storage.suspendedFilePath = badFilePath
		err = s.storage.unsafeSyncRestricted(nil, nil, suspendedPeersID)
		require.Error(s.T(), err)
	})
}

func (s *binaryStorageCborSuite) TestCBORStorageBlackList() {
	blackListDuration := time.Minute * 5
	now := s.now.Truncate(time.Millisecond)
	blackList := []BlackListedPeer{
		{
			IP:                      IPFromString("13.3.4.1"),
			RestrictTimestampMillis: now.UnixMilli(),
			RestrictDuration:        blackListDuration,
			Reason:                  "some reason #1",
		},
		{
			IP:                      IPFromString("3.54.1.9"),
			RestrictTimestampMillis: now.UnixMilli(),
			RestrictDuration:        blackListDuration,
			Reason:                  "some reason #2",
		},
		{
			IP:                      IPFromString("23.43.7.43"),
			RestrictTimestampMillis: now.UnixMilli(),
			RestrictDuration:        blackListDuration + time.Minute*2,
			Reason:                  "some reason #3",
		},
		{
			IP:                      IPFromString("42.54.1.6"),
			RestrictTimestampMillis: now.UnixMilli(),
			RestrictDuration:        blackListDuration + time.Minute*2,
			Reason:                  "some reason #4",
		},
	}

	check := func(blackList []BlackListedPeer) {
		var unmarshalled blackListedPeers
		require.NoError(s.T(), unmarshalCborFromFile(s.storage.blackListFilePath, &unmarshalled))
		assert.Equal(s.T(), len(blackList), len(unmarshalled))

		for _, expected := range blackList {
			_, in := unmarshalled[expected.IP]
			require.True(s.T(), in)
		}

		cachedBlackList := make(blackListedPeers)
		for _, blackListedPeer := range s.storage.BlackList(now) {
			cachedBlackList[blackListedPeer.IP] = blackListedPeer
		}

		for k := range cachedBlackList {
			_, in := unmarshalled[k]
			require.True(s.T(), in)
		}
	}

	s.Run("add and get black listed peers", func() {
		require.NoError(s.T(), s.storage.AddToBlackList(nil))

		err := s.storage.AddToBlackList(blackList)
		require.NoError(s.T(), err)
		check(blackList)
	})

	s.Run("ip is black listed", func() {
		for _, peer := range blackList {
			require.True(s.T(), s.storage.IsBlackListedIP(peer.IP, now))
		}
	})

	s.Run("ips are black listed", func() {
		empty := s.storage.IsBlackListedIPs(nil, now)
		assert.Empty(s.T(), empty)

		ips := make([]IP, 0, len(blackList))
		for _, peer := range blackList {
			ips = append(ips, peer.IP)
		}
		res := s.storage.IsBlackListedIPs(ips, now.Add(blackListDuration))
		assert.False(s.T(), res[0])
		assert.False(s.T(), res[1])
		assert.True(s.T(), res[2])
		assert.True(s.T(), res[3])
	})

	s.Run("delete and get black listed peers", func() {
		defer func() {
			require.NoError(s.T(), s.storage.AddToBlackList(blackList))
		}()

		require.NoError(s.T(), s.storage.DeleteBlackListedByIP(nil))

		err := s.storage.DeleteBlackListedByIP(blackList[:1])
		require.NoError(s.T(), err)
		check(blackList[1:])
	})

	s.Run("refresh black listed peers", func() {
		err := s.storage.RefreshBlackList(now.Add(blackListDuration))
		require.NoError(s.T(), err)
		check(blackList[2:])
	})

	s.Run("new cbor storage with black list refreshing", func() {
		defer func() {
			require.NoError(s.T(), s.storage.AddToBlackList(blackList))
		}()

		newNow := now.Add(blackListDuration)
		storage, err := newCBORStorageInDir(s.storage.storageDir, newNow, peersStorageCurrentVersion)
		require.NoError(s.T(), err)
		s.storage = storage

		testMap := make(blackListedPeers)
		for _, peer := range s.storage.BlackList(newNow) {
			testMap[peer.IP] = peer
		}

		for _, peer := range blackList[2:] {
			inMapPeer, in := testMap[peer.IP]
			assert.True(s.T(), in)
			assert.Equal(s.T(), peer, inMapPeer)
		}
	})

	s.Run("unsafe sync black listed peers bad storage file", func() {
		defer func(blackListStorageFile string) {
			require.NoError(s.T(), os.Remove(s.storage.blackListFilePath))
			s.storage.blackListFilePath = blackListStorageFile
		}(s.storage.blackListFilePath)

		badFilePath := filepath.Join(s.storage.storageDir, "test_invalid_black_list_storage_file")
		f, err := os.OpenFile(badFilePath, os.O_CREATE, 0100)
		require.NoError(s.T(), err)
		defer func() {
			require.NoError(s.T(), f.Chmod(0644))
			require.NoError(s.T(), f.Close())
		}()

		s.storage.blackListFilePath = badFilePath
		err = s.storage.unsafeSyncRestricted(nil, nil, blackListedPeersID)
		require.Error(s.T(), err)
	})
}

func (s *binaryStorageCborSuite) TestCBORStorageDropsAndVersioning() {
	suspendDuration := time.Minute * 5
	now := s.now.Truncate(time.Millisecond)
	suspended := []SuspendedPeer{
		{
			IP:                      IPFromString("13.3.4.1"),
			RestrictTimestampMillis: now.UnixMilli(),
			RestrictDuration:        suspendDuration,
			Reason:                  "some reason #1",
		},
		{
			IP:                      IPFromString("3.54.1.9"),
			RestrictTimestampMillis: now.UnixMilli(),
			RestrictDuration:        suspendDuration,
			Reason:                  "some reason #2",
		},
		{
			IP:                      IPFromString("23.43.7.43"),
			RestrictTimestampMillis: now.UnixMilli(),
			RestrictDuration:        suspendDuration + time.Minute*2,
			Reason:                  "some reason #3",
		},
		{
			IP:                      IPFromString("42.54.1.6"),
			RestrictTimestampMillis: now.UnixMilli(),
			RestrictDuration:        suspendDuration + time.Minute*2,
			Reason:                  "some reason #4",
		},
	}

	known := []KnownPeer{
		// nickeskov: this peers can be found in suspended peers
		KnownPeer(proto.NewIpPortFromTcpAddr(proto.NewTCPAddrFromString("13.3.4.1:2345"))),
		KnownPeer(proto.NewIpPortFromTcpAddr(proto.NewTCPAddrFromString("3.54.1.9:1454"))),
		KnownPeer(proto.NewIpPortFromTcpAddr(proto.NewTCPAddrFromString("23.43.7.43:4234"))),
		KnownPeer(proto.NewIpPortFromTcpAddr(proto.NewTCPAddrFromString("42.54.1.6:54356"))),

		KnownPeer(proto.NewIpPortFromTcpAddr(proto.NewTCPAddrFromString("13.8.4.1:2334"))),
		KnownPeer(proto.NewIpPortFromTcpAddr(proto.NewTCPAddrFromString("3.5.13.91:14554"))),
		KnownPeer(proto.NewIpPortFromTcpAddr(proto.NewTCPAddrFromString("3.43.7.47:4234"))),
		KnownPeer(proto.NewIpPortFromTcpAddr(proto.NewTCPAddrFromString("4.54.1.65:5356"))),
	}

	checkSuspendedStorageFile := func() {
		var unmarshalled suspendedPeers
		require.Equal(s.T(), io.EOF, unmarshalCborFromFile(s.storage.suspendedFilePath, &unmarshalled))
		require.Empty(s.T(), s.storage.Suspended(now))
	}

	checkKnownStorageFile := func() {
		var unmarshalled knownPeers
		require.Equal(s.T(), io.EOF, unmarshalCborFromFile(s.storage.knownFilePath, &unmarshalled))
		require.Empty(s.T(), s.storage.Known(10))
	}

	s.Run("drop suspended peers", func() {
		defer func() {
			require.NoError(s.T(), s.storage.AddSuspended(suspended))
		}()

		err := s.storage.DropSuspended()
		require.NoError(s.T(), err)

		checkSuspendedStorageFile()
	})

	s.Run("drop known peers", func() {
		defer func() {
			require.NoError(s.T(), s.storage.AddOrUpdateKnown(known, time.Now()))
		}()

		err := s.storage.DropKnown()
		require.NoError(s.T(), err)

		checkKnownStorageFile()
	})

	s.Run("drop peers storage", func() {
		defer func() {
			require.NoError(s.T(), s.storage.AddSuspended(suspended))
			require.NoError(s.T(), s.storage.AddOrUpdateKnown(known, time.Now()))
		}()

		err := s.storage.DropStorage()
		require.NoError(s.T(), err)

		checkSuspendedStorageFile()
		checkKnownStorageFile()
	})

	s.Run("drop peers storage in case of different version", func() {
		versionFilePath := storageVersionFilePath(s.storage.storageDir)
		defer func() {
			storage, err := newCBORStorageInDir(s.storage.storageDir, s.now, peersStorageCurrentVersion)
			require.NoError(s.T(), err)
			s.storage = storage

			version, err := getPeersStorageVersion(versionFilePath)
			require.NoError(s.T(), err)
			require.Equal(s.T(), peersStorageCurrentVersion, version)

			require.NoError(s.T(), s.storage.AddSuspended(suspended))
			require.NoError(s.T(), s.storage.AddOrUpdateKnown(known, time.Now()))
		}()

		storage, err := newCBORStorageInDir(s.storage.storageDir, s.now, -1)
		require.NoError(s.T(), err)
		s.storage = storage

		version, err := getPeersStorageVersion(versionFilePath)
		require.NoError(s.T(), err)
		require.Equal(s.T(), -1, version)

		checkSuspendedStorageFile()
		checkKnownStorageFile()
	})
}
