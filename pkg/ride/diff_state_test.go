package ride

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
)

const (
	binaryKey    = "binary-key"
	booleanKey   = "boolean-key"
	stringValue  = "string-value"
	integerValue = 1234567890
)

var (
	validAddress   = proto.MustAddressFromString("3MtBQhYqSeVLpBSMB2jTkb6vs8TrNFFBva3")
	validRecipient = proto.NewRecipientFromAddress(validAddress)
	//invalidAddress   = proto.MustAddressFromString("3N4NS7d4Jo9a6F14LiFUKKYVdUkkf2eP4Zx")
	binaryValue = []byte{0xca, 0xfe, 0xbe, 0xbe, 0xde, 0xad, 0xbe, 0xef}
)

type DataDiffTestSuite struct {
	suite.Suite
	diff diffState
}

func TestDataDiffTestSuite(t *testing.T) {
	suite.Run(t, new(DataDiffTestSuite))
}

func (s *DataDiffTestSuite) SetupTest() {
	m := types.NewMockEnrichedSmartState(s.T())
	m.EXPECT().NewestRecipientToAddress(mock.Anything).RunAndReturn(
		func(recipient proto.Recipient) (proto.WavesAddress, error) {
			if recipient.Eq(validRecipient) {
				return validAddress, nil
			}
			return proto.WavesAddress{}, errors.New("not found")
		},
	).Maybe()
	s.diff = newDiffState(m)
}

func (s *DataDiffTestSuite) TestPutGetBinaryEntry() {
	entry := &proto.BinaryDataEntry{Key: binaryKey, Value: binaryValue}
	s.diff.putDataEntry(entry, validAddress)
	actual := s.diff.findBinaryFromDataEntryByKey(binaryKey, validAddress)
	s.Require().NotNil(entry)
	s.Equal(entry, actual)
}

func (s *DataDiffTestSuite) TestDeleteBinaryEntry() {
	entry := &proto.BinaryDataEntry{Key: binaryKey, Value: binaryValue}
	s.diff.putDataEntry(entry, validAddress)
	first := s.diff.findBinaryFromDataEntryByKey(binaryKey, validAddress)
	s.Require().NotNil(first)
	s.Equal(entry, first)

	del := &proto.DeleteDataEntry{Key: binaryKey}
	s.diff.putDataEntry(del, validAddress)
	actual := s.diff.findDeleteFromDataEntryByKey(binaryKey, validAddress)
	s.Equal(del, actual)

	second := s.diff.findBinaryFromDataEntryByKey(binaryKey, validAddress)
	s.Assert().Nil(second)
}

func (s *DataDiffTestSuite) TestDeleteAndRestoreBinaryEntry() {
	binary := &proto.BinaryDataEntry{Key: binaryKey, Value: binaryValue}
	s.diff.putDataEntry(binary, validAddress)
	first := s.diff.findBinaryFromDataEntryByKey(binaryKey, validAddress)
	s.Require().NotNil(first)
	s.Equal(binary, first)

	del := &proto.DeleteDataEntry{Key: binaryKey}
	s.diff.putDataEntry(del, validAddress)
	second := s.diff.findDeleteFromDataEntryByKey(binaryKey, validAddress)
	s.Equal(del, second)

	third := s.diff.findBinaryFromDataEntryByKey(binaryKey, validAddress)
	s.Assert().Nil(third)

	s.diff.putDataEntry(binary, validAddress)
	forth := s.diff.findDeleteFromDataEntryByKey(binaryKey, validAddress)
	s.Assert().Nil(forth)
	fifth := s.diff.findBinaryFromDataEntryByKey(binaryKey, validAddress)
	s.Assert().NotNil(fifth)
	s.Equal(binary, fifth)
}

func (s *DataDiffTestSuite) TestReplaceBinaryEntry() {
	entry := &proto.BinaryDataEntry{Key: binaryKey, Value: binaryValue}
	s.diff.putDataEntry(entry, validAddress)
	first := s.diff.findBinaryFromDataEntryByKey(binaryKey, validAddress)
	s.Require().NotNil(entry)
	s.Equal(entry, first)

	str := &proto.StringDataEntry{Key: binaryKey, Value: stringValue}
	s.diff.putDataEntry(str, validAddress)
	second := s.diff.findStringFromDataEntryByKey(binaryKey, validAddress)
	s.Require().NotNil(second)
	s.Equal(str, second)

	third := s.diff.findBinaryFromDataEntryByKey(binaryKey, validAddress)
	s.Require().Nil(third)
}

func (s *DataDiffTestSuite) TestPutGetBooleanEntry() {
	entry := &proto.BooleanDataEntry{Key: booleanKey, Value: true}
	s.diff.putDataEntry(entry, validAddress)
	actual := s.diff.findBoolFromDataEntryByKey(booleanKey, validAddress)
	s.Require().NotNil(entry)
	s.Equal(entry, actual)
}

func (s *DataDiffTestSuite) TestDeleteBooleanEntry() {
	entry := &proto.BooleanDataEntry{Key: booleanKey, Value: true}
	s.diff.putDataEntry(entry, validAddress)
	first := s.diff.findBoolFromDataEntryByKey(booleanKey, validAddress)
	s.Require().NotNil(first)
	s.Equal(entry, first)

	del := &proto.DeleteDataEntry{Key: booleanKey}
	s.diff.putDataEntry(del, validAddress)
	actual := s.diff.findDeleteFromDataEntryByKey(booleanKey, validAddress)
	s.Equal(del, actual)

	second := s.diff.findBoolFromDataEntryByKey(booleanKey, validAddress)
	s.Assert().Nil(second)
}

func (s *DataDiffTestSuite) TestDeleteAndRestoreBooleanEntry() {
	boolean := &proto.BooleanDataEntry{Key: booleanKey, Value: true}
	s.diff.putDataEntry(boolean, validAddress)
	first := s.diff.findBoolFromDataEntryByKey(booleanKey, validAddress)
	s.Require().NotNil(first)
	s.Equal(boolean, first)

	del := &proto.DeleteDataEntry{Key: booleanKey}
	s.diff.putDataEntry(del, validAddress)
	second := s.diff.findDeleteFromDataEntryByKey(booleanKey, validAddress)
	s.Equal(del, second)

	third := s.diff.findBinaryFromDataEntryByKey(booleanKey, validAddress)
	s.Assert().Nil(third)

	s.diff.putDataEntry(boolean, validAddress)
	forth := s.diff.findDeleteFromDataEntryByKey(booleanKey, validAddress)
	s.Assert().Nil(forth)
	fifth := s.diff.findBoolFromDataEntryByKey(booleanKey, validAddress)
	s.Assert().NotNil(fifth)
	s.Equal(boolean, fifth)
}

func (s *DataDiffTestSuite) TestReplaceBooleanEntry() {
	entry := &proto.BooleanDataEntry{Key: booleanKey, Value: true}
	s.diff.putDataEntry(entry, validAddress)
	first := s.diff.findBoolFromDataEntryByKey(booleanKey, validAddress)
	s.Require().NotNil(entry)
	s.Equal(entry, first)

	integer := &proto.IntegerDataEntry{Key: booleanKey, Value: integerValue}
	s.diff.putDataEntry(integer, validAddress)
	second := s.diff.findIntFromDataEntryByKey(booleanKey, validAddress)
	s.Require().NotNil(second)
	s.Equal(integer, second)

	third := s.diff.findBoolFromDataEntryByKey(booleanKey, validAddress)
	s.Require().Nil(third)
}

func TestErrorOnDuplicateLeasing(t *testing.T) {
	m := types.NewMockEnrichedSmartState(t)
	m.EXPECT().WavesBalanceProfile(validRecipient.Address().ID()).Return(
		&types.WavesBalanceProfile{
			Balance:    10000_00000000,
			LeaseIn:    0,
			LeaseOut:   0,
			Generating: 0,
			Challenged: false,
		}, nil).Once()
	diff := newDiffState(m)
	digest1 := crypto.MustDigestFromBase58("8N6F4oV2SmfWZ45xVNLQr2rjHyvDWNz8R3wxJzE83ZHm")
	digest2 := crypto.MustDigestFromBase58("9N6F4oV2SmfWZ45xVNLQr2rjHyvDWNz8R3wxJzE83ZHm")
	err1 := diff.lease(*validRecipient.Address(), *validRecipient.Address(), 1000, digest1)
	assert.NoError(t, err1)
	err2 := diff.lease(*validRecipient.Address(), *validRecipient.Address(), 1000, digest2)
	assert.NoError(t, err2)
	err3 := diff.lease(*validRecipient.Address(), *validRecipient.Address(), 1000, digest1)
	assert.EqualError(t, err3,
		"lease with id '8N6F4oV2SmfWZ45xVNLQr2rjHyvDWNz8R3wxJzE83ZHm' already exists in ride execution diff")
}
