package ride

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/wavesplatform/gowaves/pkg/proto"
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
	mock := &MockSmartState{
		NewestRecipientToAddressFunc: func(recipient proto.Recipient) (*proto.WavesAddress, error) {
			if recipient.Eq(validRecipient) {
				return &validAddress, nil
			}
			return nil, errors.New("not found")
		},
	}
	s.diff = newDiffState(mock)
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
