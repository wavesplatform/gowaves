package state

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride"
)

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

func (s InternalDAppComplexitySnapshot) IsInternal() bool {
	return true
}

func (s InternalDAppComplexitySnapshot) AppendToProtobuf(_ *g.TransactionStateSnapshot) error {
	return nil
}

func (s InternalDAppComplexitySnapshot) InternalSnapshotMarker() {}

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

func (s InternalDAppUpdateComplexitySnapshot) IsInternal() bool {
	return true
}

func (s InternalDAppUpdateComplexitySnapshot) InternalSnapshotMarker() {}

func (s InternalDAppUpdateComplexitySnapshot) AppendToProtobuf(_ *g.TransactionStateSnapshot) error {
	return nil
}

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

func (s InternalAssetScriptComplexitySnapshot) IsInternal() bool {
	return true
}

func (s InternalAssetScriptComplexitySnapshot) InternalSnapshotMarker() {}

func (s InternalAssetScriptComplexitySnapshot) AppendToProtobuf(_ *g.TransactionStateSnapshot) error {
	return nil
}
