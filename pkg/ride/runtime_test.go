package ride

import (
	"encoding/base64"
	"math/big"
	"testing"

	"github.com/mr-tron/base58"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
)

func TestRideBytesScalaString(t *testing.T) {
	shortBytes, err := base58.Decode("3MbexUVHr88VyDborjw9Veh77MJ68BMNoKi")
	require.NoError(t, err)
	longBytes, err := base64.StdEncoding.DecodeString("Uv38ByGCZU8WP18PmmIdcpVmx00QA3xNe7sEB9HixkmBhVrYaB0NhtHpHgAWeTnLZpTSxCKs0gigByk5SH9pmeudGKRHhARdh/PGfPInRumVr1olNnlRuqL/bNRxxIPxX7kLrbN8WCG22VUmpBqVBGgLTnyLdjobHUnUlVyEhiFjJSU/7HON16nii/khEZwWDwcCRIYVu9oIMT9qjrZo0gv1BZh1kh5milvfLH/EhEWS0lcrzQZo0tbFL1BU4tCDa/hMcXTLdHY2TMPb2Wiw9xcu2FeUuzWLDDtSXaF4b5//CUJ52xlE69ehnQ97usvgJVqlt9RL7ED4TIkrm//UNimwIjvupfT3Q5H0RdFa/UKUBAN09pJLmMv4cT+Nli18jQGRksJCJOLK/Mrjph+1hrFDI6a8j5598dkpMz/5k5M76m9bOvbeA3Q2bEcZ5DobBn2JvH8B8fVzmBZZpE/xekxyFaO1OeseWEnGB327VyL1cXoomiZvl2R5gZmOvqicC0s3OXARXoLtb0ElyPpzEeTX3vqSLarneGZn9+k2zU8kq/ffhmuqVgODZ61hRd4e6PSosJk+vfiIOgrYvpw5eLBIg+VqFWqN5WOvpGfUnexqQOmh0AfwM8KCMGG90Oqln45NpkMBBSINCyloi3NLjqDzypk26EYfENd8luqAp6Zl9gb2pjt/Pf0lZ8GJeeTWDyZobZvy+ybJAf81TN4WB+4pSznzK3x4Irpk+Eq0PKDG5rkcH9O+iZBDQXnTr0SRo2kBLbktGE/DnRc0/1cWQolTu2hl/PkrDDoXyQKL6ZFOt2ScbJNHgAl50YMDVvKlTD3qsqS0R11jr76PtWmHx39YGFJvGBS+gjNQ6rE5NfMdhEhFF+kkrveK4VHAB1WSWDa3B1iFZQww7CmjcDk0v1CijaECl13tp351hXnqPf5BNqv3UrO4Jx0D6USzyds2a3UEX479adIq5UEZR8tVPXaUJnrvTrzqQGsy1hCL1oWE9X43yqxuM/6qMmOjmUNwJLqcmxRniidPAakQrilfbvv+X1q/RMzeJjtWBmM+K/AAbygpXX05Bp8BojnENlhUw69/a0HWMfkrmo0S9BJXMl//My91drBiBVYwSj4+rhTCjQzqOdKQGlJyDahcoeSzjq8/RMbG74Ni8vVPwA4J1vwlZAhUwV38rKqKLOzOWjq6U6twWxjblLTTOKUUPmNAjYcksM8/rhej95vhBy+2PDXWBCxBYPOO6eKp8/tP+wAZtFTVIrX/oXYEGT+4lmcQp5YHMspSz1PD9SDIibeb9QTPtXx2ASMtWJuszqnW4mPiXCd0HT9sYsu7FdmvvL9/faQasECOOWnC4s3PIzQ4vxd0rOdwmk8JHpqD/erg7FXrIzqbU5TLPHhWtUbTE8ijtMHA4FRH9Lo3DrNtvP3skLMC3Nw7nvUi4qbx7Qr+wfjiD6q+32sWLnF9OnSKWGd6DFY0j4khomaxHQ8zTGL+UrpTrxl3nLKUi2Vw/6C3c5Y8EwrXl93q/k460ptRJSEPDvHDFAkPB8eab1ccJG8+msC3QT7xEL1YsAznO/9wb3/0tvRAkKMnEfMgjk5LictRZc5kACy9nCiHqhE98kaJKNWiO5ynQPgMk4LZxgNK0pYMeQ==")
	require.NoError(t, err)
	shortRideBytes := rideBytes(shortBytes)
	longRideBytes := rideBytes(longBytes)
	assert.Equal(t, "3MbexUVHr88VyDborjw9Veh77MJ68BMNoKi", shortRideBytes.scalaString())
	assert.Equal(t, "base64:Uv38ByGCZU8WP18PmmIdcpVmx00QA3xNe7sEB9HixkmBhVrYaB0NhtHpHgAWeTnLZpTSxCKs0gigByk5SH9pmeudGKRHhARdh/PGfPInRumVr1olNnlRuqL/bNRxxIPxX7kLrbN8WCG22VUmpBqVBGgLTnyLdjobHUnUlVyEhiFjJSU/7HON16nii/khEZwWDwcCRIYVu9oIMT9qjrZo0gv1BZh1kh5milvfLH/EhEWS0lcrzQZo0tbFL1BU4tCDa/hMcXTLdHY2TMPb2Wiw9xcu2FeUuzWLDDtSXaF4b5//CUJ52xlE69ehnQ97usvgJVqlt9RL7ED4TIkrm//UNimwIjvupfT3Q5H0RdFa/UKUBAN09pJLmMv4cT+Nli18jQGRksJCJOLK/Mrjph+1hrFDI6a8j5598dkpMz/5k5M76m9bOvbeA3Q2bEcZ5DobBn2JvH8B8fVzmBZZpE/xekxyFaO1OeseWEnGB327VyL1cXoomiZvl2R5gZmOvqicC0s3OXARXoLtb0ElyPpzEeTX3vqSLarneGZn9+k2zU8kq/ffhmuqVgODZ61hRd4e6PSosJk+vfiIOgrYvpw5eLBIg+VqFWqN5WOvpGfUnexqQOmh0AfwM8KCMGG90Oqln45NpkMBBSINCyloi3NLjqDzypk26EYfENd8luqAp6Zl9gb2pjt/Pf0lZ8GJeeTWDyZobZvy+ybJAf81TN4WB+4pSznzK3x4Irpk+Eq0PKDG5rkcH9O+iZBDQXnTr0SRo2kBLbktGE/DnRc0/1cWQolTu2hl/PkrDDoXyQKL6ZFOt2ScbJNHgAl50YMDVvKlTD3qsqS0R11jr76PtWmHx39YGFJvGBS+gjNQ6rE5NfMdhEhFF+kkrveK4VHAB1WSWDa3B1iFZQww7CmjcDk0v1CijaECl13tp351hXnqPf5BNqv3UrO4Jx0D6USzyds2a3UEX479adIq5UEZR8tVPXaUJnrvTrzqQGsy1hCL1oWE9X43yqxuM/6qMmOjmUNwJLqcmxRniidPAakQrilfbvv+X1q/RMzeJjtWBmM+K/AAbygpXX05Bp8BojnENlhUw69/a0HWMfkrmo0S9BJXMl//My91drBiBVYwSj4+rhTCjQzqOdKQGlJyDahcoeSzjq8/RMbG74Ni8vVPwA4J1vwlZAhUwV38rKqKLOzOWjq6U6twWxjblLTTOKUUPmNAjYcksM8/rhej95vhBy+2PDXWBCxBYPOO6eKp8/tP+wAZtFTVIrX/oXYEGT+4lmcQp5YHMspSz1PD9SDIibeb9QTPtXx2ASMtWJuszqnW4mPiXCd0HT9sYsu7FdmvvL9/faQasECOOWnC4s3PIzQ4vxd0rOdwmk8JHpqD/erg7FXrIzqbU5TLPHhWtUbTE8ijtMHA4FRH9Lo3DrNtvP3skLMC3Nw7nvUi4qbx7Qr+wfjiD6q+32sWLnF9OnSKWGd6DFY0j4khomaxHQ8zTGL+UrpTrxl3nLKUi2Vw/6C3c5Y8EwrXl93q/k460ptRJSEPDvHDFAkPB8eab1ccJG8+msC3QT7xEL1YsAznO/9wb3/0tvRAkKMnEfMgjk5LictRZc5kACy9nCiHqhE98kaJKNWiO5ynQPgMk4LZxgNK0pYMeQ==",
		longRideBytes.scalaString())
}

func replaceFirstProof(obj rideProven, sig crypto.Signature) {
	p0 := rideBytes(sig.Bytes())
	pfs := obj.getProofs()
	if len(pfs) > 0 {
		pfs[0] = p0
	} else {
		pfs = rideList{p0}
	}
	obj.setProofs(pfs)
}

func makeGenesisTransactionObject(t *testing.T, txID, recipientAddress string, amount, ts int) rideType {
	id, err := crypto.NewSignatureFromBase58(txID)
	require.NoError(t, err)
	recipient, err := proto.NewAddressFromString(recipientAddress)
	require.NoError(t, err)
	tx := proto.NewUnsignedGenesis(recipient, uint64(amount), uint64(ts))
	sk := crypto.SecretKey{}
	err = tx.Sign(proto.TestNetScheme, sk)
	require.NoError(t, err)
	obj, err := genesisToObject(proto.TestNetScheme, tx)
	require.NoError(t, err)
	obj.id = id.Bytes()
	return obj
}

func makePaymentTransactionObject(t *testing.T, sig, senderPublicKey, recipientAddress string, amount, fee, ts int) rideType {
	s, err := crypto.NewSignatureFromBase58(sig)
	require.NoError(t, err)
	senderPK, err := crypto.NewPublicKeyFromBase58(senderPublicKey)
	require.NoError(t, err)
	recipient, err := proto.NewAddressFromString(recipientAddress)
	require.NoError(t, err)
	tx := proto.NewUnsignedPayment(senderPK, recipient, uint64(amount), uint64(fee), uint64(ts))
	sk := crypto.SecretKey{}
	err = tx.Sign(proto.TestNetScheme, sk)
	require.NoError(t, err)
	obj, err := paymentToObject(proto.TestNetScheme, tx)
	require.NoError(t, err)
	obj.id = s.Bytes()
	replaceFirstProof(obj, s)
	return obj
}

func makeReissueTransactionObject(t *testing.T, sig, senderPublicKey, assetID string, quantity, fee, ts int, reissuable bool) rideType {
	s, err := crypto.NewSignatureFromBase58(sig)
	require.NoError(t, err)
	senderPK, err := crypto.NewPublicKeyFromBase58(senderPublicKey)
	require.NoError(t, err)
	asset, err := crypto.NewDigestFromBase58(assetID)
	require.NoError(t, err)
	tx := proto.NewUnsignedReissueWithProofs(2, senderPK, asset, uint64(quantity), reissuable, uint64(ts), uint64(fee))
	sk := crypto.SecretKey{}
	err = tx.Sign(proto.TestNetScheme, sk)
	require.NoError(t, err)
	obj, err := reissueWithProofsToObject(proto.TestNetScheme, tx)
	require.NoError(t, err)
	replaceFirstProof(obj, s)
	return obj
}

func makeBurnTransactionObject(t *testing.T, sig, senderPublicKey, assetID string, amount, fee, ts int) rideType {
	s, err := crypto.NewSignatureFromBase58(sig)
	require.NoError(t, err)
	senderPK, err := crypto.NewPublicKeyFromBase58(senderPublicKey)
	require.NoError(t, err)
	asset, err := crypto.NewDigestFromBase58(assetID)
	require.NoError(t, err)
	tx := proto.NewUnsignedBurnWithProofs(2, senderPK, asset, uint64(amount), uint64(ts), uint64(fee))
	sk := crypto.SecretKey{}
	err = tx.Sign(proto.TestNetScheme, sk)
	require.NoError(t, err)
	obj, err := burnWithProofsToObject(proto.TestNetScheme, tx)
	require.NoError(t, err)
	replaceFirstProof(obj, s)
	return obj
}

func makeMassTransferTransactionObject(t *testing.T, sig, senderPublicKey, optionalAsset string, recipients []string, amounts []int, fee, ts int) rideType {
	require.Equal(t, len(recipients), len(amounts))
	s, err := crypto.NewSignatureFromBase58(sig)
	require.NoError(t, err)
	senderPK, err := crypto.NewPublicKeyFromBase58(senderPublicKey)
	require.NoError(t, err)
	asset, err := proto.NewOptionalAssetFromString(optionalAsset)
	require.NoError(t, err)
	transfers := make([]proto.MassTransferEntry, len(recipients))
	for i := range recipients {
		rcp, err := proto.NewRecipientFromString(recipients[i])
		require.NoError(t, err)
		transfers[i] = proto.MassTransferEntry{Recipient: rcp, Amount: uint64(amounts[i])}
	}
	tx := proto.NewUnsignedMassTransferWithProofs(2, senderPK, *asset, transfers, uint64(fee), uint64(ts), proto.Attachment(""))
	sk := crypto.SecretKey{}
	err = tx.Sign(proto.TestNetScheme, sk)
	require.NoError(t, err)
	obj, err := massTransferWithProofsToObject(proto.TestNetScheme, tx)
	require.NoError(t, err)
	replaceFirstProof(obj, s)
	return obj
}

func makeOrderAndOrderObject(t *testing.T, sig, feeAsset, amountAsset, priceAsset, publicKey string, orderType proto.OrderType, price, amount, ts, expiration, fee int) (proto.Order, rideType) {
	s, err := crypto.NewSignatureFromBase58(sig)
	require.NoError(t, err)
	fa, err := proto.NewOptionalAssetFromString(feeAsset)
	require.NoError(t, err)
	aa, err := proto.NewOptionalAssetFromString(amountAsset)
	require.NoError(t, err)
	pa, err := proto.NewOptionalAssetFromString(priceAsset)
	require.NoError(t, err)
	pk, err := crypto.NewPublicKeyFromBase58(publicKey)
	require.NoError(t, err)
	order := proto.NewUnsignedOrderV3(pk, pk, *aa, *pa, orderType, uint64(price), uint64(amount), uint64(ts), uint64(expiration), uint64(fee), *fa)
	sk := crypto.SecretKey{}
	err = order.Sign(proto.TestNetScheme, sk)
	require.NoError(t, err)
	obj, err := orderToObject(proto.TestNetScheme, order)
	require.NoError(t, err)
	replaceFirstProof(obj, s)
	return order, obj
}

func makeExchangeTransactionObject(t *testing.T, sig, digest string, buy, sell proto.Order, price, amount, buyFee, sellFee, fee, ts int) rideType {
	s, err := crypto.NewSignatureFromBase58(sig)
	require.NoError(t, err)
	d, err := crypto.NewDigestFromBase58(digest)
	require.NoError(t, err)
	tx := proto.NewUnsignedExchangeWithProofs(2, buy, sell, uint64(price), uint64(amount), uint64(buyFee), uint64(sellFee), uint64(fee), uint64(ts))
	sk := crypto.SecretKey{}
	err = tx.Sign(proto.TestNetScheme, sk)
	require.NoError(t, err)
	obj, err := exchangeWithProofsToObject(proto.TestNetScheme, tx)
	require.NoError(t, err)
	replaceFirstProof(obj, s)
	obj.id = d.Bytes()
	obj.bodyBytes = d.Bytes()
	bo := obj.buyOrder
	replaceFirstProof(bo.(rideProven), s)
	so := obj.sellOrder
	replaceFirstProof(so.(rideProven), s)
	return obj
}

func makeTransferTransactionObject(t *testing.T, sig, senderPublicKey, amountAsset, feeAsset, recipient string, amount, fee, ts int) rideType {
	s, err := crypto.NewSignatureFromBase58(sig)
	require.NoError(t, err)
	senderPK, err := crypto.NewPublicKeyFromBase58(senderPublicKey)
	require.NoError(t, err)
	aa, err := proto.NewOptionalAssetFromString(amountAsset)
	require.NoError(t, err)
	fa, err := proto.NewOptionalAssetFromString(feeAsset)
	require.NoError(t, err)
	rcp, err := proto.NewRecipientFromString(recipient)
	require.NoError(t, err)
	tx := proto.NewUnsignedTransferWithProofs(2, senderPK, *aa, *fa, uint64(ts), uint64(amount), uint64(fee), rcp, proto.Attachment(""))
	sk := crypto.SecretKey{}
	err = tx.Sign(proto.TestNetScheme, sk)
	require.NoError(t, err)
	obj, err := transferWithProofsToObject(proto.TestNetScheme, tx)
	require.NoError(t, err)
	replaceFirstProof(obj, s)
	return obj
}

func makeSetScriptTransactionObject(t *testing.T, sig, senderPublicKey, scriptBytes string, fee, ts int, consensusImprovementsActivated bool) rideType {
	s, err := crypto.NewSignatureFromBase58(sig)
	require.NoError(t, err)
	senderPK, err := crypto.NewPublicKeyFromBase58(senderPublicKey)
	require.NoError(t, err)
	var script []byte
	if scriptBytes != "" {
		script, err = base58.Decode(scriptBytes)
		require.NoError(t, err)
	}
	tx := proto.NewUnsignedSetScriptWithProofs(2, senderPK, script, uint64(fee), uint64(ts))
	sk := crypto.SecretKey{}
	err = tx.Sign(proto.TestNetScheme, sk)
	require.NoError(t, err)
	obj, err := setScriptWithProofsToObject(proto.TestNetScheme, consensusImprovementsActivated, tx)
	require.NoError(t, err)
	replaceFirstProof(obj, s)
	return obj
}

func makeSetAssetScriptTransactionObject(t *testing.T, sig, senderPublicKey, assetID, scriptBytes string, fee, ts int) rideType {
	s, err := crypto.NewSignatureFromBase58(sig)
	require.NoError(t, err)
	senderPK, err := crypto.NewPublicKeyFromBase58(senderPublicKey)
	require.NoError(t, err)
	asset, err := crypto.NewDigestFromBase58(assetID)
	require.NoError(t, err)
	script, err := base58.Decode(scriptBytes)
	require.NoError(t, err)
	tx := proto.NewUnsignedSetAssetScriptWithProofs(2, senderPK, asset, script, uint64(fee), uint64(ts))
	sk := crypto.SecretKey{}
	err = tx.Sign(proto.TestNetScheme, sk)
	require.NoError(t, err)
	obj, err := setAssetScriptWithProofsToObject(proto.TestNetScheme, tx)
	require.NoError(t, err)
	replaceFirstProof(obj, s)
	return obj
}

func makeInvokeScriptTransactionAndObject(t *testing.T, sig, senderPublicKey, feeAsset, recipient, functionName, paymentAsset string, fee, ts, arg, paymentAmount int) (*proto.InvokeScriptWithProofs, rideType) {
	s, err := crypto.NewSignatureFromBase58(sig)
	require.NoError(t, err)
	senderPK, err := crypto.NewPublicKeyFromBase58(senderPublicKey)
	require.NoError(t, err)
	fa, err := proto.NewOptionalAssetFromString(feeAsset)
	require.NoError(t, err)
	rcp, err := proto.NewRecipientFromString(recipient)
	require.NoError(t, err)
	fc := proto.FunctionCall{
		Name:      functionName,
		Arguments: proto.Arguments{&proto.IntegerArgument{Value: int64(arg)}},
	}
	pa, err := proto.NewOptionalAssetFromString(paymentAsset)
	require.NoError(t, err)
	ps := []proto.ScriptPayment{{Amount: uint64(paymentAmount), Asset: *pa}}
	tx := proto.NewUnsignedInvokeScriptWithProofs(2, senderPK, rcp, fc, ps, *fa, uint64(fee), uint64(ts))
	sk := crypto.SecretKey{}
	err = tx.Sign(proto.TestNetScheme, sk)
	require.NoError(t, err)
	obj, err := invokeScriptWithProofsToObject(ast.LibV6, proto.TestNetScheme, tx)
	txObj, ok := obj.(rideInvokeScriptTransactionV4)
	require.True(t, ok)
	require.NoError(t, err)
	replaceFirstProof(txObj, s)
	return tx, obj
}

func makeUpdateAssetInfoTransactionObject(t *testing.T, sig, senderPublicKey, feeAsset, assetID, name, description string, fee, ts int) rideType {
	s, err := crypto.NewSignatureFromBase58(sig)
	require.NoError(t, err)
	senderPK, err := crypto.NewPublicKeyFromBase58(senderPublicKey)
	require.NoError(t, err)
	fa, err := proto.NewOptionalAssetFromString(feeAsset)
	require.NoError(t, err)
	asset, err := crypto.NewDigestFromBase58(assetID)
	require.NoError(t, err)
	tx := proto.NewUnsignedUpdateAssetInfoWithProofs(2, asset, senderPK, name, description, uint64(ts), *fa, uint64(fee))
	sk := crypto.SecretKey{}
	err = tx.Sign(proto.TestNetScheme, sk)
	require.NoError(t, err)
	obj, err := updateAssetInfoWithProofsToObject(proto.TestNetScheme, tx)
	require.NoError(t, err)
	replaceFirstProof(obj, s)
	return obj
}

func makeInvokeExpressionTransactionObject(t *testing.T, sig, senderPublicKey, feeAsset, expression string, fee, ts int) rideType {
	s, err := crypto.NewSignatureFromBase58(sig)
	require.NoError(t, err)
	senderPK, err := crypto.NewPublicKeyFromBase58(senderPublicKey)
	require.NoError(t, err)
	fa, err := proto.NewOptionalAssetFromString(feeAsset)
	require.NoError(t, err)
	expr, err := base58.Decode(expression)
	require.NoError(t, err)
	tx := proto.NewUnsignedInvokeExpressionWithProofs(2, senderPK, expr, *fa, uint64(fee), uint64(ts))
	sk := crypto.SecretKey{}
	err = tx.Sign(proto.TestNetScheme, sk)
	require.NoError(t, err)
	obj, err := invokeExpressionWithProofsToObject(proto.TestNetScheme, tx)
	require.NoError(t, err)
	replaceFirstProof(obj, s)
	return obj
}

func makeIssueTransactionAndObject(t *testing.T, sig, senderPublicKey, name, description, scriptBytes string, quantity, decimals, fee, ts int, reissuable bool) (*proto.IssueWithProofs, rideType) {
	s, err := crypto.NewSignatureFromBase58(sig)
	require.NoError(t, err)
	senderPK, err := crypto.NewPublicKeyFromBase58(senderPublicKey)
	require.NoError(t, err)
	script, err := base58.Decode(scriptBytes)
	require.NoError(t, err)
	tx := proto.NewUnsignedIssueWithProofs(2, senderPK, name, description, uint64(quantity), byte(decimals), reissuable, script, uint64(ts), uint64(fee))
	sk := crypto.SecretKey{}
	err = tx.Sign(proto.TestNetScheme, sk)
	require.NoError(t, err)
	obj, err := issueWithProofsToObject(proto.TestNetScheme, tx)
	require.NoError(t, err)
	replaceFirstProof(obj, s)
	return tx, obj
}

func makeLeaseTransactionObject(t *testing.T, sig, senderPublicKey, recipient string, amount, fee, ts int) rideType {
	s, err := crypto.NewSignatureFromBase58(sig)
	require.NoError(t, err)
	senderPK, err := crypto.NewPublicKeyFromBase58(senderPublicKey)
	require.NoError(t, err)
	rcp, err := proto.NewRecipientFromString(recipient)
	require.NoError(t, err)
	tx := proto.NewUnsignedLeaseWithProofs(2, senderPK, rcp, uint64(amount), uint64(fee), uint64(ts))
	sk := crypto.SecretKey{}
	err = tx.Sign(proto.TestNetScheme, sk)
	require.NoError(t, err)
	obj, err := leaseWithProofsToObject(proto.TestNetScheme, tx)
	require.NoError(t, err)
	replaceFirstProof(obj, s)
	return obj
}

func makeLeaseCancelTransactionObject(t *testing.T, sig, senderPublicKey, leaseID string, fee, ts int) rideType {
	s, err := crypto.NewSignatureFromBase58(sig)
	require.NoError(t, err)
	senderPK, err := crypto.NewPublicKeyFromBase58(senderPublicKey)
	require.NoError(t, err)
	lease, err := crypto.NewDigestFromBase58(leaseID)
	require.NoError(t, err)
	tx := proto.NewUnsignedLeaseCancelWithProofs(2, senderPK, lease, uint64(fee), uint64(ts))
	sk := crypto.SecretKey{}
	err = tx.Sign(proto.TestNetScheme, sk)
	require.NoError(t, err)
	obj, err := leaseCancelWithProofsToObject(proto.TestNetScheme, tx)
	require.NoError(t, err)
	replaceFirstProof(obj, s)
	return obj
}

func makeCreateAliasTransactionObject(t *testing.T, sig, senderPublicKey, alias string, fee, ts int) rideType {
	s, err := crypto.NewSignatureFromBase58(sig)
	require.NoError(t, err)
	senderPK, err := crypto.NewPublicKeyFromBase58(senderPublicKey)
	require.NoError(t, err)
	a := proto.NewAlias(proto.TestNetScheme, alias)
	tx := proto.NewUnsignedCreateAliasWithProofs(2, senderPK, *a, uint64(fee), uint64(ts))
	sk := crypto.SecretKey{}
	err = tx.Sign(proto.TestNetScheme, sk)
	require.NoError(t, err)
	obj, err := createAliasWithProofsToObject(proto.TestNetScheme, tx)
	require.NoError(t, err)
	replaceFirstProof(obj, s)
	return obj
}

func makeSponsorFeeTransactionObject(t *testing.T, sig, senderPublicKey, asset string, minAssetFee, fee, ts int) rideType {
	s, err := crypto.NewSignatureFromBase58(sig)
	require.NoError(t, err)
	senderPK, err := crypto.NewPublicKeyFromBase58(senderPublicKey)
	require.NoError(t, err)
	a, err := crypto.NewDigestFromBase58(asset)
	require.NoError(t, err)
	tx := proto.NewUnsignedSponsorshipWithProofs(2, senderPK, a, uint64(minAssetFee), uint64(fee), uint64(ts))
	sk := crypto.SecretKey{}
	err = tx.Sign(proto.TestNetScheme, sk)
	require.NoError(t, err)
	obj, err := sponsorshipWithProofsToObject(proto.TestNetScheme, tx)
	require.NoError(t, err)
	replaceFirstProof(obj, s)
	return obj
}

func makeDataTransactionObject(t *testing.T, sig, senderPublicKey string, keys, values []string, fee, ts int) rideType {
	require.Equal(t, len(keys), len(values))
	s, err := crypto.NewSignatureFromBase58(sig)
	require.NoError(t, err)
	senderPK, err := crypto.NewPublicKeyFromBase58(senderPublicKey)
	require.NoError(t, err)
	tx := proto.NewUnsignedDataWithProofs(2, senderPK, uint64(fee), uint64(ts))
	for i := range keys {
		entry := &proto.StringDataEntry{
			Key:   keys[i],
			Value: values[i],
		}
		err := tx.AppendEntry(entry)
		require.NoError(t, err)
	}
	sk := crypto.SecretKey{}
	err = tx.Sign(proto.TestNetScheme, sk)
	require.NoError(t, err)
	obj, err := dataWithProofsToObject(proto.TestNetScheme, tx)
	require.NoError(t, err)
	replaceFirstProof(obj, s)
	return obj
}

func makeFullAssetInfo(digest crypto.Digest, pk crypto.PublicKey, address proto.WavesAddress, script []byte, tx proto.Transaction) rideType {
	info := &proto.FullAssetInfo{
		AssetInfo: proto.AssetInfo{
			ID:              digest,
			Quantity:        1,
			Decimals:        2,
			Issuer:          address,
			IssuerPublicKey: pk,
			Reissuable:      true,
			Scripted:        true,
			Sponsored:       true,
		},
		Name:        "name",
		Description: "description",
		ScriptInfo: proto.ScriptInfo{
			Version:    3,
			Bytes:      script,
			Base64:     base64.StdEncoding.EncodeToString(script),
			Complexity: 4,
		},
		SponsorshipCost:  5,
		IssueTransaction: tx,
		SponsorBalance:   6,
	}
	return fullAssetInfoToObject(info)
}

func makeBlockInfo(sig []byte, address proto.WavesAddress, pk crypto.PublicKey) rideType {
	info := &proto.BlockInfo{
		Timestamp:           1,
		Height:              2,
		BaseTarget:          3,
		GenerationSignature: sig,
		Generator:           address,
		GeneratorPublicKey:  pk,
		VRF:                 sig,
	}
	return blockInfoToObject(info)
}

func TestTypesStrings(t *testing.T) {
	const (
		a   = "3MbexUVHr88VyDborjw9Veh77MJ68BMNoKi"
		dig = "7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN"
		sig = "3dPbXLoVS7JNAQpdnyYo3fL1GHZCDBPGsTXgQU2wCAKPzMPHqPjJbaBhk9GJqF8mpGcbf4FgUgD1U8owEGg5efv2"
		b64 = "Uv38ByGCZU8WP18PmmIdcpVmx00QA3xNe7sEB9HixkmBhVrYaB0NhtHpHgAWeTnLZpTSxCKs0gigByk5SH9pmeudGKRHhARdh/PGfPInRumVr1olNnlRuqL/bNRxxIPxX7kLrbN8WCG22VUmpBqVBGgLTnyLdjobHUnUlVyEhiFjJSU/7HON16nii/khEZwWDwcCRIYVu9oIMT9qjrZo0gv1BZh1kh5milvfLH/EhEWS0lcrzQZo0tbFL1BU4tCDa/hMcXTLdHY2TMPb2Wiw9xcu2FeUuzWLDDtSXaF4b5//CUJ52xlE69ehnQ97usvgJVqlt9RL7ED4TIkrm//UNimwIjvupfT3Q5H0RdFa/UKUBAN09pJLmMv4cT+Nli18jQGRksJCJOLK/Mrjph+1hrFDI6a8j5598dkpMz/5k5M76m9bOvbeA3Q2bEcZ5DobBn2JvH8B8fVzmBZZpE/xekxyFaO1OeseWEnGB327VyL1cXoomiZvl2R5gZmOvqicC0s3OXARXoLtb0ElyPpzEeTX3vqSLarneGZn9+k2zU8kq/ffhmuqVgODZ61hRd4e6PSosJk+vfiIOgrYvpw5eLBIg+VqFWqN5WOvpGfUnexqQOmh0AfwM8KCMGG90Oqln45NpkMBBSINCyloi3NLjqDzypk26EYfENd8luqAp6Zl9gb2pjt/Pf0lZ8GJeeTWDyZobZvy+ybJAf81TN4WB+4pSznzK3x4Irpk+Eq0PKDG5rkcH9O+iZBDQXnTr0SRo2kBLbktGE/DnRc0/1cWQolTu2hl/PkrDDoXyQKL6ZFOt2ScbJNHgAl50YMDVvKlTD3qsqS0R11jr76PtWmHx39YGFJvGBS+gjNQ6rE5NfMdhEhFF+kkrveK4VHAB1WSWDa3B1iFZQww7CmjcDk0v1CijaECl13tp351hXnqPf5BNqv3UrO4Jx0D6USzyds2a3UEX479adIq5UEZR8tVPXaUJnrvTrzqQGsy1hCL1oWE9X43yqxuM/6qMmOjmUNwJLqcmxRniidPAakQrilfbvv+X1q/RMzeJjtWBmM+K/AAbygpXX05Bp8BojnENlhUw69/a0HWMfkrmo0S9BJXMl//My91drBiBVYwSj4+rhTCjQzqOdKQGlJyDahcoeSzjq8/RMbG74Ni8vVPwA4J1vwlZAhUwV38rKqKLOzOWjq6U6twWxjblLTTOKUUPmNAjYcksM8/rhej95vhBy+2PDXWBCxBYPOO6eKp8/tP+wAZtFTVIrX/oXYEGT+4lmcQp5YHMspSz1PD9SDIibeb9QTPtXx2ASMtWJuszqnW4mPiXCd0HT9sYsu7FdmvvL9/faQasECOOWnC4s3PIzQ4vxd0rOdwmk8JHpqD/erg7FXrIzqbU5TLPHhWtUbTE8ijtMHA4FRH9Lo3DrNtvP3skLMC3Nw7nvUi4qbx7Qr+wfjiD6q+32sWLnF9OnSKWGd6DFY0j4khomaxHQ8zTGL+UrpTrxl3nLKUi2Vw/6C3c5Y8EwrXl93q/k460ptRJSEPDvHDFAkPB8eab1ccJG8+msC3QT7xEL1YsAznO/9wb3/0tvRAkKMnEfMgjk5LictRZc5kACy9nCiHqhE98kaJKNWiO5ynQPgMk4LZxgNK0pYMeQ=="
	)
	shortBytes, err := base58.Decode(a)
	require.NoError(t, err)
	longBytes, err := base64.StdEncoding.DecodeString(b64)
	require.NoError(t, err)
	shortRideBytes := rideBytes(shortBytes)
	longRideBytes := rideBytes(longBytes)
	ad, err := proto.NewAddressFromBytes(shortRideBytes)
	require.NoError(t, err)
	testAddress := rideAddress(ad)
	al := *proto.NewAlias(proto.TestNetScheme, "str")
	testAlias := rideAlias(al)
	pk, err := crypto.NewPublicKeyFromBase58(dig)
	require.NoError(t, err)
	recipient1 := proto.NewRecipientFromAddress(ad)
	recipient2 := proto.NewRecipientFromAlias(al)
	testAsset1, err := proto.NewOptionalAssetFromString(dig)
	require.NoError(t, err)
	testAsset2, err := proto.NewOptionalAssetFromString("WAVES")
	require.NoError(t, err)
	d, err := crypto.NewDigestFromBase58(dig)
	require.NoError(t, err)
	testBalanceDetails := &proto.FullWavesBalance{Regular: 1, Generating: 2, Available: 3, Effective: 4, LeaseIn: 5, LeaseOut: 6}
	testBooleanEntry := &proto.BooleanDataEntry{Key: "key", Value: true}
	testIntEntry := &proto.IntegerDataEntry{Key: "key", Value: 1}
	testStringEntry := &proto.StringDataEntry{Key: "key", Value: "value"}
	testBinaryEntry := &proto.BinaryDataEntry{Key: "key", Value: shortBytes}
	testDeleteEntry := &proto.DeleteDataEntry{Key: "key"}
	testAttachedPayment := proto.ScriptPayment{Amount: 1, Asset: *testAsset1}
	testScriptTransfer := &proto.FullScriptTransfer{Amount: 1, Asset: *testAsset1, Recipient: recipient1, Sender: ad, SenderPK: pk, Timestamp: 2, ID: &d}

	buy, testOrder1 := makeOrderAndOrderObject(t, sig, dig, dig, "WAVES", dig, proto.Buy, 1, 2, 3, 4, 5)
	testGenesisTransaction := makeGenesisTransactionObject(t, sig, a, 1, 2)
	testPaymentTransaction := makePaymentTransactionObject(t, sig, dig, a, 1, 2, 3)
	testReissueTransaction := makeReissueTransactionObject(t, sig, dig, dig, 1, 2, 3, true)
	testBurnTransaction := makeBurnTransactionObject(t, sig, dig, dig, 1, 2, 3)
	testMassTransferTransaction := makeMassTransferTransactionObject(t, sig, dig, dig, []string{a}, []int{1}, 2, 3)
	sell, _ := makeOrderAndOrderObject(t, sig, dig, dig, "WAVES", dig, proto.Sell, 1, 2, 3, 4, 5)
	testExchangeTransaction := makeExchangeTransactionObject(t, sig, dig, buy, sell, 1, 2, 3, 4, 5, 6)
	testTransferTransaction := makeTransferTransactionObject(t, sig, dig, dig, dig, a, 1, 2, 3)
	testSetAssetScriptTransaction := makeSetAssetScriptTransactionObject(t, sig, dig, dig, dig, 1, 2)
	inv, testInvokeScriptTransaction := makeInvokeScriptTransactionAndObject(t, sig, dig, dig, a, "str", dig, 1, 2, 3, 4)
	invObj, err := invocationToObject(ast.LibV5, proto.TestNetScheme, inv)
	require.NoError(t, err)
	testUpdateAssetInfoTransaction := makeUpdateAssetInfoTransactionObject(t, sig, dig, dig, dig, "str", "description", 1, 2)
	testInvokeExpressionTransaction := makeInvokeExpressionTransactionObject(t, sig, dig, dig, dig, 1, 2)
	itx, testIssueTransaction := makeIssueTransactionAndObject(t, sig, dig, "name", "description", dig, 1, 2, 3, 4, true)
	testLeaseTransaction := makeLeaseTransactionObject(t, sig, dig, a, 1, 2, 3)
	testLeaseCancelTransaction := makeLeaseCancelTransactionObject(t, sig, dig, dig, 1, 2)
	testCreateAliasTransaction := makeCreateAliasTransactionObject(t, sig, dig, "str", 1, 2)
	testSetScriptTransaction := makeSetScriptTransactionObject(t, sig, dig, dig, 1, 2, false)
	testSponsorFeeTransaction := makeSponsorFeeTransactionObject(t, sig, dig, dig, 1, 2, 3)
	testDataTransaction := makeDataTransactionObject(t, sig, dig, []string{"key"}, []string{"value"}, 1, 2)
	testAssetInfo := makeFullAssetInfo(d, pk, ad, longBytes, itx)
	testBlockInfo := makeBlockInfo(shortBytes, ad, pk)
	testIssueAction := newRideIssue(rideUnit{}, "name", "description", 1, 2, 3, true)
	testReissueAction := newRideReissue(shortBytes, 1, true)
	testBurnAction := newRideBurn(shortBytes, 1)
	testSponsorFee := newRideSponsorFee(shortBytes, 1)
	testLease := newRideLease(recipientToObject(recipient1), 1, 2)
	testLeaseCancel := newRideLeaseCancel(shortBytes)
	for _, test := range []struct {
		v rideType
		s string
	}{
		{rideBoolean(true), "true"},
		{rideBoolean(false), "false"},
		{rideInt(-12345), "-12345"},
		{rideInt(67890), "67890"},
		{rideBigInt{v: big.NewInt(-12345)}, "-12345"},
		{rideBigInt{v: big.NewInt(67890)}, "67890"},
		{rideString(""), "\"\""},
		{rideString("xxx"), "\"xxx\""},
		{shortRideBytes, "base58'3MbexUVHr88VyDborjw9Veh77MJ68BMNoKi'"},
		{longRideBytes, "base64'Uv38ByGCZU8WP18PmmIdcpVmx00QA3xNe7sEB9HixkmBhVrYaB0NhtHpHgAWeTnLZpTSxCKs0gigByk5SH9pmeudGKRHhARdh/PGfPInRumVr1olNnlRuqL/bNRxxIPxX7kLrbN8WCG22VUmpBqVBGgLTnyLdjobHUnUlVyEhiFjJSU/7HON16nii/khEZwWDwcCRIYVu9oIMT9qjrZo0gv1BZh1kh5milvfLH/EhEWS0lcrzQZo0tbFL1BU4tCDa/hMcXTLdHY2TMPb2Wiw9xcu2FeUuzWLDDtSXaF4b5//CUJ52xlE69ehnQ97usvgJVqlt9RL7ED4TIkrm//UNimwIjvupfT3Q5H0RdFa/UKUBAN09pJLmMv4cT+Nli18jQGRksJCJOLK/Mrjph+1hrFDI6a8j5598dkpMz/5k5M76m9bOvbeA3Q2bEcZ5DobBn2JvH8B8fVzmBZZpE/xekxyFaO1OeseWEnGB327VyL1cXoomiZvl2R5gZmOvqicC0s3OXARXoLtb0ElyPpzEeTX3vqSLarneGZn9+k2zU8kq/ffhmuqVgODZ61hRd4e6PSosJk+vfiIOgrYvpw5eLBIg+VqFWqN5WOvpGfUnexqQOmh0AfwM8KCMGG90Oqln45NpkMBBSINCyloi3NLjqDzypk26EYfENd8luqAp6Zl9gb2pjt/Pf0lZ8GJeeTWDyZobZvy+ybJAf81TN4WB+4pSznzK3x4Irpk+Eq0PKDG5rkcH9O+iZBDQXnTr0SRo2kBLbktGE/DnRc0/1cWQolTu2hl/PkrDDoXyQKL6ZFOt2ScbJNHgAl50YMDVvKlTD3qsqS0R11jr76PtWmHx39YGFJvGBS+gjNQ6rE5NfMdhEhFF+kkrveK4VHAB1WSWDa3B1iFZQww7CmjcDk0v1CijaECl13tp351hXnqPf5BNqv3UrO4Jx0D6USzyds2a3UEX479adIq5UEZR8tVPXaUJnrvTrzqQGsy1hCL1oWE9X43yqxuM/6qMmOjmUNwJLqcmxRniidPAakQrilfbvv+X1q/RMzeJjtWBmM+K/AAbygpXX05Bp8BojnENlhUw69/a0HWMfkrmo0S9BJXMl//My91drBiBVYwSj4+rhTCjQzqOdKQGlJyDahcoeSzjq8/RMbG74Ni8vVPwA4J1vwlZAhUwV38rKqKLOzOWjq6U6twWxjblLTTOKUUPmNAjYcksM8/rhej95vhBy+2PDXWBCxBYPOO6eKp8/tP+wAZtFTVIrX/oXYEGT+4lmcQp5YHMspSz1PD9SDIibeb9QTPtXx2ASMtWJuszqnW4mPiXCd0HT9sYsu7FdmvvL9/faQasECOOWnC4s3PIzQ4vxd0rOdwmk8JHpqD/erg7FXrIzqbU5TLPHhWtUbTE8ijtMHA4FRH9Lo3DrNtvP3skLMC3Nw7nvUi4qbx7Qr+wfjiD6q+32sWLnF9OnSKWGd6DFY0j4khomaxHQ8zTGL+UrpTrxl3nLKUi2Vw/6C3c5Y8EwrXl93q/k460ptRJSEPDvHDFAkPB8eab1ccJG8+msC3QT7xEL1YsAznO/9wb3/0tvRAkKMnEfMgjk5LictRZc5kACy9nCiHqhE98kaJKNWiO5ynQPgMk4LZxgNK0pYMeQ=='"},
		{rideUnit{}, "Unit"},
		{rideList{rideBoolean(true), rideInt(1), rideString("x"), rideUnit{}, shortRideBytes}, "[true, 1, \"x\", Unit, base58'3MbexUVHr88VyDborjw9Veh77MJ68BMNoKi']"},
		{newDown(nil), "Down"},
		{newHalfUp(nil), "HalfUp"},
		{newHalfEven(nil), "HalfEven"},
		{newCeiling(nil), "Ceiling"},
		{newFloor(nil), "Floor"},
		{newNoAlg(nil), "NoAlg"},
		{newMd5(nil), "Md5"},
		{newSha1(nil), "Sha1"},
		{newSha224(nil), "Sha224"},
		{newSha256(nil), "Sha256"},
		{newSha384(nil), "Sha384"},
		{newSha512(nil), "Sha512"},
		{newSha3224(nil), "Sha3224"},
		{newSha3256(nil), "Sha3256"},
		{newSha3384(nil), "Sha3384"},
		{newSha3512(nil), "Sha3512"},
		{rideList{rideBoolean(true)}, "[true]"},
		{rideList{}, "[]"},
		{testAddress, "Address(\n\tbytes = base58'3MbexUVHr88VyDborjw9Veh77MJ68BMNoKi'\n)"},
		{testAlias, "Alias(\n\talias = \"str\"\n)"},
		{recipientToObject(recipient1), "Address(\n\tbytes = base58'3MbexUVHr88VyDborjw9Veh77MJ68BMNoKi'\n)"},
		{recipientToObject(recipient2), "Alias(\n\talias = \"str\"\n)"},
		{transferEntryToObject(proto.MassTransferEntry{Recipient: recipient1, Amount: 1}), "Transfer(\n\trecipient = Address(\n\t\tbytes = base58'3MbexUVHr88VyDborjw9Veh77MJ68BMNoKi'\n\t)\n\tamount = 1\n)"},
		{assetPairToObject(*testAsset1, *testAsset2), "AssetPair(\n\tamountAsset = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tpriceAsset = Unit\n)"},
		{balanceDetailsToObject(testBalanceDetails), "BalanceDetails(\n\tavailable = 3\n\tregular = 1\n\tgenerating = 2\n\teffective = 4\n)"},
		{dataEntryToObject(testBooleanEntry), "BooleanEntry(\n\tkey = \"key\"\n\tvalue = true\n)"},
		{dataEntryToObject(testIntEntry), "IntegerEntry(\n\tkey = \"key\"\n\tvalue = 1\n)"},
		{dataEntryToObject(testStringEntry), "StringEntry(\n\tkey = \"key\"\n\tvalue = \"value\"\n)"},
		{dataEntryToObject(testBinaryEntry), "BinaryEntry(\n\tkey = \"key\"\n\tvalue = base58'3MbexUVHr88VyDborjw9Veh77MJ68BMNoKi'\n)"},
		{dataEntryToObject(testDeleteEntry), "DeleteEntry(\n\tkey = \"key\"\n)"},
		{attachedPaymentToObject(testAttachedPayment), "AttachedPayment(\n\tassetId = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tamount = 1\n)"},
		{scriptTransferToObject(testScriptTransfer), "ScriptTransfer(\n\trecipient = Address(\n\t\tbytes = base58'3MbexUVHr88VyDborjw9Veh77MJ68BMNoKi'\n\t)\n\tamount = 1\n\tasset = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n)"},
		{testOrder1, "Order(\n\tassetPair = AssetPair(\n\t\tamountAsset = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\t\tpriceAsset = Unit\n\t)\n\ttimestamp = 3\n\tbodyBytes = base58'3CcpxchsneBag6wghTFYXs5NyBYG6XkBTH6KgjtMNBSkFkd15L2NVRfG5p6w5ruKNXLiMRRshdiJ2SP2mjA9TgZSiEGm5qS944qm9cY91gaZYcb9kutErrtZJ8pgq1SqtbtTqtNzz81TJEnewoLfRmoAsBEgVtEr7fPnAh6ytF2KQtHiFCyKUiJjtYwsp9gDri5hWaHjMsN7udhmsLxwTzes52HezaJyw5RUG7fWkvoY'\n\tamount = 2\n\tmatcherFeeAssetId = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tid = base58'9JekSZhdMviY41ibFK3BoReUN3q8PAeRCEykK1a4BVmX'\n\tsenderPublicKey = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tmatcherPublicKey = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tsender = Address(\n\t\tbytes = base58'3NAG3ZUW9iH53gVNcvCRRSrbBYcJ27jm5ow'\n\t)\n\torderType = Buy\n\tproofs = [base58'3dPbXLoVS7JNAQpdnyYo3fL1GHZCDBPGsTXgQU2wCAKPzMPHqPjJbaBhk9GJqF8mpGcbf4FgUgD1U8owEGg5efv2', base58'', base58'', base58'', base58'', base58'', base58'', base58'']\n\texpiration = 4\n\tmatcherFee = 5\n\tprice = 1\n)"},
		{testGenesisTransaction, "GenesisTransaction(\n\trecipient = Address(\n\t\tbytes = base58'3MbexUVHr88VyDborjw9Veh77MJ68BMNoKi'\n\t)\n\ttimestamp = 2\n\tamount = 1\n\tversion = 1\n\tid = base58'3dPbXLoVS7JNAQpdnyYo3fL1GHZCDBPGsTXgQU2wCAKPzMPHqPjJbaBhk9GJqF8mpGcbf4FgUgD1U8owEGg5efv2'\n\tfee = 0\n)"},
		{testPaymentTransaction, "PaymentTransaction(\n\trecipient = Address(\n\t\tbytes = base58'3MbexUVHr88VyDborjw9Veh77MJ68BMNoKi'\n\t)\n\ttimestamp = 3\n\tbodyBytes = base58'1112shAp7eMKU23Q8YzJ43GiPCRqPPmtAzVTCFxadJrxXQnodUM99Riffg5JkqoWDFkAMQcoYYYshBhgDZVeNvFvdAd64ciTJYNBbRTagFrUSpBJedaZ'\n\tamount = 1\n\tversion = 1\n\tid = base58'3dPbXLoVS7JNAQpdnyYo3fL1GHZCDBPGsTXgQU2wCAKPzMPHqPjJbaBhk9GJqF8mpGcbf4FgUgD1U8owEGg5efv2'\n\tsenderPublicKey = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tsender = Address(\n\t\tbytes = base58'3NAG3ZUW9iH53gVNcvCRRSrbBYcJ27jm5ow'\n\t)\n\tproofs = [base58'3dPbXLoVS7JNAQpdnyYo3fL1GHZCDBPGsTXgQU2wCAKPzMPHqPjJbaBhk9GJqF8mpGcbf4FgUgD1U8owEGg5efv2', base58'', base58'', base58'', base58'', base58'', base58'', base58'']\n\tfee = 2\n)"},
		{testReissueTransaction, "ReissueTransaction(\n\tquantity = 1\n\ttimestamp = 3\n\tbodyBytes = base58'GJ4XLXXyCGcq6ow2CkcCJSPUCSqh1Ei3CBiYF9umQ47XM766TiRrMLsa1xEDtegJ5T3jRYHjMJeeaaguvdF7rCrHUWC7WzrqYaEfEX9dcPUQyrBuQWvLGCe81qHrA'\n\tassetId = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tversion = 2\n\tid = base58'6p2iV3mPe4xRATv99JHk7ub7uS9npK3vyjFdpg3QUzhS'\n\tsenderPublicKey = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tsender = Address(\n\t\tbytes = base58'3NAG3ZUW9iH53gVNcvCRRSrbBYcJ27jm5ow'\n\t)\n\tproofs = [base58'3dPbXLoVS7JNAQpdnyYo3fL1GHZCDBPGsTXgQU2wCAKPzMPHqPjJbaBhk9GJqF8mpGcbf4FgUgD1U8owEGg5efv2', base58'', base58'', base58'', base58'', base58'', base58'', base58'']\n\treissuable = true\n\tfee = 2\n)"},
		{testBurnTransaction, "BurnTransaction(\n\tquantity = 1\n\ttimestamp = 3\n\tbodyBytes = base58'5A6c1omsWLwHrvWm8dAVVxMB7wjVAN5VXCypjek7fFHePWJ651W9mSDe7sd59BFzuYD32toafAFA8G36yCiwv1SuZbfztmRc8w5LaWo1pGsR8wAztXA61sSroovi'\n\tassetId = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tversion = 2\n\tid = base58'2nbKo4ph2dTvNNJBPQXkksghQg6jVg23SVoV6rzDkiYR'\n\tsenderPublicKey = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tsender = Address(\n\t\tbytes = base58'3NAG3ZUW9iH53gVNcvCRRSrbBYcJ27jm5ow'\n\t)\n\tproofs = [base58'3dPbXLoVS7JNAQpdnyYo3fL1GHZCDBPGsTXgQU2wCAKPzMPHqPjJbaBhk9GJqF8mpGcbf4FgUgD1U8owEGg5efv2', base58'', base58'', base58'', base58'', base58'', base58'', base58'']\n\tfee = 2\n)"},
		{testMassTransferTransaction, "MassTransferTransaction(\n\ttransferCount = 1\n\ttimestamp = 3\n\tbodyBytes = base58'24B4RzcMGAcuzzKrNxD3VLHA4KUeTVwAUPpWKhGZ9PcXKm8GRheVtP4MA7n5LhMQbYVG66keQCmREqoU5DZiduDYvUep81bAyzxJcMKuBV45nk2UqtRSnzUMwkscjHBpUTznc2eTGGVcjRtbGAhgt'\n\tassetId = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tid = base58'Bqzpqsrc3XUfLi8U6m1nPztV4vsRsFXvrvYcfHhSHaAm'\n\tsenderPublicKey = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tattachment = base58''\n\tsender = Address(\n\t\tbytes = base58'3NAG3ZUW9iH53gVNcvCRRSrbBYcJ27jm5ow'\n\t)\n\ttransfers = [Transfer(\n\trecipient = Address(\n\t\tbytes = base58'3MbexUVHr88VyDborjw9Veh77MJ68BMNoKi'\n\t)\n\tamount = 1\n)]\n\tproofs = [base58'3dPbXLoVS7JNAQpdnyYo3fL1GHZCDBPGsTXgQU2wCAKPzMPHqPjJbaBhk9GJqF8mpGcbf4FgUgD1U8owEGg5efv2', base58'', base58'', base58'', base58'', base58'', base58'', base58'']\n\tfee = 2\n\ttotalAmount = 1\n\tversion = 2\n)"},
		{testExchangeTransaction, "ExchangeTransaction(\n\ttimestamp = 6\n\tbodyBytes = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tbuyOrder = Order(\n\t\tassetPair = AssetPair(\n\t\t\tamountAsset = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\t\t\tpriceAsset = Unit\n\t\t)\n\t\ttimestamp = 3\n\t\tbodyBytes = base58'3CcpxchsneBag6wghTFYXs5NyBYG6XkBTH6KgjtMNBSkFkd15L2NVRfG5p6w5ruKNXLiMRRshdiJ2SP2mjA9TgZSiEGm5qS944qm9cY91gaZYcb9kutErrtZJ8pgq1SqtbtTqtNzz81TJEnewoLfRmoAsBEgVtEr7fPnAh6ytF2KQtHiFCyKUiJjtYwsp9gDri5hWaHjMsN7udhmsLxwTzes52HezaJyw5RUG7fWkvoY'\n\t\tamount = 2\n\t\tmatcherFeeAssetId = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\t\tid = base58'9JekSZhdMviY41ibFK3BoReUN3q8PAeRCEykK1a4BVmX'\n\t\tsenderPublicKey = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\t\tmatcherPublicKey = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\t\tsender = Address(\n\t\t\tbytes = base58'3NAG3ZUW9iH53gVNcvCRRSrbBYcJ27jm5ow'\n\t\t)\n\t\torderType = Buy\n\t\tproofs = [base58'3dPbXLoVS7JNAQpdnyYo3fL1GHZCDBPGsTXgQU2wCAKPzMPHqPjJbaBhk9GJqF8mpGcbf4FgUgD1U8owEGg5efv2', base58'', base58'', base58'', base58'', base58'', base58'', base58'']\n\t\texpiration = 4\n\t\tmatcherFee = 5\n\t\tprice = 1\n\t)\n\tprice = 1\n\tamount = 2\n\tversion = 2\n\tid = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tsellOrder = Order(\n\t\tassetPair = AssetPair(\n\t\t\tamountAsset = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\t\t\tpriceAsset = Unit\n\t\t)\n\t\ttimestamp = 3\n\t\tbodyBytes = base58'3CcpxchsneBag6wghTFYXs5NyBYG6XkBTH6KgjtMNBSkFkd15L2NVRfG5p6w5ruKNXLiMRRshdiJ2SP2mjA9TgZSiEGm5qS944qm9cY91gaZYcb9kutErrtZJ8pgq1SqtbtTqtP1GoLXEbMEVcEwDCBPbQ7b4GRfFQiMvY4mF3XYFW73z5hmkD9HSKa5aAjtkoQdRVgKhZwumpx1dqmT6uth1jgWA2mvathB5tDXXoXt'\n\t\tamount = 2\n\t\tmatcherFeeAssetId = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\t\tid = base58'H9ZJ2bvXRnnCyE1o5N6V9YUFoPWVLVUCaqLW1f7vFDDR'\n\t\tsenderPublicKey = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\t\tmatcherPublicKey = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\t\tsender = Address(\n\t\t\tbytes = base58'3NAG3ZUW9iH53gVNcvCRRSrbBYcJ27jm5ow'\n\t\t)\n\t\torderType = Sell\n\t\tproofs = [base58'3dPbXLoVS7JNAQpdnyYo3fL1GHZCDBPGsTXgQU2wCAKPzMPHqPjJbaBhk9GJqF8mpGcbf4FgUgD1U8owEGg5efv2', base58'', base58'', base58'', base58'', base58'', base58'', base58'']\n\t\texpiration = 4\n\t\tmatcherFee = 5\n\t\tprice = 1\n\t)\n\tsenderPublicKey = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tbuyMatcherFee = 3\n\tsender = Address(\n\t\tbytes = base58'3NAG3ZUW9iH53gVNcvCRRSrbBYcJ27jm5ow'\n\t)\n\tfee = 5\n\tproofs = [base58'3dPbXLoVS7JNAQpdnyYo3fL1GHZCDBPGsTXgQU2wCAKPzMPHqPjJbaBhk9GJqF8mpGcbf4FgUgD1U8owEGg5efv2', base58'', base58'', base58'', base58'', base58'', base58'', base58'']\n\tsellMatcherFee = 4\n)"},
		{testTransferTransaction, "TransferTransaction(\n\trecipient = Address(\n\t\tbytes = base58'3MbexUVHr88VyDborjw9Veh77MJ68BMNoKi'\n\t)\n\ttimestamp = 3\n\tbodyBytes = base58'AaMaJ1R45gFjJYUoZvpVxwbB4RQQndLofZyVfQEWJsydtCQs8Frwt1xiseTP83yFuqUJyebcALgxfxSVvAzhhckAKEJGyXi9Yzp2LG64SZkjj19dvwqkxb29g882KvwjmCq7FVxmpxa3onPxktK3ECnp1Pd35XZdVMsUNwk4sM2YZyxhTboP9YpUDivpjjyvy63mqsEvoYrwMi7'\n\tassetId = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tfeeAssetId = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tamount = 1\n\tversion = 2\n\tid = base58'6Y23BPbMY6d2GYnSaGNnaakYpTqpJZejyedr2ZUzxtN'\n\tsenderPublicKey = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tattachment = base58''\n\tsender = Address(\n\t\tbytes = base58'3NAG3ZUW9iH53gVNcvCRRSrbBYcJ27jm5ow'\n\t)\n\tproofs = [base58'3dPbXLoVS7JNAQpdnyYo3fL1GHZCDBPGsTXgQU2wCAKPzMPHqPjJbaBhk9GJqF8mpGcbf4FgUgD1U8owEGg5efv2', base58'', base58'', base58'', base58'', base58'', base58'', base58'']\n\tfee = 2\n)"},
		{testSetAssetScriptTransaction, "SetAssetScriptTransaction(\n\ttimestamp = 2\n\tbodyBytes = base58'3KTRSLxaUZ4SCR4mzzQA1BfSUmz4XNbDNdiEfRfBvSimumQAcA7VHsdW8mzDyoHGCHHu96NoUgG1f6VY1vtpaAtzwZY2xZdXVeXCZVCvXSTDRWtjju2JD16NwgQoWvfT1qihJszNT7KmTe7jHtsxr66sCjQkE'\n\tassetId = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tversion = 2\n\tid = base58'Ci27s3TpmDW2f2gQr1CMsTvc1usB6pmq8t7CCjdhKw4A'\n\tsenderPublicKey = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tsender = Address(\n\t\tbytes = base58'3NAG3ZUW9iH53gVNcvCRRSrbBYcJ27jm5ow'\n\t)\n\tscript = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tproofs = [base58'3dPbXLoVS7JNAQpdnyYo3fL1GHZCDBPGsTXgQU2wCAKPzMPHqPjJbaBhk9GJqF8mpGcbf4FgUgD1U8owEGg5efv2', base58'', base58'', base58'', base58'', base58'', base58'', base58'']\n\tfee = 1\n)"},
		{testInvokeScriptTransaction, "InvokeScriptTransaction(\n\tpayments = [AttachedPayment(\n\tassetId = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tamount = 4\n)]\n\ttimestamp = 2\n\tbodyBytes = base58'BqknDErU7qiYLzrCjmd16o5CGAYhmpYBLjmogoxC9nbrMHzHSnTUoHsz5Pk18AwKV1DjQcwmpiJ8aE8RwRsVgm2gEgARNBGovKxn5CEayEnEkLr2LeDKabFk1AnNCBRen87kakN9vWW75y9XsHFKZwBCYpXWafTNrm1xUo2E2cGkRcQtVx4R1zGWs3esyctyzpPCwGgx6uWfEp615RNZ5QMwYoRgYC2mYEy55'\n\tfeeAssetId = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tid = base58'66V9DMnGSaKuHWWLA62W2kG7kSqmpCYmbsdsraHQGATT'\n\tproofs = [base58'3dPbXLoVS7JNAQpdnyYo3fL1GHZCDBPGsTXgQU2wCAKPzMPHqPjJbaBhk9GJqF8mpGcbf4FgUgD1U8owEGg5efv2', base58'', base58'', base58'', base58'', base58'', base58'', base58'']\n\tfee = 1\n\tdApp = Address(\n\t\tbytes = base58'3MbexUVHr88VyDborjw9Veh77MJ68BMNoKi'\n\t)\n\tversion = 2\n\tsenderPublicKey = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tfunction = \"str\"\n\tsender = Address(\n\t\tbytes = base58'3NAG3ZUW9iH53gVNcvCRRSrbBYcJ27jm5ow'\n\t)\n\targs = [3]\n)"},
		{testUpdateAssetInfoTransaction, "UpdateAssetInfoTransaction(\n\tname = \"str\"\n\ttimestamp = 2\n\tbodyBytes = base58'RcfZAaBsnn93PsV9yUQLyCHsogShp8rYwxFw8H5X72zNwsiNYS1mhDNkXMNwXNBAZ8KhAkVhTfeSE7VEmdyxCEV7q1CS23z68FdVrtGYD2oz44E5gsb7CvbEVcZJtF9DPGovcY4jnK4wsWwFpDWdx4rA8meYYX1JYC63bDPvm9unREQZ4unCD'\n\tassetId = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tdescription = \"description\"\n\tversion = 2\n\tid = base58'7aYJDXzTHQEcdYDVXV2wYLXENnV2NcsDgoDjHB75sXoA'\n\tsenderPublicKey = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tsender = Address(\n\t\tbytes = base58'3NAG3ZUW9iH53gVNcvCRRSrbBYcJ27jm5ow'\n\t)\n\tproofs = [base58'3dPbXLoVS7JNAQpdnyYo3fL1GHZCDBPGsTXgQU2wCAKPzMPHqPjJbaBhk9GJqF8mpGcbf4FgUgD1U8owEGg5efv2', base58'', base58'', base58'', base58'', base58'', base58'', base58'']\n\tfee = 1\n)"},
		{testInvokeExpressionTransaction, "InvokeExpressionTransaction(\n\ttimestamp = 2\n\tbodyBytes = base58'3KTRSLxaUZ4SCR4mzzQA1BfSUmz4XNbDNdiEfRfBvSimumQAcA8bDyUhvDnyuZDki9CHNeZmzABzzuT7oeqhEjtiVuEaYiNvWMfoAvApqRKsinx7aDUHKh4uriuUdaYAtLEtwGRckR5cZkiHbjNY1KxM6YbvN'\n\tfeeAssetId = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tversion = 2\n\tid = base58'8Z8JfoT7YvSja5jGMD8tnDZhqSNBDeUpWnF3ri5EWM4J'\n\texpression = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tsenderPublicKey = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tsender = Address(\n\t\tbytes = base58'3NAG3ZUW9iH53gVNcvCRRSrbBYcJ27jm5ow'\n\t)\n\tproofs = [base58'3dPbXLoVS7JNAQpdnyYo3fL1GHZCDBPGsTXgQU2wCAKPzMPHqPjJbaBhk9GJqF8mpGcbf4FgUgD1U8owEGg5efv2', base58'', base58'', base58'', base58'', base58'', base58'', base58'']\n\tfee = 1\n)"},
		{testIssueTransaction, "IssueTransaction(\n\ttimestamp = 4\n\tbodyBytes = base58'qacgpxL5ZmV5CWZ9hzPog9hxMhUvLqTr5ubuMsNdYCzxeWz7jwtfxYSN5avCipcLVCFXAjd5uspHtFgA7sZkgXWwzwDSSjvoShiRiYJGugsiE4JJYmw3Ld2w7Ttm2xifgdE1KFNAToUSq7dWfPPtG5TDo1W6'\n\tdescription = \"description\"\n\tversion = 2\n\tid = base58'GCJUDXGnyd6hUYcagb4xJB3Zpz459Sa79jWosaXcVV1B'\n\tsenderPublicKey = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tsender = Address(\n\t\tbytes = base58'3NAG3ZUW9iH53gVNcvCRRSrbBYcJ27jm5ow'\n\t)\n\tscript = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\treissuable = true\n\tfee = 3\n\tname = \"name\"\n\tquantity = 1\n\tproofs = [base58'3dPbXLoVS7JNAQpdnyYo3fL1GHZCDBPGsTXgQU2wCAKPzMPHqPjJbaBhk9GJqF8mpGcbf4FgUgD1U8owEGg5efv2', base58'', base58'', base58'', base58'', base58'', base58'', base58'']\n\tdecimals = 2\n)"},
		{testLeaseTransaction, "LeaseTransaction(\n\trecipient = Address(\n\t\tbytes = base58'3MbexUVHr88VyDborjw9Veh77MJ68BMNoKi'\n\t)\n\ttimestamp = 3\n\tbodyBytes = base58'3XBHZM57sJ86TGVLjKqQSxWJesoQJyUQFiUEBZBE3jvTXGbMVKPTPA8VTJHe2n5cEbscAdpMkAKW6pHJy5Y5GMFqSZcFGe95PFww2Z7HrUipEDyM6i8a'\n\tamount = 1\n\tversion = 2\n\tid = base58'FxdMDTYD1fQAzSKHTiSYMy3j4ELpJ3FFZiyQK3AkHfUq'\n\tsenderPublicKey = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tsender = Address(\n\t\tbytes = base58'3NAG3ZUW9iH53gVNcvCRRSrbBYcJ27jm5ow'\n\t)\n\tproofs = [base58'3dPbXLoVS7JNAQpdnyYo3fL1GHZCDBPGsTXgQU2wCAKPzMPHqPjJbaBhk9GJqF8mpGcbf4FgUgD1U8owEGg5efv2', base58'', base58'', base58'', base58'', base58'', base58'', base58'']\n\tfee = 2\n)"},
		{testLeaseCancelTransaction, "LeaseCancelTransaction(\n\ttimestamp = 2\n\tbodyBytes = base58'9ScSse3sG1iovcPZJeiaLeowaLewU2cduibYfXnTWQ2XYycjaANjfW4EUN9jvU5mBAGzKqNXxDkFzB3XQupWYRtKHVd7qAJaFHKaDsktK3GS3foBU'\n\tversion = 2\n\tid = base58'GWS8byKAP2M6MJB25eQDvdhDL9R98LE5bGWp87aYQE91'\n\tsenderPublicKey = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tleaseId = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tsender = Address(\n\t\tbytes = base58'3NAG3ZUW9iH53gVNcvCRRSrbBYcJ27jm5ow'\n\t)\n\tproofs = [base58'3dPbXLoVS7JNAQpdnyYo3fL1GHZCDBPGsTXgQU2wCAKPzMPHqPjJbaBhk9GJqF8mpGcbf4FgUgD1U8owEGg5efv2', base58'', base58'', base58'', base58'', base58'', base58'', base58'']\n\tfee = 1\n)"},
		{testCreateAliasTransaction, "CreateAliasTransaction(\n\ttimestamp = 2\n\tbodyBytes = base58'QJtUMWVuLBnbLm351tm4G676t3dqva7f3wxAXqzCAkG2bkDG9y4L5skaQ96hJnVfxv3AGcbVJuAZWeEu'\n\tid = base58'6iAUk6JGfM3FH8wMdcAa6gNEZUASnAo1XAEz9Qt3jcYj'\n\tsenderPublicKey = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tsender = Address(\n\t\tbytes = base58'3NAG3ZUW9iH53gVNcvCRRSrbBYcJ27jm5ow'\n\t)\n\tproofs = [base58'3dPbXLoVS7JNAQpdnyYo3fL1GHZCDBPGsTXgQU2wCAKPzMPHqPjJbaBhk9GJqF8mpGcbf4FgUgD1U8owEGg5efv2', base58'', base58'', base58'', base58'', base58'', base58'', base58'']\n\tfee = 1\n\talias = \"str\"\n\tversion = 2\n)"},
		{testSetScriptTransaction, "SetScriptTransaction(\n\ttimestamp = 2\n\tbodyBytes = base58'QEXE2Ry1o4WheSyuacEGoZZBvDQihbX3k17VFUifTsSA7sFdo9TQPPghGKDyPnLbYHEEy48w5Eozum23DTNDatsw4yg2vVBKyqhRsvzwJh9br6'\n\tversion = 2\n\tid = base58'ApLDfTom8CrkmwmEmfaEwg8erG2bk9Ptn8mRR57Kk5jm'\n\tsenderPublicKey = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tsender = Address(\n\t\tbytes = base58'3NAG3ZUW9iH53gVNcvCRRSrbBYcJ27jm5ow'\n\t)\n\tscript = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tproofs = [base58'3dPbXLoVS7JNAQpdnyYo3fL1GHZCDBPGsTXgQU2wCAKPzMPHqPjJbaBhk9GJqF8mpGcbf4FgUgD1U8owEGg5efv2', base58'', base58'', base58'', base58'', base58'', base58'', base58'']\n\tfee = 1\n)"},
		{testSponsorFeeTransaction, "SponsorFeeTransaction(\n\ttimestamp = 3\n\tbodyBytes = base58'3d2hnFNZFcb3aSVr5tp7iRYSYiM4qNrm68EuhjTvFcmgWemrW4WTCzGUYUfbXrWYdt24vFY1FopHesJEJpyHYRkdgsZByrP4AXEaWsn9BBRuDJdWcVEQ'\n\tassetId = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tversion = 2\n\tid = base58'YnZytE9r8T8HPinCKQovYDGcBD5m1mXaQLRovmWjwSM'\n\tsenderPublicKey = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tsender = Address(\n\t\tbytes = base58'3NAG3ZUW9iH53gVNcvCRRSrbBYcJ27jm5ow'\n\t)\n\tproofs = [base58'3dPbXLoVS7JNAQpdnyYo3fL1GHZCDBPGsTXgQU2wCAKPzMPHqPjJbaBhk9GJqF8mpGcbf4FgUgD1U8owEGg5efv2', base58'', base58'', base58'', base58'', base58'', base58'', base58'']\n\tminSponsoredAssetFee = 1\n\tfee = 2\n)"},
		{testDataTransaction, "DataTransaction(\n\ttimestamp = 2\n\tbodyBytes = base58'7WpvRJaykJitESY9Yj4gvhDyMiUxYmz6emAkxmnuhkvvX8eGAxhvtYAzsEdbG4FbmPuaUoeMUnVWZNaaD4p'\n\tdata = [StringEntry(\n\tkey = \"key\"\n\tvalue = \"value\"\n)]\n\tversion = 2\n\tid = base58'EMXZDEQ7BKbmpxn3hQ432gCPHCpu26mV9ck557XtSTno'\n\tsenderPublicKey = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tsender = Address(\n\t\tbytes = base58'3NAG3ZUW9iH53gVNcvCRRSrbBYcJ27jm5ow'\n\t)\n\tproofs = [base58'3dPbXLoVS7JNAQpdnyYo3fL1GHZCDBPGsTXgQU2wCAKPzMPHqPjJbaBhk9GJqF8mpGcbf4FgUgD1U8owEGg5efv2', base58'', base58'', base58'', base58'', base58'', base58'', base58'']\n\tfee = 1\n)"},
		{invObj, "Invocation(\n\toriginCaller = Address(\n\t\tbytes = base58'3NAG3ZUW9iH53gVNcvCRRSrbBYcJ27jm5ow'\n\t)\n\tpayments = [AttachedPayment(\n\tassetId = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tamount = 4\n)]\n\tcallerPublicKey = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tfeeAssetId = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\toriginCallerPublicKey = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\ttransactionId = base58'66V9DMnGSaKuHWWLA62W2kG7kSqmpCYmbsdsraHQGATT'\n\tcaller = Address(\n\t\tbytes = base58'3NAG3ZUW9iH53gVNcvCRRSrbBYcJ27jm5ow'\n\t)\n\tfee = 1\n)"},
		{testAssetInfo, "Asset(\n\tdescription = \"description\"\n\tissuer = Address(\n\t\tbytes = base58'3MbexUVHr88VyDborjw9Veh77MJ68BMNoKi'\n\t)\n\tscripted = true\n\tissuerPublicKey = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tminSponsoredFee = 5\n\tid = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n\tdecimals = 2\n\treissuable = true\n\tname = \"name\"\n\tquantity = 1\n)"},
		{testBlockInfo, "BlockInfo(\n\tbaseTarget = 3\n\tgenerator = Address(\n\t\tbytes = base58'3MbexUVHr88VyDborjw9Veh77MJ68BMNoKi'\n\t)\n\ttimestamp = 1\n\tvrf = base58'3MbexUVHr88VyDborjw9Veh77MJ68BMNoKi'\n\theight = 2\n\tgenerationSignature = base58'3MbexUVHr88VyDborjw9Veh77MJ68BMNoKi'\n\tgeneratorPublicKey = base58'7oKcRfWMsCPKRH6hpZ3oS2qVmknX9dwQ9bzUFXHwFcQN'\n)"},
		{testIssueAction, "Issue(\n\tisReissuable = true\n\tnonce = 1\n\tdescription = \"description\"\n\tdecimals = 2\n\tcompiledScript = Unit\n\tname = \"name\"\n\tquantity = 3\n)"},
		{testReissueAction, "Reissue(\n\tassetId = base58'3MbexUVHr88VyDborjw9Veh77MJ68BMNoKi'\n\tquantity = 1\n\tisReissuable = true\n)"},
		{testBurnAction, "Burn(\n\tassetId = base58'3MbexUVHr88VyDborjw9Veh77MJ68BMNoKi'\n\tquantity = 1\n)"},
		{testSponsorFee, "SponsorFee(\n\tassetId = base58'3MbexUVHr88VyDborjw9Veh77MJ68BMNoKi'\n\tminSponsoredAssetFee = 1\n)"},
		{testLease, "Lease(\n\trecipient = Address(\n\t\tbytes = base58'3MbexUVHr88VyDborjw9Veh77MJ68BMNoKi'\n\t)\n\tamount = 1\n\tnonce = 2\n)"},
		{testLeaseCancel, "LeaseCancel(\n\tleaseId = base58'3MbexUVHr88VyDborjw9Veh77MJ68BMNoKi'\n)"},
	} {
		assert.Equal(t, test.s, test.v.String())
	}
}
