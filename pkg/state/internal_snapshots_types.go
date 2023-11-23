package state

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride"
)

type internalSnapshot interface {
	ApplyInternal(internalSnapshotApplier) error
}

type internalSnapshotApplier interface {
	ApplyDAppComplexity(snapshot InternalDAppComplexitySnapshot) error
	ApplyDAppUpdateComplexity(snapshot InternalDAppUpdateComplexitySnapshot) error
	ApplyAssetScriptComplexity(snapshot InternalAssetScriptComplexitySnapshot) error
	ApplyLeaseStateActiveInfo(snapshot InternalLeaseStateActiveInfoSnapshot) error
	ApplyLeaseStateCancelInfo(snapshot InternalLeaseStateCancelInfoSnapshot) error
}

/*
Below are internal snapshots only.
They are not necessary and used for optimization, initialized in the full node mode only.
*/
type InternalDAppComplexitySnapshot struct {
	ScriptAddress proto.WavesAddress
	Estimation    ride.TreeEstimation
	ScriptIsEmpty bool
}

func (s InternalDAppComplexitySnapshot) ApplyInternal(a internalSnapshotApplier) error {
	return a.ApplyDAppComplexity(s)
}

type InternalDAppUpdateComplexitySnapshot struct {
	ScriptAddress proto.WavesAddress
	Estimation    ride.TreeEstimation
	ScriptIsEmpty bool
}

func (s InternalDAppUpdateComplexitySnapshot) ApplyInternal(a internalSnapshotApplier) error {
	return a.ApplyDAppUpdateComplexity(s)
}

type InternalAssetScriptComplexitySnapshot struct {
	AssetID       crypto.Digest
	Estimation    ride.TreeEstimation
	ScriptIsEmpty bool
}

func (s InternalAssetScriptComplexitySnapshot) ApplyInternal(a internalSnapshotApplier) error {
	return a.ApplyAssetScriptComplexity(s)
}

type InternalLeaseStateActiveInfoSnapshot struct {
	LeaseID             crypto.Digest
	OriginHeight        proto.Height
	OriginTransactionID *crypto.Digest
}

func (s InternalLeaseStateActiveInfoSnapshot) ApplyInternal(a internalSnapshotApplier) error {
	return a.ApplyLeaseStateActiveInfo(s)
}

type InternalLeaseStateCancelInfoSnapshot struct {
	LeaseID             crypto.Digest
	CancelHeight        proto.Height
	CancelTransactionID *crypto.Digest
}

func (s InternalLeaseStateCancelInfoSnapshot) ApplyInternal(a internalSnapshotApplier) error {
	return a.ApplyLeaseStateCancelInfo(s)
}
