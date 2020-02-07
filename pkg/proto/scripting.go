package proto

import (
	"encoding/binary"
	"unicode/utf16"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated"
)

// ScriptAction common interface of script invocation actions.
type ScriptAction interface {
}

// DataEntryScriptAction is an action to manipulate account data state.
type DataEntryScriptAction struct {
	entry DataEntry
}

// TransferScriptAction is an action to emit transfer of asset.
type TransferScriptAction struct {
	Recipient Address
	Amount    int64
	Asset     OptionalAsset
}

// IssueScriptAction is an action to issue a new asset as a result of script invocation.
type IssueScriptAction struct {
	ID          crypto.Digest
	Name        string
	Description string
	Quantity    uint64
	Decimals    byte
	Reissuable  bool
	Timestamp   uint64
}

// ReissueScriptAction is an action to emit Reissue transaction as a result of script invocation.
type ReissueScriptAction struct {
	AssetID    crypto.Digest
	Reissuable bool
	Quantity   uint64
}

// BurnScriptAction is an action to burn some assets in response to script invocation.
type BurnScriptAction struct {
	AssetID    crypto.Digest
	Reissuable bool
	Quantity   uint64
}

type ScriptResult struct {
	DataEntries []DataEntryScriptAction
	Transfers   []TransferScriptAction
	Issues      []IssueScriptAction
	Reissues    []ReissueScriptAction
	Burns       []BurnScriptAction
}

// NewScriptResult creates correct representation of invocation actions for storage and API.
func NewScriptResult(version int, actions []ScriptAction) (*ScriptResult, error) {
	return nil, nil
}

func (sr *ScriptResultV3) MarshalWithAddresses() ([]byte, error) {
	transfersBytes, err := sr.Transfers.MarshalWithAddresses()
	if err != nil {
		return nil, err
	}
	writesBytes, err := sr.Writes.MarshalBinary()
	if err != nil {
		return nil, err
	}
	res := make([]byte, len(transfersBytes)+len(writesBytes)+8)
	pos := 0
	transfersSize := uint32(len(transfersBytes))
	binary.BigEndian.PutUint32(res[pos:], transfersSize)
	pos += 4
	copy(res[pos:], transfersBytes)
	pos += len(transfersBytes)
	writesSize := uint32(len(writesBytes))
	binary.BigEndian.PutUint32(res[pos:], writesSize)
	pos += 4
	copy(res[pos:], writesBytes)
	return res, nil
}

func (sr *ScriptResultV3) UnmarshalWithAddresses(data []byte) error {
	pos := 4
	if len(data) < pos {
		return errors.New("invalid data size")
	}
	transfersSize := binary.BigEndian.Uint32(data[:pos])
	pos += int(transfersSize)
	if len(data) < pos {
		return errors.New("invalid data size")
	}
	var ts TransferSet
	if err := ts.UnmarshalWithAddresses(data[4:pos]); err != nil {
		return err
	}
	if len(data) < pos {
		return errors.New("invalid data size")
	}
	writesSize := binary.BigEndian.Uint32(data[pos:])
	pos += 4
	if len(data) < pos {
		return errors.New("invalid data size")
	}
	var ws WriteSet
	if err := ws.UnmarshalBinary(data[pos:]); err != nil {
		return err
	}
	pos += int(writesSize)
	if pos != len(data) {
		return errors.New("invalid data size")
	}
	sr.Transfers = ts
	sr.Writes = ws
	return nil
}

func (sr *ScriptResultV3) Valid() error {
	if err := sr.Transfers.Valid(); err != nil {
		return err
	}
	if err := sr.Writes.Valid(); err != nil {
		return err
	}
	return nil
}

func (sr *ScriptResultV3) ToProtobuf() (*g.InvokeScriptResult, error) {
	transfers, err := sr.Transfers.ToProtobuf()
	if err != nil {
		return nil, err
	}
	return &g.InvokeScriptResult{
		Data:      sr.Writes.ToProtobuf(),
		Transfers: transfers,
	}, nil
}

type TransferSet []ScriptResultTransfer

func (ts *TransferSet) binarySize() int {
	totalSize := 0
	for _, tr := range *ts {
		totalSize += tr.binarySize()
	}
	return totalSize
}

func (ts *TransferSet) MarshalWithAddresses() ([]byte, error) {
	res := make([]byte, ts.binarySize())
	pos := 0
	for _, tr := range *ts {
		trBytes, err := tr.MarshalWithAddress()
		if err != nil {
			return nil, err
		}
		if pos+len(trBytes) > len(res) {
			return nil, errors.New("invalid data size")
		}
		copy(res[pos:], trBytes)
		pos += len(trBytes)
	}
	return res, nil
}

func (ts *TransferSet) UnmarshalWithAddresses(data []byte) error {
	pos := 0
	for pos < len(data) {
		var tr ScriptResultTransfer
		if err := tr.UnmarshalWithAddress(data[pos:]); err != nil {
			return err
		}
		pos += tr.binarySize()
		*ts = append(*ts, tr)
	}
	return nil
}

func (ts *TransferSet) Valid() error {
	if len(*ts) > maxInvokeTransfers {
		return errors.Errorf("transfer set of size %d is greater than allowed maximum of %d\n", len(*ts), maxInvokeTransfers)
	}
	for _, tr := range *ts {
		if tr.Amount < 0 {
			return errors.New("transfer amount is < 0")
		}
	}
	return nil
}

func (ts *TransferSet) ToProtobuf() ([]*g.InvokeScriptResult_Payment, error) {
	res := make([]*g.InvokeScriptResult_Payment, len(*ts))
	var err error
	for i, tr := range *ts {
		res[i], err = tr.ToProtobuf()
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

type WriteSet []DataEntry

func (ws *WriteSet) binarySize() int {
	totalSize := 0
	for _, entry := range *ws {
		totalSize += entry.binarySize()
	}
	return totalSize
}

func (ws *WriteSet) MarshalBinary() ([]byte, error) {
	res := make([]byte, ws.binarySize())
	pos := 0
	for _, entry := range *ws {
		entryBytes, err := entry.MarshalBinary()
		if err != nil {
			return nil, err
		}
		if pos+len(entryBytes) > len(res) {
			return nil, errors.New("invalid data size")
		}
		copy(res[pos:], entryBytes)
		pos += len(entryBytes)
	}
	return res, nil
}

func (ws *WriteSet) UnmarshalBinary(data []byte) error {
	pos := 0
	for pos < len(data) {
		entry, err := NewDataEntryFromBytes(data[pos:])
		if err != nil {
			return err
		}
		pos += entry.binarySize()
		*ws = append(*ws, entry)
	}
	return nil
}

func (ws *WriteSet) Valid() error {
	if len(*ws) > maxInvokeWrites {
		return errors.Errorf("write set of size %d is greater than allowed maximum of %d\n", len(*ws), maxInvokeWrites)
	}
	totalSize := 0
	for _, entry := range *ws {
		if len(utf16.Encode([]rune(entry.GetKey()))) > maxInvokeWriteKeySizeInBytes {
			return errors.New("key is too large")
		}
		totalSize += entry.binarySize()
	}
	if totalSize > maxWriteSetSizeInBytes {
		return errors.Errorf("total write set size %d is greater than maximum %d\n", totalSize, maxWriteSetSizeInBytes)
	}
	return nil
}

func (ws *WriteSet) ToProtobuf() []*g.DataTransactionData_DataEntry {
	res := make([]*g.DataTransactionData_DataEntry, len(*ws))
	for i, entry := range *ws {
		res[i] = entry.ToProtobuf()
	}
	return res
}

type ScriptResultTransfer struct {
	Recipient Recipient
	Amount    int64
	Asset     OptionalAsset
}

func (tr *ScriptResultTransfer) binarySize() int {
	return AddressSize + 8 + tr.Asset.binarySize()
}

func (tr *ScriptResultTransfer) MarshalWithAddress() ([]byte, error) {
	if tr.Recipient.Address == nil {
		return nil, errors.New("can't marshal Recipient with no address set")
	}
	recipientBytes := tr.Recipient.Address.Bytes()
	if len(recipientBytes) != AddressSize {
		return nil, errors.New("invalid address size")
	}
	amountBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(amountBytes, uint64(tr.Amount))
	assetBytes, err := tr.Asset.MarshalBinary()
	if err != nil {
		return nil, err
	}
	res := make([]byte, tr.binarySize())
	copy(res, amountBytes)
	copy(res[len(amountBytes):], assetBytes)
	copy(res[len(amountBytes)+len(assetBytes):], recipientBytes)
	return res, nil
}

func (tr *ScriptResultTransfer) UnmarshalWithAddress(data []byte) error {
	if len(data) < 8 {
		return errors.New("invalid data size")
	}
	tr.Amount = int64(binary.BigEndian.Uint64(data[:8]))
	var asset OptionalAsset
	if err := asset.UnmarshalBinary(data[8:]); err != nil {
		return err
	}
	tr.Asset = asset
	pos := 8 + asset.binarySize()
	addr, err := NewAddressFromBytes(data[pos:])
	if err != nil {
		return err
	}
	tr.Recipient = NewRecipientFromAddress(addr)
	return nil
}

func (tr *ScriptResultTransfer) ToProtobuf() (*g.InvokeScriptResult_Payment, error) {
	if tr.Recipient.Address == nil {
		return nil, errors.New("script transfer has alias recipient, protobuf needs address")
	}
	addrBody, err := tr.Recipient.Address.Body()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get address body")
	}
	return &g.InvokeScriptResult_Payment{
		Amount:  &g.Amount{AssetId: tr.Asset.ToID(), Amount: tr.Amount},
		Address: addrBody,
	}, nil
}
