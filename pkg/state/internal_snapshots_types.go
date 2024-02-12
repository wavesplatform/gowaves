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
	ApplyNewLeaseInfo(snapshot InternalNewLeaseInfoSnapshot) error
	ApplyCancelledLeaseInfo(snapshot InternalCancelledLeaseInfoSnapshot) error
	ApplyScriptResult(snapshot InternalScriptResultSnapshot) error
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

type InternalNewLeaseInfoSnapshot struct {
	LeaseID             crypto.Digest
	OriginHeight        proto.Height
	OriginTransactionID *crypto.Digest
}

func (s InternalNewLeaseInfoSnapshot) ApplyInternal(a internalSnapshotApplier) error {
	return a.ApplyNewLeaseInfo(s)
}

type InternalCancelledLeaseInfoSnapshot struct {
	LeaseID             crypto.Digest
	CancelHeight        proto.Height
	CancelTransactionID *crypto.Digest
}

func (s InternalCancelledLeaseInfoSnapshot) ApplyInternal(a internalSnapshotApplier) error {
	return a.ApplyCancelledLeaseInfo(s)
}

type InternalScriptResultSnapshot struct {
	ScriptResult *proto.ScriptResult
}

func (s InternalScriptResultSnapshot) ApplyInternal(a internalSnapshotApplier) error {
	return a.ApplyScriptResult(s)
}
