package state

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/snappy"
	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
	"github.com/valyala/bytebufferpool"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/errs"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/ride/serialization"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/types"
	"go.uber.org/zap"
)

type InvocationStateHandleMode byte

const (
	InvocationStateNoOpMode InvocationStateHandleMode = iota
	InvocationStateWriteMode
	InvocationStateReadMode
)

func (i InvocationStateHandleMode) String() string {
	var s string
	switch i {
	case InvocationStateNoOpMode:
		s = ""
	case InvocationStateWriteMode:
		s = "write"
	case InvocationStateReadMode:
		s = "read"
	default:
		s = fmt.Sprintf("unknown InvocationStateHandleMode value (%d)", i)
	}
	return s
}

func (i *InvocationStateHandleMode) UnmarshalText(text []byte) error {
	switch string(text) {
	case "":
		*i = InvocationStateNoOpMode
	case "write":
		*i = InvocationStateWriteMode
	case "read":
		*i = InvocationStateReadMode
	default:
		return errors.Errorf("failed to UnmarshalText InvocationStateHandleMode: invalid value %q", text)
	}
	return nil
}

func (i InvocationStateHandleMode) MarshalText() (text []byte, err error) {
	var s string
	switch i {
	case InvocationStateNoOpMode, InvocationStateWriteMode, InvocationStateReadMode:
		s = i.String()
	default:
		return nil, errors.Errorf("failed to MarshalText InvocationStateHandleMode: invalid value (%d)", i)
	}
	return []byte(s), nil
}

type invokeApplier struct {
	state types.SmartState
	sc    *scriptCaller

	txHandler *transactionHandler

	stor     *blockchainEntitiesStorage
	settings *settings.BlockchainSettings

	blockDiffer    *blockDiffer
	invokeDiffStor *diffStorageWrapped
	diffApplier    *diffApplier

	buildApiData bool

	scriptInvocationStateWriteChan chan<- ride.AnyScriptInvocationStateWithHeight
	scriptInvocationStateReadChan  <-chan ride.AnyScriptInvocationState
}

const (
	invocationStatesChanSize = 2048
	invocationStatesFileName = "script_invocation_states"
)

func newInvokeApplier(
	ctx context.Context,
	dataDir string,
	state types.SmartState,
	sc *scriptCaller,
	txHandler *transactionHandler,
	stor *blockchainEntitiesStorage,
	settings *settings.BlockchainSettings,
	blockDiffer *blockDiffer,
	diffStor *diffStorageWrapped,
	diffApplier *diffApplier,
	buildApiData bool,
	mode InvocationStateHandleMode,
	stateHeight proto.Height,
) (*invokeApplier, error) {
	ia := &invokeApplier{
		state:          state,
		sc:             sc,
		txHandler:      txHandler,
		stor:           stor,
		settings:       settings,
		blockDiffer:    blockDiffer,
		invokeDiffStor: diffStor,
		diffApplier:    diffApplier,
		buildApiData:   buildApiData,
	}
	switch mode {
	case InvocationStateNoOpMode:
	// no-op
	case InvocationStateWriteMode:
		writeCh, err := runInvocationStatesWriter(ctx, filepath.Join(dataDir, invocationStatesFileName), invocationStatesChanSize)
		if err != nil {
			return nil, errors.Wrap(err, "failed to run invocations states writer")
		}
		ia.scriptInvocationStateWriteChan = writeCh
	case InvocationStateReadMode:
		readCh, err := runInvocationStatesReader(ctx, filepath.Join(dataDir, invocationStatesFileName), invocationStatesChanSize, stateHeight)
		if err != nil {
			return nil, errors.Wrap(err, "failed to run invocations states reader")
		}
		ia.scriptInvocationStateReadChan = readCh
	default:
		return nil, errors.Errorf("invalid InvocationStateHandleMode (%d)", mode)
	}
	return ia, nil
}

func runInvocationStatesWriter(ctx context.Context, invocationStatesFilePath string, chanSize int) (chan<- ride.AnyScriptInvocationStateWithHeight, error) {
	handle := func(ctx context.Context, ch <-chan ride.AnyScriptInvocationStateWithHeight, file *os.File) {
		w := snappy.NewBufferedWriter(file)
		defer func() {
			if err := w.Close(); err != nil {
				zap.S().Errorf("failed to close snappy writer to %q file: %v", invocationStatesFilePath, err)
			}
			if err := file.Close(); err != nil {
				zap.S().Errorf("failed to close %q file: %v", invocationStatesFilePath, err)
			}
		}()
		var (
			currentHeight      proto.Height
			statesOnSameHeight ride.AnyScriptInvocationStates
		)
		defer func() {
			if len(statesOnSameHeight) != 0 { // write last block statements
				// write to the file
				if err := writeStatesOnSameHeight(w, statesOnSameHeight, currentHeight); err != nil {
					// TODO: Is fatal necessary here?
					zap.S().Fatalf("failed to write state results on height %d: %v", currentHeight, err)
				}
			}
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case is, ok := <-ch:
				if !ok {
					return
				}
				if currentHeight != is.Height {
					if len(statesOnSameHeight) != 0 {
						// write to the file
						if err := writeStatesOnSameHeight(w, statesOnSameHeight, currentHeight); err != nil {
							// TODO: Is fatal necessary here?
							zap.S().Fatalf("failed to write state results on height %d: %v", currentHeight, err)
						}
						// reuse the same buffer
						statesOnSameHeight = statesOnSameHeight[:0]
					}
					// update current height
					currentHeight = is.Height
				}
				statesOnSameHeight = append(statesOnSameHeight, is.State)
			}
		}
	}
	f, err := os.OpenFile(invocationStatesFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open %q file", invocationStatesFilePath)
	}
	ch := make(chan ride.AnyScriptInvocationStateWithHeight, chanSize)
	go handle(ctx, ch, f)
	return ch, nil
}

func runInvocationStatesReader(ctx context.Context, invocationStatesFilePath string, chanSize int, initialHeight proto.Height) (<-chan ride.AnyScriptInvocationState, error) {
	handle := func(ctx context.Context, ch chan<- ride.AnyScriptInvocationState, iter *invocationStatesIterator) {
		defer func() {
			if err := iter.Close(); err != nil {
				zap.S().Errorf("failed to close iterator of file %q: %v", invocationStatesFilePath, err)
			}
			close(ch)
		}()
		select {
		case <-ctx.Done():
			return
		default:
			// continue execution
		}
		var (
			next bool
			err  error
		)
		for next, err = iter.Next(); next && err == nil; next, err = iter.Next() {
			select {
			case <-ctx.Done():
				return
			case ch <- iter.Get():
			}
		}
		if err != nil {
			// TODO: Is fatal necessary here?
			zap.S().Fatalf("failed to iterate on invocation states file %q, corrupted data: %v", invocationStatesFilePath, err)
		}
		// EOF
	}
	f, err := os.Open(invocationStatesFilePath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open %q file", invocationStatesFilePath)
	}
	iter := newInvocationStatesIterator(f)
	if err := iter.SkipUntil(initialHeight); err != nil {
		return nil, errors.Wrapf(err, "failed to skip invocations states until %d height", initialHeight)
	}
	ch := make(chan ride.AnyScriptInvocationState, chanSize)
	go handle(ctx, ch, iter)
	return ch, nil
}

type invocationStatesIterator struct {
	file              *os.File
	reader            io.Reader
	lastDecodedHeader invocationStatesHeader
	lastDecodedStates ride.AnyScriptInvocationStates
	currState         ride.AnyScriptInvocationState
	pos               int
}

func newInvocationStatesIterator(f *os.File) *invocationStatesIterator {
	return &invocationStatesIterator{file: f, reader: snappy.NewReader(f)}
}

func (i *invocationStatesIterator) SkipUntil(height proto.Height) error {
	for {
		// read header
		if err := i.lastDecodedHeader.UnmarshalFrom(i.reader); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		// break cycle if we reached state height
		if proto.Height(i.lastDecodedHeader.height) > height {
			break
		}
		// go ahead if we haven't reached necessary height yet
		// discard unnecessary states
		if _, err := io.CopyN(io.Discard, i.reader, int64(i.lastDecodedHeader.marshaledStatesBinSize)); err != nil {
			return err
		}
	}
	return i.lastDecodedStates.UnmarshalFrom(i.reader)
}

func (i *invocationStatesIterator) Next() (bool, error) {
	if i.pos >= len(i.lastDecodedStates) {
		// read and prepare new pack
		if err := i.lastDecodedHeader.UnmarshalFrom(i.reader); err != nil {
			if errors.Is(err, io.EOF) {
				return false, nil
			}
			return false, err
		}
		if err := i.lastDecodedStates.UnmarshalFrom(i.reader); err != nil {
			return false, err
		}
		i.pos = 0
	}
	i.currState = i.lastDecodedStates[i.pos]
	i.pos++
	return true, nil
}

func (i *invocationStatesIterator) Get() ride.AnyScriptInvocationState {
	return i.currState
}

func (i *invocationStatesIterator) Close() error {
	return i.file.Close()
}

func writeStatesOnSameHeight(w io.Writer, states ride.AnyScriptInvocationStates, height proto.Height) error {
	marshaledStates, err := states.MarshalBinary()
	if err != nil {
		return err
	}
	header := invocationStatesHeader{
		height:                 uint32(height),
		marshaledStatesBinSize: uint32(len(marshaledStates)),
	}

	buff := bytebufferpool.Get()
	defer bytebufferpool.Put(buff)
	// fill buffer
	if err := header.MarshalTo(buff); err != nil {
		return err
	}
	if _, err := buff.Write(marshaledStates); err != nil {
		return err
	}
	// write to w
	if _, err := w.Write(buff.Bytes()); err != nil {
		return nil
	}
	return nil
}

const invocationStatesHeaderSize = 4 + 4 // height(uint32) + marshaledStatesBinSize (uint32)

type invocationStatesHeader struct {
	height                 uint32
	marshaledStatesBinSize uint32
}

func (i *invocationStatesHeader) MarshalTo(w io.Writer) error {
	var data [invocationStatesHeaderSize]byte
	binary.BigEndian.PutUint32(data[4:invocationStatesHeaderSize], i.marshaledStatesBinSize)
	binary.BigEndian.PutUint32(data[:4], i.height)
	_, err := w.Write(data[:])
	return err
}

func (i *invocationStatesHeader) UnmarshalFrom(r io.Reader) error {
	var data [invocationStatesHeaderSize]byte
	if _, err := io.ReadFull(r, data[:]); err != nil {
		return err
	}
	i.marshaledStatesBinSize = binary.BigEndian.Uint32(data[4:invocationStatesHeaderSize])
	i.height = binary.BigEndian.Uint32(data[:4])
	return nil
}

type payment struct {
	sender   proto.WavesAddress
	receiver proto.WavesAddress
	amount   uint64
	asset    proto.OptionalAsset
}

func (ia *invokeApplier) newPaymentFromTransferScriptAction(senderAddress proto.WavesAddress, action *proto.TransferScriptAction) (*payment, error) {
	if action.Recipient.Address() == nil {
		return nil, errors.New("transfer has unresolved aliases")
	}
	if action.Amount < 0 {
		return nil, errors.New("negative transfer amount")
	}
	return &payment{
		sender:   senderAddress,
		receiver: *action.Recipient.Address(),
		amount:   uint64(action.Amount),
		asset:    action.Asset,
	}, nil
}

func (ia *invokeApplier) newPaymentFromAttachedPaymentAction(senderAddress proto.WavesAddress, action *proto.AttachedPaymentScriptAction) (*payment, error) {
	if action.Recipient.Address() == nil {
		return nil, errors.New("payment has unresolved aliases")
	}
	if action.Amount < 0 {
		return nil, errors.New("negative payment amount")
	}
	return &payment{
		sender:   senderAddress,
		receiver: *action.Recipient.Address(),
		amount:   uint64(action.Amount),
		asset:    action.Asset,
	}, nil
}

func (ia *invokeApplier) newTxDiffFromPayment(pmt *payment, updateMinIntermediateBalance bool) (txDiff, error) {
	diff := newTxDiff()
	senderKey := byteKey(pmt.sender.ID(), pmt.asset)
	senderBalanceDiff := -int64(pmt.amount)
	if err := diff.appendBalanceDiff(senderKey, newBalanceDiff(senderBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
		return txDiff{}, err
	}
	receiverKey := byteKey(pmt.receiver.ID(), pmt.asset)
	receiverBalanceDiff := int64(pmt.amount)
	if err := diff.appendBalanceDiff(receiverKey, newBalanceDiff(receiverBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
		return txDiff{}, err
	}
	return diff, nil
}

func (ia *invokeApplier) newTxDiffFromScriptTransfer(scriptAddr proto.WavesAddress, action *proto.TransferScriptAction) (txDiff, error) {
	pmt, err := ia.newPaymentFromTransferScriptAction(scriptAddr, action)
	if err != nil {
		return txDiff{}, err
	}
	// updateMinIntermediateBalance is set to false here, because in Scala implementation
	// only fee and payments are checked for temporary negative balance.
	return ia.newTxDiffFromPayment(pmt, false)
}

func (ia *invokeApplier) newTxDiffFromAttachedPaymentAction(scriptAddr proto.WavesAddress, action *proto.AttachedPaymentScriptAction) (txDiff, error) {
	pmt, err := ia.newPaymentFromAttachedPaymentAction(scriptAddr, action)
	if err != nil {
		return txDiff{}, err
	}
	// updateMinIntermediateBalance is set to false here, because in Scala implementation
	// only fee and payments are checked for temporary negative balance.
	return ia.newTxDiffFromPayment(pmt, false)
}

func (ia *invokeApplier) newTxDiffFromScriptIssue(senderAddress proto.AddressID, action *proto.IssueScriptAction) (txDiff, error) {
	diff := newTxDiff()
	senderAssetKey := assetBalanceKey{address: senderAddress, asset: proto.AssetIDFromDigest(action.ID)}
	senderAssetBalanceDiff := action.Quantity
	if err := diff.appendBalanceDiff(senderAssetKey.bytes(), newBalanceDiff(senderAssetBalanceDiff, 0, 0, false)); err != nil {
		return nil, err
	}
	return diff, nil
}

func (ia *invokeApplier) newTxDiffFromScriptReissue(senderAddress proto.AddressID, action *proto.ReissueScriptAction) (txDiff, error) {
	diff := newTxDiff()
	senderAssetKey := assetBalanceKey{address: senderAddress, asset: proto.AssetIDFromDigest(action.AssetID)}
	senderAssetBalanceDiff := action.Quantity
	if err := diff.appendBalanceDiff(senderAssetKey.bytes(), newBalanceDiff(senderAssetBalanceDiff, 0, 0, false)); err != nil {
		return nil, err
	}
	return diff, nil
}

func (ia *invokeApplier) newTxDiffFromScriptBurn(senderAddress proto.AddressID, action *proto.BurnScriptAction) (txDiff, error) {
	diff := newTxDiff()
	senderAssetKey := assetBalanceKey{address: senderAddress, asset: proto.AssetIDFromDigest(action.AssetID)}
	senderAssetBalanceDiff := -action.Quantity
	if err := diff.appendBalanceDiff(senderAssetKey.bytes(), newBalanceDiff(senderAssetBalanceDiff, 0, 0, false)); err != nil {
		return nil, err
	}
	return diff, nil
}

func (ia *invokeApplier) newTxDiffFromScriptLease(senderAddress, recipientAddress proto.AddressID, action *proto.LeaseScriptAction) (txDiff, error) {
	diff := newTxDiff()
	senderKey := wavesBalanceKey{address: senderAddress}
	receiverKey := wavesBalanceKey{address: recipientAddress}
	if err := diff.appendBalanceDiff(senderKey.bytes(), newBalanceDiff(0, 0, action.Amount, false)); err != nil {
		return nil, err
	}
	if err := diff.appendBalanceDiff(receiverKey.bytes(), newBalanceDiff(0, action.Amount, 0, false)); err != nil {
		return nil, err
	}
	return diff, nil
}

func (ia *invokeApplier) newTxDiffFromScriptLeaseCancel(senderAddress proto.AddressID, leaseInfo *leasing) (txDiff, error) {
	diff := newTxDiff()
	senderKey := wavesBalanceKey{address: senderAddress}
	senderLeaseOutDiff := -int64(leaseInfo.Amount)
	if err := diff.appendBalanceDiff(senderKey.bytes(), newBalanceDiff(0, 0, senderLeaseOutDiff, false)); err != nil {
		return nil, err
	}
	receiverKey := wavesBalanceKey{address: leaseInfo.Recipient.ID()}
	receiverLeaseInDiff := -int64(leaseInfo.Amount)
	if err := diff.appendBalanceDiff(receiverKey.bytes(), newBalanceDiff(0, receiverLeaseInDiff, 0, false)); err != nil {
		return nil, err
	}
	return diff, nil
}

func (ia *invokeApplier) saveIntermediateDiff(diff txDiff) error {
	return ia.invokeDiffStor.saveTxDiff(diff)
}

func (ia *invokeApplier) resolveAliases(actions []proto.ScriptAction) error {
	for i, a := range actions {
		switch ta := a.(type) {
		case *proto.TransferScriptAction:
			addr, err := recipientToAddress(ta.Recipient, ia.stor.aliases)
			if err != nil {
				return err
			}
			ta.Recipient = proto.NewRecipientFromAddress(*addr)
			actions[i] = ta
		case *proto.LeaseScriptAction:
			addr, err := recipientToAddress(ta.Recipient, ia.stor.aliases)
			if err != nil {
				return err
			}
			ta.Recipient = proto.NewRecipientFromAddress(*addr)
			actions[i] = ta
		}
	}
	return nil
}

func (ia *invokeApplier) countIssuedAssets(actions []proto.ScriptAction) (uint64, error) {
	issuedAssetsCount := uint64(0)
	for _, action := range actions {
		switch a := action.(type) {
		case *proto.IssueScriptAction:
			assetParams := assetParams{a.Quantity, a.Decimals, a.Reissuable}
			nft, err := isNFT(ia.stor.features, assetParams)
			if err != nil {
				return 0, err
			}
			if !nft {
				issuedAssetsCount += 1
			}
		}
	}
	return issuedAssetsCount, nil
}

func (ia *invokeApplier) countEmptyDataEntryKeys(actions []proto.ScriptAction) uint64 {
	var out uint64 = 0
	for _, action := range actions {
		switch a := action.(type) {
		case *proto.DataEntryScriptAction:
			if len(a.Entry.GetKey()) == 0 {
				out = +1
			}
		}
	}
	return out
}

func (ia *invokeApplier) countActionScriptRuns(actions []proto.ScriptAction) (uint64, error) {
	scriptRuns := uint64(0)
	for _, action := range actions {
		var assetID proto.AssetID
		switch a := action.(type) {
		case *proto.TransferScriptAction:
			if !a.Asset.Present {
				continue // This is waves asset and it can't be scripted
			}
			assetID = proto.AssetIDFromDigest(a.Asset.ID)
		case *proto.ReissueScriptAction:
			assetID = proto.AssetIDFromDigest(a.AssetID)
		case *proto.BurnScriptAction:
			assetID = proto.AssetIDFromDigest(a.AssetID)
		default:
			continue
		}
		isSmartAsset, err := ia.stor.scriptsStorage.newestIsSmartAsset(assetID)
		if err != nil {
			return 0, errors.Errorf("failed to count actions: failed to check whether the asset was smart")
		}
		if isSmartAsset {
			scriptRuns++
		}
	}
	return scriptRuns, nil
}

func errorForSmartAsset(msg string, asset crypto.Digest) error {
	var text string
	if msg != "" {
		text = fmt.Sprintf("Transaction is not allowed by token-script id %s: throw from asset script.", asset.String())
	} else {
		// scala compatible error message
		text = fmt.Sprintf("Transaction is not allowed by token-script id %s. Transaction is not allowed by script of the asset", asset.String())
	}
	return errors.New(text)
}

type addlInvokeInfo struct {
	*fallibleValidationParams

	scriptAddr           *proto.WavesAddress
	scriptPK             crypto.PublicKey
	scriptRuns           uint64
	failedChanges        txBalanceChanges
	actions              []proto.ScriptAction
	paymentSmartAssets   []crypto.Digest
	disableSelfTransfers bool
	libVersion           ast.LibraryVersion
}

func (ia *invokeApplier) senderCredentialsFromScriptAction(a proto.ScriptAction, info *addlInvokeInfo) (crypto.PublicKey, proto.WavesAddress, error) {
	senderPK := info.scriptPK
	senderAddress := *info.scriptAddr
	if a.SenderPK() != nil {
		var err error
		senderPK = *a.SenderPK()
		senderAddress, err = proto.NewAddressFromPublicKey(ia.settings.AddressSchemeCharacter, senderPK)
		if err != nil {
			return crypto.PublicKey{}, proto.WavesAddress{}, err
		}
	}
	return senderPK, senderAddress, nil
}

func (ia *invokeApplier) fallibleValidation(tx proto.Transaction, info *addlInvokeInfo) (proto.TxFailureReason, txBalanceChanges, error) {
	// Check smart asset scripts on payments.
	for _, smartAsset := range info.paymentSmartAssets {
		r, err := ia.sc.callAssetScript(tx, smartAsset, info.fallibleValidationParams.appendTxParams)
		if err != nil {
			return proto.SmartAssetOnPaymentFailure, info.failedChanges, errorForSmartAsset(err.Error(), smartAsset)
		}
		if !r.Result() {
			return proto.SmartAssetOnPaymentFailure, info.failedChanges, errorForSmartAsset("", smartAsset)
		}
	}
	// Resolve all aliases.
	// It has to be done before validation because we validate addresses, not aliases.
	if err := ia.resolveAliases(info.actions); err != nil {
		return proto.DAppError, info.failedChanges, errors.New("ScriptResult; failed to resolve aliases")
	}
	// Validate produced actions.
	isUTF16KeyLen := !info.blockV5Activated // if RideV4 isn't activated
	maxDataEntriesSize := proto.MaxDataEntriesScriptActionsSizeInBytesV1
	if info.blockV5Activated {
		maxDataEntriesSize = proto.MaxDataEntriesScriptActionsSizeInBytesV2
	}
	restrictions := proto.ActionsValidationRestrictions{
		DisableSelfTransfers:  info.disableSelfTransfers,
		IsUTF16KeyLen:         isUTF16KeyLen,
		IsProtobufTransaction: proto.IsProtobufTx(tx),
		MaxDataEntriesSize:    maxDataEntriesSize,
		Scheme:                ia.settings.AddressSchemeCharacter,
		ScriptAddress:         *info.scriptAddr,
	}
	validatePayments := info.checkerInfo.height > ia.settings.InternalInvokePaymentsValidationAfterHeight
	if err := proto.ValidateActions(info.actions, restrictions, info.rideV6Activated, info.libVersion, validatePayments); err != nil {
		return proto.DAppError, info.failedChanges, err
	}
	// Check full transaction fee (with actions and payments scripts).
	issuedAssetsCount, err := ia.countIssuedAssets(info.actions)
	if err != nil {
		return proto.DAppError, info.failedChanges, err
	}
	if err := ia.checkFullFee(tx, info.scriptRuns, issuedAssetsCount); err != nil {
		return proto.InsufficientActionsFee, info.failedChanges, err
	}

	txIDBytes, err := tx.GetID(ia.settings.AddressSchemeCharacter)
	if err != nil {
		return proto.DAppError, info.failedChanges, err
	}
	txID, err := crypto.NewDigestFromBytes(txIDBytes)
	if err != nil {
		return proto.DAppError, info.failedChanges, err
	}
	// Add feeAndPaymentChanges to stor before performing actions.
	feeAndPaymentChanges, err := ia.blockDiffer.createTransactionDiff(tx, info.block, newDifferInfo(info.blockInfo))
	if err != nil {
		return proto.DAppError, info.failedChanges, err
	}
	totalChanges := feeAndPaymentChanges
	if err := ia.saveIntermediateDiff(totalChanges.diff); err != nil {
		return proto.DAppError, info.failedChanges, err
	}
	// Empty keys rejected since protobuf version.
	if proto.IsProtobufTx(tx) && ia.countEmptyDataEntryKeys(info.actions) > 0 {
		return proto.DAppError, info.failedChanges, errs.NewTxValidationError(fmt.Sprintf("Empty keys aren't allowed in tx version >= %d", tx.GetVersion()))
	}

	// Perform actions.
	for _, action := range info.actions {
		senderPK, senderAddress, err := ia.senderCredentialsFromScriptAction(action, info)
		if err != nil {
			return proto.DAppError, info.failedChanges, err
		}
		totalChanges.appendAddr(senderAddress)
		switch a := action.(type) {
		case *proto.DataEntryScriptAction:
			ia.stor.accountsDataStor.appendEntryUncertain(senderAddress, a.Entry)

		case *proto.TransferScriptAction:
			// Perform transfers.
			recipientAddress := a.Recipient.Address()
			totalChanges.appendAddr(*recipientAddress)
			assetExists := ia.stor.assets.newestAssetExists(a.Asset)
			if !assetExists {
				return proto.DAppError, info.failedChanges, errors.New("invalid asset in transfer")
			}
			var isSmartAsset bool
			if a.Asset.Present {
				isSmartAsset, err = ia.stor.scriptsStorage.newestIsSmartAsset(proto.AssetIDFromDigest(a.Asset.ID))
				if err != nil {
					return proto.DAppError, info.failedChanges, errors.Errorf("transfer script actions: failed to check whether the asset was smart")
				}
			}
			if isSmartAsset {
				fullTr, err := proto.NewFullScriptTransfer(a, senderAddress, info.scriptPK, &txID, tx.GetTimestamp())
				if err != nil {
					return proto.DAppError, info.failedChanges, errors.Wrap(err, "failed to convert transfer to full script transfer")
				}
				// Call asset script if transferring smart asset.
				res, err := ia.sc.callAssetScriptWithScriptTransfer(fullTr, a.Asset.ID, info.appendTxParams)
				if err != nil {
					return proto.SmartAssetOnActionFailure, info.failedChanges, errorForSmartAsset(err.Error(), a.Asset.ID)
				}
				if !res.Result() {
					return proto.SmartAssetOnActionFailure, info.failedChanges, errorForSmartAsset("", a.Asset.ID)
				}
			}
			txDiff, err := ia.newTxDiffFromScriptTransfer(senderAddress, a)
			if err != nil {
				return proto.DAppError, info.failedChanges, err
			}
			// diff must be saved to storage, because further asset scripts must take
			// recent balance changes into account.
			if err := ia.saveIntermediateDiff(txDiff); err != nil {
				return proto.DAppError, info.failedChanges, err
			}
			// Append intermediate diff to common diff.
			for key, balanceDiff := range txDiff {
				if err := totalChanges.diff.appendBalanceDiffStr(key, balanceDiff); err != nil {
					return proto.DAppError, info.failedChanges, err
				}
			}
		case *proto.AttachedPaymentScriptAction:
			// Perform transfers.
			recipientAddress := a.Recipient.Address()
			totalChanges.appendAddr(*recipientAddress)
			assetExists := ia.stor.assets.newestAssetExists(a.Asset)
			if !assetExists {
				return proto.DAppError, info.failedChanges, errors.New("invalid asset in transfer")
			}
			var isSmartAsset bool
			if a.Asset.Present {
				isSmartAsset, err = ia.stor.scriptsStorage.newestIsSmartAsset(proto.AssetIDFromDigest(a.Asset.ID))
				if err != nil {
					return proto.DAppError, info.failedChanges, errors.Errorf("attached payment script actions: failed to check whether the asset was smart")
				}
			}
			if isSmartAsset {
				fullTr, err := proto.NewFullScriptTransferFromPaymentAction(a, senderAddress, info.scriptPK, &txID, tx.GetTimestamp())
				if err != nil {
					return proto.DAppError, info.failedChanges, errors.Wrap(err, "failed to convert transfer to full script transfer")
				}
				// Call asset script if transferring smart asset.
				res, err := ia.sc.callAssetScriptWithScriptTransfer(fullTr, a.Asset.ID, info.appendTxParams)
				if err != nil {
					return proto.SmartAssetOnActionFailure, info.failedChanges, errorForSmartAsset(err.Error(), a.Asset.ID)
				}
				if !res.Result() {
					return proto.SmartAssetOnActionFailure, info.failedChanges, errorForSmartAsset("", a.Asset.ID)
				}
			}
			txDiff, err := ia.newTxDiffFromAttachedPaymentAction(senderAddress, a)
			if err != nil {
				if !validatePayments && txID == id1 {
					txDiff = diff1
				} else {
					return proto.DAppError, info.failedChanges, err
				}
			}
			// diff must be saved to storage, because further asset scripts must take
			// recent balance changes into account.
			if err := ia.saveIntermediateDiff(txDiff); err != nil {
				return proto.DAppError, info.failedChanges, err
			}
			// Append intermediate diff to common diff.
			for key, balanceDiff := range txDiff {
				if err := totalChanges.diff.appendBalanceDiffStr(key, balanceDiff); err != nil {
					return proto.DAppError, info.failedChanges, err
				}
			}

		case *proto.IssueScriptAction:
			// Create asset's info.
			assetInfo := &assetInfo{
				assetConstInfo: assetConstInfo{
					tail:     proto.DigestTail(a.ID),
					issuer:   senderPK,
					decimals: int8(a.Decimals),
				},
				assetChangeableInfo: assetChangeableInfo{
					quantity:    *big.NewInt(a.Quantity),
					name:        a.Name,
					description: a.Description,
					reissuable:  a.Reissuable,
				},
			}
			id := proto.AssetIDFromDigest(a.ID)
			ia.stor.assets.issueAssetUncertain(id, assetInfo)
			// Currently asset script is always empty.
			// TODO: if this script is ever set, don't forget to
			// also save complexity for it here using saveComplexityForAsset().
			if err := ia.stor.scriptsStorage.setAssetScriptUncertain(a.ID, proto.Script{}, senderPK); err != nil {
				return proto.DAppError, info.failedChanges, err
			}
			txDiff, err := ia.newTxDiffFromScriptIssue(senderAddress.ID(), a)
			if err != nil {
				return proto.DAppError, info.failedChanges, err
			}
			// diff must be saved to storage, because further asset scripts must take
			// recent balance changes into account.
			if err := ia.saveIntermediateDiff(txDiff); err != nil {
				return proto.DAppError, info.failedChanges, err
			}
			// Append intermediate diff to common diff.
			for key, balanceDiff := range txDiff {
				if err := totalChanges.diff.appendBalanceDiffStr(key, balanceDiff); err != nil {
					return proto.DAppError, info.failedChanges, err
				}
			}

		case *proto.ReissueScriptAction:
			// Check validity of reissue.
			id := proto.AssetIDFromDigest(a.AssetID)
			assetInfo, err := ia.stor.assets.newestAssetInfo(id)
			if err != nil {
				return proto.DAppError, info.failedChanges, err
			}
			if assetInfo.issuer != senderPK {
				return proto.DAppError, info.failedChanges, errs.NewAssetIssuedByOtherAddress("asset was issued by other address")
			}
			if !assetInfo.reissuable {
				return proto.DAppError, info.failedChanges, errors.New("attempt to reissue asset which is not reissuable")
			}
			if math.MaxInt64-a.Quantity < assetInfo.quantity.Int64() && info.block.Timestamp >= ia.settings.ReissueBugWindowTimeEnd {
				return proto.DAppError, info.failedChanges, errors.New("asset total value overflow")
			}
			ok, err := ia.validateActionSmartAsset(a.AssetID, a, senderPK, txID, tx.GetTimestamp(), info.appendTxParams)
			if err != nil {
				return proto.SmartAssetOnActionFailure, info.failedChanges, errorForSmartAsset(err.Error(), a.AssetID)
			}
			if !ok {
				return proto.SmartAssetOnActionFailure, info.failedChanges, errorForSmartAsset("", a.AssetID)
			}
			// Update asset's info.
			change := &assetReissueChange{
				reissuable: a.Reissuable,
				diff:       a.Quantity,
			}
			if err := ia.stor.assets.reissueAssetUncertain(id, change); err != nil {
				return proto.DAppError, info.failedChanges, err
			}
			txDiff, err := ia.newTxDiffFromScriptReissue(senderAddress.ID(), a)
			if err != nil {
				return proto.DAppError, info.failedChanges, err
			}
			// diff must be saved to storage, because further asset scripts must take
			// recent balance changes into account.
			if err := ia.saveIntermediateDiff(txDiff); err != nil {
				return proto.DAppError, info.failedChanges, err
			}
			// Append intermediate diff to common diff.
			for key, balanceDiff := range txDiff {
				if err := totalChanges.diff.appendBalanceDiffStr(key, balanceDiff); err != nil {
					return proto.DAppError, info.failedChanges, err
				}
			}

		case *proto.BurnScriptAction:
			// Check burn.
			id := proto.AssetIDFromDigest(a.AssetID)
			assetInfo, err := ia.stor.assets.newestAssetInfo(id)
			if err != nil {
				return proto.DAppError, info.failedChanges, err
			}
			burnAnyTokensEnabled, err := ia.stor.features.newestIsActivated(int16(settings.BurnAnyTokens))
			if err != nil {
				return proto.DAppError, info.failedChanges, err
			}
			if !burnAnyTokensEnabled && assetInfo.issuer != senderPK {
				return proto.DAppError, info.failedChanges, errors.New("asset was issued by other address")
			}
			quantityDiff := big.NewInt(a.Quantity)
			if assetInfo.quantity.Cmp(quantityDiff) == -1 {
				return proto.DAppError, info.failedChanges, errs.NewAccountBalanceError("trying to burn more assets than exist at all")
			}
			ok, err := ia.validateActionSmartAsset(a.AssetID, a, senderPK, txID, tx.GetTimestamp(), info.appendTxParams)
			if err != nil {
				return proto.SmartAssetOnActionFailure, info.failedChanges, errorForSmartAsset(err.Error(), a.AssetID)
			}
			if !ok {
				return proto.SmartAssetOnActionFailure, info.failedChanges, errorForSmartAsset("", a.AssetID)
			}
			// Update asset's info
			// Modify asset.
			change := &assetBurnChange{
				diff: a.Quantity,
			}
			if err := ia.stor.assets.burnAssetUncertain(id, change); err != nil {
				return proto.DAppError, info.failedChanges, errors.Wrap(err, "failed to burn asset")
			}
			txDiff, err := ia.newTxDiffFromScriptBurn(senderAddress.ID(), a)
			if err != nil {
				return proto.DAppError, info.failedChanges, err
			}
			// diff must be saved to storage, because further asset scripts must take
			// recent balance changes into account.
			if err := ia.saveIntermediateDiff(txDiff); err != nil {
				return proto.DAppError, info.failedChanges, err
			}
			// Append intermediate diff to common diff.
			for key, balanceDiff := range txDiff {
				if err := totalChanges.diff.appendBalanceDiffStr(key, balanceDiff); err != nil {
					return proto.DAppError, info.failedChanges, err
				}
			}

		case *proto.SponsorshipScriptAction:
			assetID := proto.AssetIDFromDigest(a.AssetID)
			assetInfo, err := ia.stor.assets.newestAssetInfo(assetID)
			if err != nil {
				return proto.DAppError, info.failedChanges, err
			}
			sponsorshipActivated, err := ia.stor.features.newestIsActivated(int16(settings.FeeSponsorship))
			if err != nil {
				return proto.DAppError, info.failedChanges, err
			}
			if !sponsorshipActivated {
				return proto.DAppError, info.failedChanges, errors.New("sponsorship has not been activated yet")
			}
			if assetInfo.issuer != senderPK {
				return proto.DAppError, info.failedChanges, errors.Errorf("asset %s was not issued by this DApp", a.AssetID.String())
			}

			isSmartAsset, err := ia.stor.scriptsStorage.newestIsSmartAsset(assetID)
			if err != nil {
				return proto.DAppError, info.failedChanges, errors.Errorf("sponsorships: failed to check whether the asset was smart")
			}
			if isSmartAsset {
				return proto.DAppError, info.failedChanges, errors.Errorf("can not sponsor smart asset %s", a.AssetID.String())
			}
			ia.stor.sponsoredAssets.sponsorAssetUncertain(a.AssetID, uint64(a.MinFee))

		case *proto.LeaseScriptAction:
			if a.Recipient.Address() == nil {
				return proto.DAppError, info.failedChanges, errors.New("transfer has unresolved aliases")
			}
			recipientAddress := *a.Recipient.Address()
			if senderAddress == recipientAddress {
				return proto.DAppError, info.failedChanges, errors.New("leasing to itself is not allowed")
			}
			if a.Amount <= 0 {
				return proto.DAppError, info.failedChanges, errors.New("non-positive leasing amount")
			}
			totalChanges.appendAddr(recipientAddress)

			// Add new leasing info
			l := &leasing{
				OriginTransactionID: &txID,
				Sender:              senderAddress,
				Recipient:           recipientAddress,
				Amount:              uint64(a.Amount),
				Height:              info.blockInfo.Height,
				Status:              LeaseActive,
				RecipientAlias:      a.Recipient.Alias(),
			}
			ia.stor.leases.addLeasingUncertain(a.ID, l)

			txDiff, err := ia.newTxDiffFromScriptLease(senderAddress.ID(), recipientAddress.ID(), a)
			if err != nil {
				return proto.DAppError, info.failedChanges, err
			}
			if err := ia.saveIntermediateDiff(txDiff); err != nil {
				return proto.DAppError, info.failedChanges, err
			}
			for key, balanceDiff := range txDiff {
				if err := totalChanges.diff.appendBalanceDiffStr(key, balanceDiff); err != nil {
					return proto.DAppError, info.failedChanges, err
				}
			}

		case *proto.LeaseCancelScriptAction:
			li, err := ia.stor.leases.newestLeasingInfo(a.LeaseID)
			if err != nil {
				return proto.DAppError, info.failedChanges, err
			}
			if senderAddress != li.Sender {
				return proto.DAppError, info.failedChanges, errors.Errorf("attempt to cancel leasing that was created by other account; leaser '%s'; canceller '%s'; leasing: %s", li.Sender.String(), senderAddress.String(), a.LeaseID.String()) //TODO: Create a scala compatible error in errs package and use it here
			}
			// Update leasing info
			if err := ia.stor.leases.cancelLeasingUncertain(a.LeaseID, info.blockInfo.Height, &txID); err != nil {
				return proto.DAppError, info.failedChanges, errors.Wrap(err, "failed to cancel leasing")
			}

			totalChanges.appendAddr(li.Sender)
			totalChanges.appendAddr(li.Recipient)
			txDiff, err := ia.newTxDiffFromScriptLeaseCancel(senderAddress.ID(), li)
			if err != nil {
				return proto.DAppError, info.failedChanges, err
			}
			if err := ia.saveIntermediateDiff(txDiff); err != nil {
				return proto.DAppError, info.failedChanges, err
			}
			for key, balanceDiff := range txDiff {
				if err := totalChanges.diff.appendBalanceDiffStr(key, balanceDiff); err != nil {
					return proto.DAppError, info.failedChanges, err
				}
			}

		default:
			return proto.DAppError, info.failedChanges, errors.Errorf("unsupported script action '%T'", a)
		}
	}
	if info.acceptFailed {
		// Validate total balance changes.
		if err := ia.diffApplier.validateTxDiff(totalChanges.diff, ia.invokeDiffStor.diffStorage); err != nil {
			// Total balance changes lead to negative balance, hence invoke has failed.
			// TODO: use different code for negative balances after it is introduced; use better error text here (addr + amount + asset).
			return proto.DAppError, info.failedChanges, err
		}
	}
	// If we are here, invoke succeeded.
	ia.blockDiffer.appendBlockInfoToTxDiff(totalChanges.diff, info.block)
	return 0, totalChanges, nil
}

// applyInvokeScript checks InvokeScript transaction, creates its balance diffs and adds changes to `uncertain` storage.
// If the transaction does not fail, changes are committed (moved from uncertain to normal storage)
// later in performInvokeScriptWithProofs().
// If the transaction fails, performInvokeScriptWithProofs() is not called and changes are discarded later using dropUncertain().
func (ia *invokeApplier) applyInvokeScript(tx proto.Transaction, info *fallibleValidationParams) (*applicationResult, error) {
	// In defer we should clean all the temp changes invoke does to state.
	defer func() {
		ia.invokeDiffStor.invokeDiffsStor.reset()
	}()

	var (
		paymentsLength int
		scriptAddr     *proto.WavesAddress
		txID           crypto.Digest
		sender         proto.Address
		scriptPK       crypto.PublicKey
		libVersion     ast.LibraryVersion
	)
	switch transaction := tx.(type) {
	case *proto.InvokeScriptWithProofs:
		var err error
		scriptAddr, err = recipientToAddress(transaction.ScriptRecipient, ia.stor.aliases)
		if err != nil {
			return nil, errors.Wrap(err, "recipientToAddress() failed")
		}
		paymentsLength = len(transaction.Payments)
		txID = *transaction.ID
		sender, err = proto.NewAddressFromPublicKey(ia.settings.AddressSchemeCharacter, transaction.SenderPK)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to apply script invocation")
		}
		si, err := ia.stor.scriptsStorage.newestScriptBasicInfoByAddressID(scriptAddr.ID())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get script's public key on address '%s'", scriptAddr.String())
		}
		scriptPK = si.PK
		libVersion = si.LibraryVersion

	case *proto.InvokeExpressionTransactionWithProofs:
		addr, err := proto.NewAddressFromPublicKey(ia.settings.AddressSchemeCharacter, transaction.SenderPK)
		if err != nil {
			return nil, errors.Wrap(err, "recipientToAddress() failed")
		}
		sender = addr
		scriptAddr = &addr
		// TODO: parse only scriptHeader to get ast.LibraryVersion
		tree, err := serialization.Parse(transaction.Expression)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse decoded invoke expression into tree")
		}
		txID = *transaction.ID
		scriptPK = transaction.SenderPK
		libVersion = tree.LibVersion

	case *proto.EthereumTransaction:
		var err error
		scriptAddr, err = transaction.WavesAddressTo(ia.settings.AddressSchemeCharacter)
		if err != nil {
			return nil, err
		}
		decodedData := transaction.TxKind.DecodedData()
		paymentsLength = len(decodedData.Payments)
		txID = *transaction.ID
		sender, err = transaction.WavesAddressFrom(ia.settings.AddressSchemeCharacter)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to apply script invocation")
		}
		si, err := ia.stor.scriptsStorage.newestScriptBasicInfoByAddressID(scriptAddr.ID())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get script's public key on address '%s'", scriptAddr.String())
		}
		scriptPK = si.PK
		libVersion = si.LibraryVersion

	default:
		return nil, errors.Errorf("failed to apply an invoke script: unexpected type of transaction (%T)", tx)
	}

	// If BlockV5 feature is not activated, we never accept failed transactions.
	info.acceptFailed = info.blockV5Activated && info.acceptFailed
	// Check sender script, if any.
	// TODO: ignore this check in ScriptInvocationState read mode (blockchain file can't have invoke tx with failed sender's script)
	if info.senderScripted {
		if err := ia.sc.callAccountScriptWithTx(tx, info.appendTxParams); err != nil {
			// Never accept invokes with failed script on transaction sender.
			return nil, err
		}
	}
	// Basic checks against state.
	paymentSmartAssets, err := ia.txHandler.checkTx(tx, info.checkerInfo)
	if err != nil {
		return nil, err
	}

	// Check that the script's library supports multiple payments.
	// We don't have to check feature activation because we've done it before.
	if paymentsLength >= 2 && libVersion < ast.LibV4 {
		return nil, errors.Errorf("multiple payments is not allowed for RIDE library version %d", libVersion)
	}
	// Refuse payments to DApp itself since activation of BlockV5 (acceptFailed) and for DApps with StdLib V4.
	disableSelfTransfers := info.acceptFailed && libVersion >= 4
	if disableSelfTransfers && paymentsLength > 0 {
		if sender == *scriptAddr {
			return nil, errors.New("paying to DApp itself is forbidden since RIDE V4")
		}
	}
	// Basic differ for InvokeScript creates only fee and payment diff.
	// Create changes for both failed and successful scenarios.
	failedChanges, err := ia.blockDiffer.createFailedTransactionDiff(tx, info.block, newDifferInfo(info.blockInfo))
	if err != nil {
		return nil, err
	}

	// Call script function.
	r, err := ia.handleInvoke(tx, info, *scriptAddr)
	if err != nil {
		// Script returned error, it's OK, but we have to decide is it failed or rejected transaction.
		// After activation of RideV6 feature transactions are failed if they are not cheap regardless the error kind.
		isCheap := int(ia.sc.recentTxComplexity) <= FailFreeInvokeComplexity
		if info.rideV6Activated {
			if !info.acceptFailed || isCheap {
				return nil, errors.Wrapf(
					err, "transaction rejected with spent complexity %d and following call stack:\n%s",
					ride.EvaluationErrorSpentComplexity(err),
					strings.Join(ride.EvaluationErrorCallStack(err), "\n"),
				)
			}
			res := &invocationResult{failed: true, code: proto.DAppError, text: err.Error(), changes: failedChanges}
			return ia.handleInvocationResult(txID, info, res)
		}
		// Before RideV6 activation in the following cases the transaction is rejected:
		// 1) Failing of transactions is not activated yet, reject everything
		// 2) The error is ride.InternalInvocationError and correct fail/reject behaviour is activated
		// 3) The spent complexity is less than limit
		switch ride.GetEvaluationErrorType(err) {
		case ride.UserError, ride.RuntimeError, ride.ComplexityLimitExceed:
			// Usual script error produced by user code or system functions.
			// We reject transaction if spent complexity is less than limit.
			if !info.acceptFailed || isCheap { // Reject transaction if no failed transactions or the transaction is cheap
				return nil, errors.Wrapf(
					err, "transaction rejected with spent complexity %d and following call stack:\n%s",
					ride.EvaluationErrorSpentComplexity(err),
					strings.Join(ride.EvaluationErrorCallStack(err), "\n"),
				)
			}
			res := &invocationResult{failed: true, code: proto.DAppError, text: err.Error(), changes: failedChanges}
			return ia.handleInvocationResult(txID, info, res)
		case ride.InternalInvocationError:
			// Special script error produced by internal script invocation or application of results.
			// Reject transaction after certain height
			rejectOnInvocationError := info.checkerInfo.height >= ia.settings.InternalInvokeCorrectFailRejectBehaviourAfterHeight
			if !info.acceptFailed || rejectOnInvocationError || isCheap {
				return nil, errors.Wrapf(
					err, "transaction rejected with spent complexity %d and following call stack:\n%s",
					ride.EvaluationErrorSpentComplexity(err),
					strings.Join(ride.EvaluationErrorCallStack(err), "\n"),
				)
			}
			res := &invocationResult{failed: true, code: proto.DAppError, text: err.Error(), changes: failedChanges}
			return ia.handleInvocationResult(txID, info, res)
		case ride.Undefined, ride.EvaluationFailure: // Unhandled or evaluator error
			return nil, errors.Wrapf(err, "invocation of transaction '%s' failed", txID.String())
		default:
			return nil, errors.Wrapf(err, "invocation of transaction '%s' failed", txID.String())
		}
	}
	var scriptRuns uint64 = 0
	// After activation of RideV5 (16) feature we don't take extra fee for execution of smart asset scripts.
	if !info.rideV5Activated {
		actionScriptRuns, err := ia.countActionScriptRuns(r.ScriptActions())
		if err != nil {
			return nil, errors.Wrap(err, "failed to countActionScriptRuns")
		}
		scriptRuns += uint64(len(paymentSmartAssets)) + actionScriptRuns
	}
	if info.senderScripted {
		// Since activation of RideV5 (16) feature we don't take fee for verifier execution if it's complexity is less than `FreeVerifierComplexity` limit
		if info.rideV5Activated {
			treeEstimation, err := ia.stor.scriptsComplexity.newestScriptComplexityByAddr(info.senderAddress, info.checkerInfo.estimatorVersion())
			if err != nil {
				return nil, errors.Wrap(err, "invoke failed to get verifier complexity")
			}
			if treeEstimation.Verifier > FreeVerifierComplexity {
				scriptRuns++
			}
		} else {
			scriptRuns++
		}
	}
	var res *invocationResult
	code, changes, err := ia.fallibleValidation(tx, &addlInvokeInfo{
		fallibleValidationParams: info,
		scriptAddr:               scriptAddr,
		scriptPK:                 scriptPK,
		scriptRuns:               scriptRuns,
		failedChanges:            failedChanges,
		actions:                  r.ScriptActions(),
		paymentSmartAssets:       paymentSmartAssets,
		disableSelfTransfers:     disableSelfTransfers,
		libVersion:               libVersion,
	})
	if err != nil {
		zap.S().Debugf("fallibleValidation error in tx %s. Error: %s", txID.String(), err.Error())
		// If fallibleValidation fails, we should save transaction to blockchain when acceptFailed is true.
		if !info.acceptFailed ||
			(ia.sc.recentTxComplexity <= FailFreeInvokeComplexity &&
				info.checkerInfo.height >= ia.settings.InternalInvokeCorrectFailRejectBehaviourAfterHeight) {
			return nil, err
		}
		res = &invocationResult{
			failed:     true,
			code:       code,
			text:       err.Error(),
			scriptRuns: scriptRuns,
			actions:    r.ScriptActions(),
			changes:    changes,
		}
	} else {
		res = &invocationResult{
			failed:     false,
			scriptRuns: scriptRuns,
			actions:    r.ScriptActions(),
			changes:    changes,
		}
	}
	return ia.handleInvocationResult(txID, info, res)
}

func (ia *invokeApplier) handleInvoke(tx proto.Transaction, info *fallibleValidationParams, scriptAddr proto.WavesAddress) (ride.Result, error) {
	if ia.scriptInvocationStateReadChan != nil {
		is, ok := <-ia.scriptInvocationStateReadChan
		if !ok {
			id, err := tx.GetID(ia.settings.AddressSchemeCharacter)
			if err != nil {
				return nil, err
			}
			return nil, errors.Errorf(
				"failed to read invocation state from channel on (%T) with ID %q: scriptInvocationStateReadChan was unexpectedly closed",
				tx, base58.Encode(id),
			)
		}
		if info.rideV5Activated {
			ia.sc.increaseRecentSpentComplexityAfterRideV5(is.Result, is.Err)
		} else {
			if invokeTx, ok := tx.(*proto.InvokeScriptWithProofs); ok {
				if err := ia.sc.increaseRecentSpentComplexityBeforeRideV5(scriptAddr, invokeTx.FunctionCall.Name, invokeTx.FunctionCall.Default); err != nil {
					return nil, errors.Wrap(err, "failed to increase recent spent complexity before RideV5")
				}
			} else {
				return nil, errors.Errorf("unknown tx type (%T) before RideV5 activation", tx)
			}
		}
		return is.Result, is.Err
	}
	var (
		tree *ast.Tree
		err  error
	)
	switch tx := tx.(type) {
	case *proto.InvokeExpressionTransactionWithProofs:
		tree, err = serialization.Parse(tx.Expression)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse decoded invoke expression into tree")
		}
	case *proto.EthereumTransaction, *proto.InvokeScriptWithProofs:
		tree, err = ia.stor.scriptsStorage.newestScriptByAddr(scriptAddr)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to instantiate script on address '%s'", scriptAddr.String())
		}
	default:
		return nil, errors.Errorf("failed to apply an invoke script: unexpected type of transaction (%T)", tx)
	}
	r, err := ia.sc.invokeFunction(tree, tx, info, scriptAddr)
	if ia.scriptInvocationStateWriteChan != nil {
		ia.scriptInvocationStateWriteChan <- ride.AnyScriptInvocationStateWithHeight{
			Height: info.blockInfo.Height,
			State:  ride.AnyScriptInvocationState{Result: r, Err: err},
		}
	}
	return r, err
}

type invocationResult struct {
	failed bool
	code   proto.TxFailureReason
	text   string

	scriptRuns uint64
	actions    []proto.ScriptAction
	changes    txBalanceChanges
}

func toScriptResult(ir *invocationResult) (*proto.ScriptResult, error) {
	errorMsg := proto.ScriptErrorMessage{}
	if ir.failed {
		errorMsg = proto.ScriptErrorMessage{Code: ir.code, Text: ir.text}
	}
	sr, _, err := proto.NewScriptResult(ir.actions, errorMsg)
	return sr, err
}

func (ia *invokeApplier) handleInvocationResult(txID crypto.Digest, info *fallibleValidationParams, res *invocationResult) (*applicationResult, error) {
	if ia.buildApiData && !info.validatingUtx {
		// Save invoke result for extended API.
		res, err := toScriptResult(res)
		if err != nil {
			return nil, errors.Wrap(err, "failed to save script result")
		}
		if err := ia.stor.invokeResults.saveResult(txID, res, info.block.BlockID()); err != nil {
			return nil, errors.Wrap(err, "failed to save script result")
		}
	}
	// Total scripts invoked = scriptRuns + invocation itself.
	totalScriptsInvoked := res.scriptRuns + 1
	return &applicationResult{
		totalScriptsRuns: totalScriptsInvoked,
		changes:          res.changes,
		status:           !res.failed,
	}, nil
}

func (ia *invokeApplier) checkFullFee(tx proto.Transaction, scriptRuns, issuedAssetsCount uint64) error {
	sponsorshipActivated, err := ia.stor.features.newestIsActivated(int16(settings.FeeSponsorship))
	if err != nil {
		return err
	}
	if !sponsorshipActivated {
		// Minimum fee is not checked before sponsorship activation.
		return nil
	}
	minIssueFee := feeConstants[proto.IssueTransaction] * FeeUnit * issuedAssetsCount
	minWavesFee := scriptExtraFee*scriptRuns + feeConstants[proto.InvokeScriptTransaction]*FeeUnit + minIssueFee

	wavesFee := tx.GetFee()

	var feeAssetStr string
	switch t := tx.(type) {
	case *proto.InvokeScriptWithProofs:
		if t.FeeAsset.Present {
			wavesFee, err = ia.stor.sponsoredAssets.sponsoredAssetToWaves(proto.AssetIDFromDigest(t.FeeAsset.ID), t.Fee)
			if err != nil {
				return errs.Extend(err, "failed to convert fee asset to waves")
			}

		}
		feeAssetStr = t.FeeAsset.String()
	case *proto.EthereumTransaction:
		wavesAsset := proto.NewOptionalAssetWaves()
		feeAssetStr = wavesAsset.String()

	}

	if wavesFee < minWavesFee {
		return errs.NewFeeValidation(fmt.Sprintf(
			"Fee in %s for InvokeScriptTransaction (%d in %s) with %d total scripts invoked does not exceed minimal value of %d WAVES",
			feeAssetStr, tx.GetFee(), feeAssetStr, scriptRuns, minWavesFee))
	}

	return nil
}

func (ia *invokeApplier) validateActionSmartAsset(asset crypto.Digest, action proto.ScriptAction, callerPK crypto.PublicKey,
	txID crypto.Digest, txTimestamp uint64, params *appendTxParams) (bool, error) {
	isSmartAsset, err := ia.stor.scriptsStorage.newestIsSmartAsset(proto.AssetIDFromDigest(asset))
	if err != nil {
		return false, err
	}
	if !isSmartAsset {
		return true, nil
	}
	env, err := ride.NewEnvironment(
		ia.settings.AddressSchemeCharacter,
		ia.state,
		ia.settings.InternalInvokePaymentsValidationAfterHeight,
		params.blockV5Activated,
		params.rideV6Activated,
		params.consensusImprovementsActivated,
		params.invokeExpressionActivated,
	)
	if err != nil {
		return false, err
	}

	setTx := func(env *ride.EvaluationEnvironment) error {
		return env.SetTransactionFromScriptAction(action, callerPK, txID, txTimestamp)
	}

	res, err := ia.sc.callAssetScriptCommon(env, setTx, asset, params)
	if err != nil {
		return false, err
	}
	return res.Result(), nil
}
