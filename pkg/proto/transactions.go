package proto

import (
	"bytes"
	"encoding/binary"
	"github.com/pkg/errors"
	"github.com/wavewplatform/gowaves/pkg/crypto"
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
)

type IssueV1 struct {
	Type        TransactionType  `json:"type"`
	Version     byte             `json:"version,omitempty"`
	ID          crypto.Digest    `json:"id"`
	Signature   crypto.Signature `json:"signature"`
	SenderPK    crypto.PublicKey `json:"senderPublicKey"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Quantity    uint64           `json:"quantity"`
	Decimals    byte             `json:"decimals"`
	Reissuable  bool             `json:"reissuable"`
	Timestamp   uint64           `json:"timestamp,omitempty"`
	Fee         uint64           `json:"fee"`
}

func NewUnsignedIssue(senderPK crypto.PublicKey, name, description string, quantity uint64, decimals byte, reissuable bool, timestamp int64, fee uint64) (*IssueV1, error) {
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

func (tx *IssueV1) body() []byte {
	nameLen := len(tx.Name)
	descLen := len(tx.Description)
	buf := make([]byte, 0, 1+crypto.PublicKeySize+2+nameLen+2+descLen+8+1+1+8+8)
	var p int

	buf[p] = byte(tx.Type)
	p += 1

	copy(buf[p:p+crypto.PublicKeySize], tx.SenderPK[:])
	p += crypto.PublicKeySize

	d := make([]byte, 2)
	binary.BigEndian.PutUint16(d, uint16(nameLen))
	copy(buf[p:p+2], d[:])
	p += 2

	copy(buf[p:p+nameLen], tx.Name[:])
	p += nameLen

	d = make([]byte, 2)
	binary.BigEndian.PutUint16(d, uint16(descLen))
	copy(buf[p:p+2], d[:])
	p += 2

	copy(buf[p:p+descLen], tx.Description[:])
	p += descLen

	d = make([]byte, 8)
	binary.BigEndian.PutUint64(d, tx.Quantity)
	copy(buf[p:p+8], d[:])
	p += 8

	buf[p] = tx.Decimals
	p += 1

	if tx.Reissuable {
		buf[p] = 1
	} else {
		buf[p] = 0
	}
	p += 1

	d = make([]byte, 8)
	binary.BigEndian.PutUint64(d, tx.Fee)
	copy(buf[p:p+8], d[:])
	p += 8

	d = make([]byte, 8)
	binary.BigEndian.PutUint64(d, tx.Timestamp)
	copy(buf[p:p+8], d[:])
	p += 8

	return buf
}

func (tx *IssueV1) Sign(secretKey crypto.SecretKey) error {
	b := tx.body()
	tx.Signature = crypto.Sign(secretKey, b)
	var err error
	tx.ID, err = crypto.FastHash(b)
	if err != nil {
		return errors.Wrap(err, "failed to hash Issue")
	}
	return nil
}

func (tx *IssueV1) Verify(publicKey crypto.PublicKey) bool {
	return crypto.Verify(publicKey, tx.Signature, tx.body())
}

func (tx *IssueV1) MarshalBinary() ([]byte, error) {
	buf := &bytes.Buffer{}
	if err := buf.WriteByte(byte(tx.Type)); err != nil {
		return nil, err
	}
	if _, err := buf.Write(tx.Signature[:]); err != nil {
		return nil, err
	}
	b := tx.body()
	if _, err := buf.Write(b); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
