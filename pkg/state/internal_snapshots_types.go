package state

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride"
)

type internalSnapshot interface {
	IsGeneratedByTxDiff() bool
	ApplyInternal(internalSnapshotApplier) error
}

type internalSnapshotApplier interface {
	ApplyDAppComplexity(snapshot InternalDAppComplexitySnapshot) error
	ApplyDAppUpdateComplexity(snapshot InternalDAppUpdateComplexitySnapshot) error
	ApplyAssetScriptComplexity(snapshot InternalAssetScriptComplexitySnapshot) error
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

func (s InternalDAppComplexitySnapshot) IsGeneratedByTxDiff() bool {
	return false
}

func (s InternalDAppComplexitySnapshot) Apply(a proto.SnapshotApplier) error {
	return a.ApplyInternalSnapshot(&s)
}

func (s InternalDAppComplexitySnapshot) ApplyInternal(a internalSnapshotApplier) error {
	return a.ApplyDAppComplexity(s)
}

func (s InternalDAppComplexitySnapshot) IsInternal() bool {
	return true
}

func (s *InternalDAppComplexitySnapshot) InternalSnapshotMarker() {}

type InternalDAppUpdateComplexitySnapshot struct {
	ScriptAddress proto.WavesAddress
	Estimation    ride.TreeEstimation
	ScriptIsEmpty bool
}

func (s InternalDAppUpdateComplexitySnapshot) IsGeneratedByTxDiff() bool {
	return false
}

func (s InternalDAppUpdateComplexitySnapshot) Apply(a proto.SnapshotApplier) error {
	return a.ApplyInternalSnapshot(&s)
}

func (s InternalDAppUpdateComplexitySnapshot) ApplyInternal(a internalSnapshotApplier) error {
	return a.ApplyDAppUpdateComplexity(s)
}

func (s InternalDAppUpdateComplexitySnapshot) IsInternal() bool {
	return true
}

func (s *InternalDAppUpdateComplexitySnapshot) InternalSnapshotMarker() {}

type InternalAssetScriptComplexitySnapshot struct {
	AssetID       crypto.Digest
	Estimation    ride.TreeEstimation
	ScriptIsEmpty bool
}

func (s InternalAssetScriptComplexitySnapshot) IsGeneratedByTxDiff() bool {
	return false
}

func (s InternalAssetScriptComplexitySnapshot) Apply(a proto.SnapshotApplier) error {
	return a.ApplyInternalSnapshot(&s)
}

func (s InternalAssetScriptComplexitySnapshot) ApplyInternal(a internalSnapshotApplier) error {
	return a.ApplyAssetScriptComplexity(s)
}

func (s InternalAssetScriptComplexitySnapshot) IsInternal() bool {
	return true
}

func (s *InternalAssetScriptComplexitySnapshot) InternalSnapshotMarker() {}
