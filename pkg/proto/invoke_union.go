package proto

import "github.com/wavesplatform/gowaves/pkg/crypto"

type InvokeTxUnion interface {
	TxID() crypto.Digest
}

type InvokeScriptSubTransaction interface {
	EmptyInvokeScriptTxMethod()
}

type InvokeExpressionSubTransaction interface {
	EmptyInvokeExpressionTxMethod()
}




type InvokeScriptTxUnion struct {
	SubTx InvokeScriptSubTransaction
	txID crypto.Digest
}

func NewInvokeScriptTxUnion(subTransaction InvokeScriptSubTransaction, id crypto.Digest) *InvokeScriptTxUnion {
	return &InvokeScriptTxUnion{SubTx: subTransaction, txID: id}
}

func (invScript InvokeScriptTxUnion) TxID() crypto.Digest {
	return invScript.txID
}

type InvokeExpressionTxUnion struct {
	SubTx InvokeExpressionSubTransaction
	txID crypto.Digest
}

func NewInvokeExpressionTxUnion(subTransaction InvokeExpressionSubTransaction, id crypto.Digest) *InvokeExpressionTxUnion {
	return &InvokeExpressionTxUnion{SubTx: subTransaction, txID: id}
}

func (invExp InvokeExpressionTxUnion) TxID() crypto.Digest{
	return invExp.txID
}
