package proto

import (
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type TransactionType byte

const (
	GenesisTransaction TransactionType = iota + 1
	PaymentTransaction
	IssueTransaction
	TransferTransaction
	ReissueTransaction
	BurnTransaction
	ExchangeTransaction
	LeaseTransaction
	LeaseCancelTransaction
	CreateAliasTransaction
	MassTransferTransaction
	DataTransaction
	SetScriptTransaction
	SponsorshipTransaction
)

const (
	maxAttachmentLengthBytes      = 140
	maxDescriptionLen             = 1000
	maxAssetNameLen               = 16
	minAssetNameLen               = 4
	maxDecimals                   = 8
	proofsVersion            byte = 1

	fixedIssueV1BodyLen = 1 + crypto.PublicKeySize + 2 + 2 + 8 + 1 + 1 + 8 + 8
	minIssueV1BodyLen   = fixedIssueV1BodyLen + 4
	minIssueV1Len       = 1 + crypto.SignatureSize + minIssueV1BodyLen
)

type Genesis struct {
	Type      TransactionType `json:"type"`
	Timestamp uint64          `json:"timestamp"`
	Recipient Address         `json:"recipient"`
	Amount    uint64          `json:"amount"`
}

type IssueV1 struct {
	Type        TransactionType   `json:"type"`
	Version     byte              `json:"version,omitempty"`
	ID          *crypto.Digest    `json:"id,omitempty"`
	Signature   *crypto.Signature `json:"signature,omitempty"`
	SenderPK    crypto.PublicKey  `json:"senderPublicKey"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Quantity    uint64            `json:"quantity"`
	Decimals    byte              `json:"decimals"`
	Reissuable  bool              `json:"reissuable"`
	Timestamp   uint64            `json:"timestamp,omitempty"`
	Fee         uint64            `json:"fee"`
}

func NewUnsignedIssueV1(senderPK crypto.PublicKey, name, description string, quantity uint64, decimals byte, reissuable bool, timestamp, fee uint64) (*IssueV1, error) {
	if l := len(name); l < minAssetNameLen || l > maxAssetNameLen {
		return nil, errors.New("incorrect number of bytes in the asset's name")
	}
	if l := len(description); l > maxDescriptionLen {
		return nil, errors.New("incorrect number of bytes in the asset's description")
	}
	if quantity <= 0 {
		return nil, errors.New("quantity should be positive")
	}
	if decimals > maxDecimals {
		return nil, errors.Errorf("incorrect decimals, should be no more then %d", maxDecimals)
	}
	if fee <= 0 {
		return nil, errors.New("fee should be positive")
	}
	return &IssueV1{Type: IssueTransaction, Version: 1, SenderPK: senderPK, Name: name, Description: description, Quantity: quantity, Decimals: decimals, Reissuable: reissuable, Timestamp: timestamp, Fee: fee}, nil
}

func (tx *IssueV1) marshalBody() []byte {
	kl := crypto.PublicKeySize
	nl := len(tx.Name)
	dl := len(tx.Description)
	buf := make([]byte, fixedIssueV1BodyLen+nl+dl)
	buf[0] = byte(tx.Type)
	copy(buf[1:], tx.SenderPK[:])
	PutStringWithUInt16Len(buf[1+kl:], tx.Name)
	PutStringWithUInt16Len(buf[3+kl+nl:], tx.Description)
	binary.BigEndian.PutUint64(buf[5+kl+nl+dl:], tx.Quantity)
	buf[13+kl+nl+dl] = tx.Decimals
	PutBool(buf[14+kl+nl+dl:], tx.Reissuable)
	binary.BigEndian.PutUint64(buf[15+kl+nl+dl:], tx.Fee)
	binary.BigEndian.PutUint64(buf[23+kl+nl+dl:], tx.Timestamp)
	return buf
}

func (tx *IssueV1) unmarshalBody(data []byte) error {
	tx.Type = TransactionType(data[0])
	tx.Version = 1
	if l := len(data); l < minIssueV1BodyLen {
		return fmt.Errorf("not enough data for IssueV1 transaction %d, expected not less then %d", l, minIssueV1BodyLen)
	}
	if tx.Type != IssueTransaction {
		return fmt.Errorf("unexpected transaction type %d for IssueV1 transaction", tx.Type)
	}
	data = data[1:]
	copy(tx.SenderPK[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	var err error
	tx.Name, err = StringWithUInt16Len(data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal Name: %s", err.Error())
	}
	data = data[2+len(tx.Name):]
	tx.Description, err = StringWithUInt16Len(data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal Description: %s", err.Error())
	}
	data = data[2+len(tx.Description):]
	tx.Quantity = binary.BigEndian.Uint64(data)
	data = data[8:]
	tx.Decimals = data[0]
	data = data[1:]
	tx.Reissuable, err = Bool(data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal Reissuble: %s", err.Error())
	}
	data = data[1:]
	tx.Fee = binary.BigEndian.Uint64(data)
	data = data[8:]
	tx.Timestamp = binary.BigEndian.Uint64(data)
	return nil
}

func (tx *IssueV1) Sign(secretKey crypto.SecretKey) error {
	b := tx.marshalBody()
	s := crypto.Sign(secretKey, b)
	tx.Signature = &s
	var err error
	d, err := crypto.FastHash(b)
	if err != nil {
		return errors.Wrap(err, "failed to hash Issue")
	}
	tx.ID = &d
	return nil
}

func (tx *IssueV1) Verify(publicKey crypto.PublicKey) bool {
	return crypto.Verify(publicKey, *tx.Signature, tx.marshalBody())
}

func (tx *IssueV1) MarshalBinary() ([]byte, error) {
	sl := crypto.SignatureSize
	b := tx.marshalBody()
	bl := len(b)
	buf := make([]byte, 1+sl+bl)
	buf[0] = byte(tx.Type)
	copy(buf[1:], tx.Signature[:])
	copy(buf[1+sl:], b)
	return buf, nil
}

func (tx *IssueV1) UnmarshalBinary(data []byte) error {
	if l := len(data); l < minIssueV1Len {
		return fmt.Errorf("not enough data for IssueV1 transaction, expected not less then %d, received %d", minIssueV1Len, l)
	}
	if data[0] != byte(IssueTransaction) {
		return fmt.Errorf("incorrect transaction type %d for IssueV1 transaction", data[0])
	}
	data = data[1:]
	var s crypto.Signature
	copy(s[:], data[:crypto.SignatureSize])
	tx.Signature = &s
	data = data[crypto.SignatureSize:]
	err := tx.unmarshalBody(data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal IssueV1 transaction: %s", err.Error())
	}
	d, err := crypto.FastHash(data)
	if err != nil {
		return errors.Wrap(err, "failed to hash Issue")
	}
	tx.ID = &d
	return nil
}
