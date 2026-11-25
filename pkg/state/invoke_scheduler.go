package state

import (
	"slices"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

// Scheduling of speculative invoke evaluation.
//
// Before an invoke is executed, the accesses of the upcoming transactions of the block are predicted: from
// statistics for invokes and from the transaction body for simple transactions. Predictions are used to build
// a dependency graph and to select a batch of invokes that are expected to be independent of all transactions
// that are not applied yet, so their scripts can be evaluated in parallel over the current state.
//
// Everything here is advisory. A wrong prediction costs a wasted evaluation and never affects the result of
// the block: speculative results are validated by actual read and write sets before use.
// Functions of this file are pure: the state is reached only through the resolvers of accessPredictor.

// predictedAccess is the predicted access of a transaction.
type predictedAccess struct {
	// reads and writes are the keys the transaction is expected to read and to modify. For simple transactions
	// only alias lookups are listed in reads: simple transactions are never speculated, so only their writes
	// matter, and a write is treated as a read by the dependency graph anyway.
	reads, writes accessSet
	// barrier is set if the access of the transaction can not be predicted. A barrier depends on every earlier
	// transaction and every later transaction depends on it.
	barrier bool
	// speculatable is set for invokes which script evaluation can be performed speculatively.
	speculatable bool
}

func newPredictedAccess() predictedAccess {
	return predictedAccess{reads: newAccessSet(), writes: newAccessSet()}
}

func barrierAccess() predictedAccess {
	res := newPredictedAccess()
	res.barrier = true
	return res
}

// accessPredictor predicts accesses of transactions. Resolvers are the only way it reaches the state.
type accessPredictor struct {
	scheme proto.Scheme
	// stats returns statistics of the callable function or nil if there are none. Nil resolver means that
	// there are no statistics at all.
	stats func(key invokeStatsKey) *invokeStats
	// resolveAlias returns the address of the alias, the second result is false if the alias is unknown.
	// Nil resolver means that no alias is resolvable.
	resolveAlias func(alias proto.Alias) (proto.AddressID, bool)
}

func (p *accessPredictor) senderID(pk crypto.PublicKey) (proto.AddressID, bool) {
	addr, err := proto.NewAddressFromPublicKey(p.scheme, pk)
	if err != nil {
		return proto.AddressID{}, false
	}
	return addr.ID(), true
}

// recipientID returns the address of the recipient. For an alias recipient the read of the alias is added to
// the prediction, and the second result is false if the alias is not resolvable.
func (p *accessPredictor) recipientID(res *predictedAccess, r proto.Recipient) (proto.AddressID, bool) {
	if addr := r.Address(); addr != nil {
		return addr.ID(), true
	}
	alias := r.Alias()
	if alias == nil {
		return proto.AddressID{}, false
	}
	res.reads.add(aliasAccessKey(alias.Alias))
	if p.resolveAlias == nil {
		return proto.AddressID{}, false
	}
	return p.resolveAlias(*alias)
}

// addRecipientBalance adds the write of the balance of the recipient. The balance of an unresolvable alias
// recipient is left unpredicted: the alias is either created earlier in the block or the transaction fails.
func (p *accessPredictor) addRecipientBalance(res *predictedAccess, r proto.Recipient) {
	if id, ok := p.recipientID(res, r); ok {
		res.writes.add(accountBalanceAccessKey(id))
	}
}

// simple predicts the access of a simple transaction: the balance of the sender, which pays the fee, and the
// keys given by the type of the transaction. Balances changed by the fee itself are not predicted completely:
// the sponsor of the fee asset is known only to the state.
func (p *accessPredictor) simple(
	senderPK crypto.PublicKey, fill func(res *predictedAccess, sender proto.AddressID),
) predictedAccess {
	sender, ok := p.senderID(senderPK)
	if !ok {
		return barrierAccess()
	}
	res := newPredictedAccess()
	res.writes.add(accountBalanceAccessKey(sender))
	fill(&res, sender)
	return res
}

func (p *accessPredictor) transfer(tx proto.Transfer) predictedAccess {
	return p.simple(tx.SenderPK, func(res *predictedAccess, _ proto.AddressID) {
		p.addRecipientBalance(res, tx.Recipient)
	})
}

// issue predicts the access of an issue transaction, the ID of the new asset is the ID of the transaction.
func (p *accessPredictor) issue(tx proto.Issue, id *crypto.Digest) predictedAccess {
	return p.simple(tx.SenderPK, func(res *predictedAccess, _ proto.AddressID) {
		if id != nil {
			res.writes.add(assetAccessKey(*id))
		}
	})
}

func (p *accessPredictor) asset(senderPK crypto.PublicKey, assetID crypto.Digest) predictedAccess {
	return p.simple(senderPK, func(res *predictedAccess, _ proto.AddressID) {
		res.writes.add(assetAccessKey(assetID))
	})
}

// lease predicts the access of a lease transaction, the ID of the new lease is the ID of the transaction.
func (p *accessPredictor) lease(tx proto.Lease, id *crypto.Digest) predictedAccess {
	return p.simple(tx.SenderPK, func(res *predictedAccess, _ proto.AddressID) {
		p.addRecipientBalance(res, tx.Recipient)
		if id != nil {
			res.writes.add(leaseAccessKey(*id))
		}
	})
}

func (p *accessPredictor) createAlias(tx proto.CreateAlias) predictedAccess {
	return p.simple(tx.SenderPK, func(res *predictedAccess, _ proto.AddressID) {
		res.writes.add(aliasAccessKey(tx.Alias.Alias))
	})
}

// exchange predicts the access of an exchange transaction: balances of the matcher, which is the sender of
// the transaction, and of the senders of both orders.
func (p *accessPredictor) exchange(tx proto.Exchange) predictedAccess {
	orders := [...]proto.Order{tx.GetOrder1(), tx.GetOrder2()}
	ids := make([]proto.AddressID, 0, len(orders))
	for _, o := range orders {
		if o == nil {
			return barrierAccess()
		}
		addr, err := o.GetSender(p.scheme)
		if err != nil {
			return barrierAccess()
		}
		ids = append(ids, addr.ID())
	}
	return p.simple(tx.GetSenderPK(), func(res *predictedAccess, _ proto.AddressID) {
		for _, id := range ids {
			res.writes.add(accountBalanceAccessKey(id))
		}
	})
}

// invoke predicts the access of an invoke script transaction: static keys known from the body of the
// transaction combined with the statistics of the called function.
//
// If there are no statistics yet, the prediction is optimistic: only static keys are used, so the first
// execution is speculated freely and creates the statistics.
func (p *accessPredictor) invoke(tx *proto.InvokeScriptWithProofs) predictedAccess {
	sender, ok := p.senderID(tx.SenderPK)
	if !ok {
		return barrierAccess()
	}
	res := newPredictedAccess()
	res.speculatable = true
	// The sender pays the fee and payments.
	res.reads.add(accountBalanceAccessKey(sender))
	res.writes.add(accountBalanceAccessKey(sender))
	for _, payment := range tx.Payments {
		if payment.Asset.Present {
			res.reads.add(assetAccessKey(payment.Asset.ID))
		}
	}
	dApp, ok := p.recipientID(&res, tx.ScriptRecipient)
	if !ok {
		// The dApp is called by an alias that does not exist yet. The read of the alias makes the invoke
		// dependent on the transaction that creates it, nothing else is known.
		return res
	}
	res.reads.add(accountScriptAccessKey(dApp))
	res.reads.add(accountDataAccessKey(dApp))
	res.reads.add(accountBalanceAccessKey(dApp))
	if len(tx.Payments) != 0 {
		res.writes.add(accountBalanceAccessKey(dApp))
	}
	if p.stats == nil {
		return res
	}
	stats := p.stats(invokeStatsKey{dApp: dApp, function: tx.FunctionCall.Name()})
	if stats == nil {
		return res
	}
	if stats.wide {
		res.barrier = true
		return res
	}
	reads, writes := instantiate(stats, sender, dApp)
	res.reads.union(reads)
	res.writes.union(writes)
	return res
}

// predictTxAccess predicts the access of the transaction. Transactions of types that are not handled
// explicitly are barriers.
func (p *accessPredictor) predictTxAccess(tx proto.Transaction) predictedAccess {
	switch t := tx.(type) {
	case *proto.InvokeScriptWithProofs:
		return p.invoke(t)
	case *proto.Payment:
		return p.simple(t.SenderPK, func(res *predictedAccess, _ proto.AddressID) {
			res.writes.add(accountBalanceAccessKey(t.Recipient.ID()))
		})
	case *proto.TransferWithSig:
		return p.transfer(t.Transfer)
	case *proto.TransferWithProofs:
		return p.transfer(t.Transfer)
	case *proto.MassTransferWithProofs:
		return p.simple(t.SenderPK, func(res *predictedAccess, _ proto.AddressID) {
			for i := range t.Transfers {
				p.addRecipientBalance(res, t.Transfers[i].Recipient)
			}
		})
	case *proto.LeaseWithSig:
		return p.lease(t.Lease, t.ID)
	case *proto.LeaseWithProofs:
		return p.lease(t.Lease, t.ID)
	case *proto.DataWithProofs:
		return p.simple(t.SenderPK, func(res *predictedAccess, sender proto.AddressID) {
			res.writes.add(accountDataAccessKey(sender))
		})
	case *proto.SetScriptWithProofs:
		return p.simple(t.SenderPK, func(res *predictedAccess, sender proto.AddressID) {
			res.writes.add(accountScriptAccessKey(sender))
		})
	case *proto.IssueWithSig:
		return p.issue(t.Issue, t.ID)
	case *proto.IssueWithProofs:
		return p.issue(t.Issue, t.ID)
	case *proto.ReissueWithSig:
		return p.asset(t.SenderPK, t.AssetID)
	case *proto.ReissueWithProofs:
		return p.asset(t.SenderPK, t.AssetID)
	case *proto.BurnWithSig:
		return p.asset(t.SenderPK, t.AssetID)
	case *proto.BurnWithProofs:
		return p.asset(t.SenderPK, t.AssetID)
	case *proto.SponsorshipWithProofs:
		return p.asset(t.SenderPK, t.AssetID)
	case *proto.SetAssetScriptWithProofs:
		return p.asset(t.SenderPK, t.AssetID)
	case *proto.UpdateAssetInfoWithProofs:
		return p.asset(t.SenderPK, t.AssetID)
	case *proto.CreateAliasWithSig:
		return p.createAlias(t.CreateAlias)
	case *proto.CreateAliasWithProofs:
		return p.createAlias(t.CreateAlias)
	case *proto.ExchangeWithSig:
		return p.exchange(t)
	case *proto.ExchangeWithProofs:
		return p.exchange(t)
	default:
		// Genesis, lease cancel (the recipient of the lease is known only to the state), commitment to
		// generation, Ethereum transactions, invoke expression and everything unknown.
		return barrierAccess()
	}
}

// dependencyGraph is the graph of predicted dependencies between transactions of a window.
type dependencyGraph struct {
	// incoming holds for every transaction of the window the sorted indexes of earlier transactions
	// it depends on.
	incoming [][]int
}

// buildDependencyGraph builds the dependency graph of the window of consecutive transactions. There is an
// edge from transaction i to a later transaction j if the writes of i intersect the reads or the writes of j,
// or if any of the two is a barrier.
func buildDependencyGraph(window []predictedAccess) dependencyGraph {
	b := dependencyGraphBuilder{
		window:  window,
		writers: make(map[accessKey][]int),
		mark:    make([]int, len(window)),
	}
	g := dependencyGraph{incoming: make([][]int, len(window))}
	for j := range window {
		g.incoming[j] = b.next(j)
	}
	return g
}

// dependencyGraphBuilder holds the state of the dependency graph construction.
type dependencyGraphBuilder struct {
	window []predictedAccess
	// writers holds indexes of the processed transactions by the keys they write.
	writers map[accessKey][]int
	// barriers holds indexes of the processed barriers.
	barriers []int
	// mark[i] == j+1 means that the edge from i to j is already added.
	mark []int
	deps []int
}

func (b *dependencyGraphBuilder) add(i, j int) {
	if b.mark[i] != j+1 {
		b.mark[i] = j + 1
		b.deps = append(b.deps, i)
	}
}

func (b *dependencyGraphBuilder) addWriters(set accessSet, j int) {
	for k := range set {
		for _, i := range b.writers[k] {
			b.add(i, j)
		}
	}
}

// next returns the sorted dependencies of transaction j. Transactions must be passed in order.
func (b *dependencyGraphBuilder) next(j int) []int {
	tx := b.window[j]
	b.deps = nil
	if tx.barrier {
		for i := range j {
			b.add(i, j)
		}
		b.barriers = append(b.barriers, j)
	} else {
		for _, i := range b.barriers {
			b.add(i, j)
		}
		b.addWriters(tx.reads, j)
		b.addWriters(tx.writes, j)
	}
	for k := range tx.writes {
		b.writers[k] = append(b.writers[k], j)
	}
	slices.Sort(b.deps)
	return b.deps
}

// selectBatch returns indexes of invokes of the window to evaluate speculatively. The window is the sequence
// of not yet applied transactions of the block starting from the next one to apply; skip tells which
// of them must not be selected, because they already hold a speculative result, it may be shorter than the
// window.
//
// An invoke is selected if it is speculatable, is not skipped and does not depend on any
// earlier transaction of the window. The head of the window has no predecessors, so it always qualifies when
// it is a speculatable invoke. Only the first invokeSpeculationLookahead transactions are considered and
// nothing is selected after the first barrier.
func selectBatch(window []predictedAccess, skip []bool) []int {
	n := min(len(window), invokeSpeculationLookahead)
	for i := range n {
		if window[i].barrier {
			n = i + 1
			break
		}
	}
	g := buildDependencyGraph(window[:n])
	var batch []int
	for j := range n {
		if !window[j].speculatable || (j < len(skip) && skip[j]) {
			continue
		}
		if len(g.incoming[j]) == 0 {
			batch = append(batch, j)
		}
	}
	return batch
}
