package proto

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/libs/deserializer"
)

type Signer struct {
	Generator crypto.PublicKey
	Signature crypto.Signature
}

type MicroBlock struct {
	versionField          byte
	prevResBlockSigField  crypto.Signature
	totalResBlockSigField crypto.Signature
	signer                Signer
	transactionsBytes     []byte
}

func (a *MicroBlock) UnmarshalBinary(b []byte) error {
	var err error
	d := deserializer.NewDeserializer(b)

	a.versionField, err = d.Byte()
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal microblock version")
	}

	a.prevResBlockSigField, err = d.Signature()
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal microblock prevResBlockSigField")
	}

	a.totalResBlockSigField, err = d.Signature()
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal microblock totalResBlockSigField")
	}

	tBytesLength, err := d.Uint32()
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal microblock transaction bytes len")
	}

	a.transactionsBytes, err = d.Bytes(uint(tBytesLength))
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal microblock transaction bytes")
	}

	a.signer.Generator, err = d.PublicKey()
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal microblock public key")
	}

	a.signer.Signature, err = d.Signature()
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal microblock signature")
	}

	return nil
}
