package client

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type EthereumTransactionType byte

const (
	EthereumTransactionTransferType EthereumTransactionType = iota + 1
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

type EthereumTransactionInvocationPayload InvokeAction

func (p *EthereumTransactionInvocationPayload) GetType() EthereumTransactionType {
	return EthereumTransactionInvocationType
}

func (p *EthereumTransactionInvocationPayload) GetTypeString() string {
	return EthereumTransactionInvocationTypeString
}

type ethereumTransactionTypeDetector struct {
	Payload struct {
		Type string `json:"type"`
	} `json:"payload"`
}

func guessEthereumTransactionPayload(detector *ethereumTransactionTypeDetector) (EthereumTransactionPayload, error) {
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
	transactionInfoCommonImpl

	Payload EthereumTransactionPayload `json:"payload"`
}

func (txInfo *EthereumTransactionInfo) getInfoCommonObject() *transactionInfoCommonImpl {
	return &txInfo.transactionInfoCommonImpl
}

func (txInfo *EthereumTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.EthereumTransaction
}

func (txInfo *EthereumTransactionInfo) UnmarshalJSON(data []byte) error {
	return transactionInfoUnmarshalJSON(data, txInfo)
}

func (txInfo *EthereumTransactionInfo) unmarshalSpecificData(data []byte) error {
	detector := new(ethereumTransactionTypeDetector)
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
