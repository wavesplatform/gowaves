package client

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

/*
** TransactionInfo
 */

type TransactionInfo interface {
	// Supposed that each struct which implements this interface embeddes
	// related proto.<*>Transaction struct and TransactionInfoCommon struct.

	proto.Transaction

	// Must call common function TransactionInfoUnmarshalJSON and pass itself as the 2nd argument
	json.Unmarshaler

	TransactionInfoCommon

	unmarshalSpecificData(data []byte) error
	getTransactionObject() proto.Transaction
	getInfoCommonObject() *TransactionInfoCommon
}

func TransactionInfoUnmarshalJSON(data []byte, txInfo TransactionInfo) error {
	if err := json.Unmarshal(data, txInfo.getTransactionObject()); err != nil {
		return errors.Wrap(err, "Unmarshal proto.Transaction failed")
	}

	if err := json.Unmarshal(data, txInfo.getInfoCommonObject()); err != nil {
		return errors.Wrap(err, "Unmarshal TransactionInfoCommon failed")
	}

	if err := txInfo.unmarshalSpecificData(data); err != nil {
		return errors.Wrap(err, "Unmarshal specific data failed")
	}
	return nil
}

/*
** TransactionInfoCommon
 */

type TransactionInfoCommon interface {
	GetSpentComplexity() int
	GetHeight() proto.Height
}

type TransactionInfoCommonImpl struct {
	SpentComplexity int          `json:"spentComplexity"`
	Height          proto.Height `json:"height"`
}

func (txInfoCommon *TransactionInfoCommonImpl) GetSpentComplexity() int {
	return txInfoCommon.SpentComplexity
}

func (txInfoCommon *TransactionInfoCommonImpl) GetHeight() proto.Height {
	return txInfoCommon.Height
}

/*
** Ethereum Transaction
 */

type EthereumTransactionType byte

const (
	EthereumTransactionTransferType EthereumTransactionType = iota
	EthereumTransactionInvocationType
)

const (
	EthereumTransactionTransferTypeString   string = "transfer"
	EthereumTransactionInvocationTypeString string = "invocation"
)

type EthereumTransactionPayload interface {
	GetType() EthereumTransactionType
	GetTypeString() string
}

type EthereumTransactionTransferPayload struct {
	Recipient proto.Recipient     `json:"recipient"`
	Asset     proto.OptionalAsset `json:"asset"`
	Amount    uint64              `json:"amount"`
}

func (p *EthereumTransactionTransferPayload) GetType() EthereumTransactionType {
	return EthereumTransactionTransferType
}

func (p *EthereumTransactionTransferPayload) GetTypeString() string {
	return EthereumTransactionTransferTypeString
}

// TODO
type EthereumTransactionInvocationPayload struct{}

func (p *EthereumTransactionInvocationPayload) GetType() EthereumTransactionType {
	return EthereumTransactionInvocationType
}

func (p *EthereumTransactionInvocationPayload) GetTypeString() string {
	return EthereumTransactionInvocationTypeString
}

type EthereumTransactionTypeDetector struct {
	Payload struct {
		Type string `json:"type"`
	} `json:"payload"`
}

func guessEthereumTransactionPayload(detector *EthereumTransactionTypeDetector) (EthereumTransactionPayload, error) {
	switch detector.Payload.Type {
	case EthereumTransactionTransferTypeString:
		return &EthereumTransactionTransferPayload{}, nil
	case EthereumTransactionInvocationTypeString:
		return &EthereumTransactionInvocationPayload{}, nil
	default:
		return nil, errors.Errorf("Unknown payload type: %s", detector.Payload.Type)
	}
}

type EthereumTransactionInfo struct {
	proto.EthereumTransaction
	TransactionInfoCommon

	Payload EthereumTransactionPayload `json:"payload"`
}

func (txInfo *EthereumTransactionInfo) getInfoCommonObject() *TransactionInfoCommon {
	return &txInfo.TransactionInfoCommon
}

func (txInfo *EthereumTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.EthereumTransaction
}

func (txInfo *EthereumTransactionInfo) UnmarshalJSON(data []byte) error {
	return TransactionInfoUnmarshalJSON(data, txInfo)
}

func (txInfo *EthereumTransactionInfo) unmarshalSpecificData(data []byte) error {
	detector := new(EthereumTransactionTypeDetector)
	if err := json.Unmarshal(data, detector); err != nil {
		return errors.Wrap(err, "Ethereum transaction type unmarshal")
	}

	if len(detector.Payload.Type) == 0 {
		return errors.New("Field 'type' in Ethereum transaction payload is empty or missing")
	}

	payload, err := guessEthereumTransactionPayload(detector)
	if err != nil {
		return errors.Wrap(err, "Guess Ethereum transaction type")
	}

	tmp := struct {
		Payload EthereumTransactionPayload `json:"payload"`
	}{
		Payload: payload,
	}

	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	txInfo.Payload = tmp.Payload
	return nil
}
