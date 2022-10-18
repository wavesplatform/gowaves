package metrics

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"sync"
	"time"

	influx "github.com/influxdata/influxdb1-client/v2"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

const (
	defaultTimeout = 5 * time.Second
	defaultPort    = 8086
	reportInterval = time.Second
	bufferSize     = 2000

	eventInv      = "Inv"
	eventReceived = "Received"
	eventApplied  = "Applied"
	eventAppended = "Appended"
	eventDeclined = "Declined"
	eventMined    = "Mined"
)

/*
Notes on InfluxDB schema design.

Both tags and fields are key-value pairs but with one significant difference is that tags are automatically indexed.
Because fields are not being indexed, every query where InfluxDB is asked to find a specified field, it needs to
sequentially scan every value of the field column. On the other hand, to index tags InfluxDB would try to construct an
inverted index in memory, which would always be growing with the cardinality.

A rule of thumb would be to persist highly dynamic values as fields and only use tags for GROUP BY clauses.

Tag keys and values are stored only once and always as strings, where field values and timestamps are stored per-point.
This however doesn't affect the memory footprint, only the storage requirements.

Excerpt from the documentation:

In general, your queries should guide what gets stored as a tag and what gets stored as a field:

  - Store commonly-queried meta data in tags.
  - Store data in fields if each data point contains a different value.
  - Store numeric values as fields (tag values only support string values).

Tags containing highly variable information like unique IDs, hashes, and random strings lead to a large number of
series, also known as high series cardinality.

High series cardinality is a primary driver of high memory usage for many database workloads. InfluxDB uses
measurements and tags to create indexes and speed up reads. However, when too many indexes created, both writes and
reads may start to slow down. Therefore, if a system has memory constraints, consider storing high-cardinality data as
a field rather than a tag.

Use the following conventions when naming your tag and field keys:

  - Avoid keywords in tag and field names
  - Avoid the same tag and field name
  - Avoid encoding data in measurement names
  - Avoid more than one piece of information in one tag
*/
var (
	once sync.Once
	rep  *reporter = nil
)

func MicroBlockInv(mb *proto.MicroBlockInv, source string) {
	if rep == nil {
		return
	}
	t := newTags().withMicro().withEvent(eventInv).withID(mb.TotalBlockID).withParentID(mb.Reference)
	f := newFields().withSourceNode(source)
	reportBlock(t, f)
}

func MicroBlockReceived(mb *proto.MicroBlock, source string) {
	if rep == nil {
		return
	}
	t := newTags().withMicro().withEvent(eventReceived).withID(mb.TotalBlockID).withParentID(mb.Reference)
	f := newFields().withSourceNode(source)
	reportBlock(t, f)
}

func MicroBlockApplied(mb *proto.MicroBlock) {
	if rep == nil {
		return
	}
	t := newTags().withMicro().withEvent(eventApplied).withID(mb.TotalBlockID)
	f := newFields().withTransactionsCount(int(mb.TransactionCount))
	reportBlock(t, f)
}

func BlockReceived(block *proto.Block, source string) {
	if rep == nil {
		return
	}
	t := newTags().withBlock().withEvent(eventReceived).withID(block.ID).withBroadcast()
	f := newFields().withSourceNode(source).withBaseTarget(block.BaseTarget)
	reportBlock(t, f)
}

func BlockReceivedFromExtension(block *proto.Block, source string) {
	if rep == nil {
		return
	}
	t := newTags().withBlock().withEvent(eventReceived).withID(block.ID).withExtension()
	f := newFields().withSourceNode(source).withBaseTarget(block.BaseTarget)
	reportBlock(t, f)
}

func BlockAppended(block *proto.Block, complexity int) {
	if rep == nil {
		return
	}
	t := newTags().withHost().withBlock().withEvent(eventAppended).withID(block.ID)
	f := newFields().withComplexity(complexity)
	reportBlock(t, f)
}

func BlockApplied(block *proto.Block, height proto.Height) {
	if rep == nil {
		return
	}
	t := newTags().withBlock().withEvent(eventApplied).withID(block.ID).withBroadcast()
	f := newFields().withHeight(height).withTransactionsCount(block.TransactionCount)
	reportBlock(t, f)
}

func BlockDeclined(block *proto.Block) {
	if rep == nil {
		return
	}
	t := newTags().withBlock().withEvent(eventDeclined).withID(block.ID).withBroadcast()
	f := newFields()
	reportBlock(t, f)
}

func BlockDeclinedFromExtension(block *proto.Block) {
	if rep == nil {
		return
	}
	t := newTags().withBlock().withEvent(eventDeclined).withID(block.ID).withExtension()
	f := newFields()
	reportBlock(t, f)
}

func BlockAppliedFromExtension(block *proto.Block, height proto.Height) {
	if rep == nil {
		return
	}
	t := newTags().withBlock().withEvent(eventApplied).withID(block.ID).withExtension()
	f := newFields().withHeight(height).withTransactionsCount(block.TransactionCount)
	reportBlock(t, f)
}

func BlockMined(block *proto.Block, height proto.Height) {
	if rep == nil {
		return
	}
	t := newTags().withBlock().withEvent(eventMined).withID(block.ID).withParentID(block.Parent).withBroadcast()
	f := newFields().withHeight(height).withTransactionsCount(block.TransactionCount).withBaseTarget(block.BaseTarget)
	reportBlock(t, f)
}

func FSMKeyBlockReceived(fsm string, block *proto.Block, source string) {
	if rep == nil {
		return
	}
	t := emptyTags().node().fsm(fsm).block().received()
	f := emptyFields().blockID(block.BlockID()).referenceID(block.Parent).source(source).blockTS(block.Timestamp).genPK(block.GeneratorPublicKey)
	reportFSM(t, f)
}

func FSMKeyBlockGenerated(fsm string, block *proto.Block) {
	if rep == nil {
		return
	}
	t := emptyTags().node().fsm(fsm).block().generated()
	f := emptyFields().blockID(block.BlockID()).referenceID(block.Parent)
	reportFSM(t, f)
}

func FSMKeyBlockApplied(fsm string, block *proto.Block) {
	if rep == nil {
		return
	}
	t := emptyTags().node().fsm(fsm).block().applied()
	f := emptyFields().blockID(block.BlockID()).referenceID(block.Parent)
	reportFSM(t, f)
}

func FSMKeyBlockDeclined(fsm string, block *proto.Block, err error) {
	if rep == nil {
		return
	}
	t := emptyTags().node().fsm(fsm).block().declined()
	f := emptyFields().blockID(block.BlockID()).referenceID(block.Parent).error(err)
	reportFSM(t, f)
}

func FSMMicroBlockReceived(fsm string, microblock *proto.MicroBlock, source string) {
	if rep == nil {
		return
	}
	t := emptyTags().node().fsm(fsm).microblock().received()
	f := emptyFields().blockID(microblock.TotalBlockID).referenceID(microblock.Reference).source(source)
	reportFSM(t, f)
}

func FSMMicroBlockGenerated(fsm string, microblock *proto.MicroBlock) {
	if rep == nil {
		return
	}
	t := emptyTags().node().fsm(fsm).microblock().generated()
	f := emptyFields().blockID(microblock.TotalBlockID).referenceID(microblock.Reference).signature(microblock.TotalResBlockSigField)
	reportFSM(t, f)
}

func FSMMicroBlockDeclined(fsm string, microblock *proto.MicroBlock, err error) {
	if rep == nil {
		return
	}
	t := emptyTags().node().fsm(fsm).microblock().declined()
	f := emptyFields().blockID(microblock.TotalBlockID).referenceID(microblock.Reference).signature(microblock.TotalResBlockSigField).error(err)
	reportFSM(t, f)
}

func FSMMicroBlockApplied(fsm string, microblock *proto.MicroBlock) {
	if rep == nil {
		return
	}
	t := emptyTags().node().fsm(fsm).microblock().applied()
	f := emptyFields().blockID(microblock.TotalBlockID).referenceID(microblock.Reference).signature(microblock.TotalResBlockSigField)
	reportFSM(t, f)
}

func FSMScore(fsm string, score *proto.Score, source string) {
	if rep == nil {
		return
	}
	t := emptyTags().node().fsm(fsm).score().received()
	f := emptyFields().score(score).source(source)
	reportFSM(t, f)
}

type tags map[string]string

func emptyTags() tags {
	t := make(map[string]string)
	return t
}

func newTags() tags {
	t := emptyTags()
	t["node"] = strconv.Itoa(rep.id)
	return t
}

func (t tags) withHost() tags {
	t["host"] = strconv.Itoa(rep.id)
	return t
}

func (t tags) node() tags {
	t["node"] = strconv.Itoa(rep.id)
	return t
}

func (t tags) fsm(fsm string) tags {
	t["fsm"] = fsm
	return t
}

func (t tags) block() tags {
	t["object"] = "block"
	return t
}

func (t tags) microblock() tags {
	t["object"] = "micro"
	return t
}

func (t tags) received() tags {
	t["event"] = "received"
	return t
}

func (t tags) generated() tags {
	t["event"] = "generated"
	return t
}

func (t tags) declined() tags {
	t["event"] = "declined"
	return t
}

func (t tags) applied() tags {
	t["event"] = "applied"
	return t
}

func (t tags) score() tags {
	t["object"] = "score"
	return t
}

func (t tags) withEvent(event string) tags {
	t["event"] = event
	return t
}

func (t tags) withID(id proto.BlockID) tags {
	t["id"] = shortID(id)
	return t
}

func (t tags) withParentID(id proto.BlockID) tags {
	t["parent-id"] = shortID(id)
	return t
}

func (t tags) withBlock() tags {
	t["type"] = "Block"
	return t
}

func (t tags) withMicro() tags {
	t["type"] = "Micro"
	return t
}

func (t tags) withBroadcast() tags {
	t["source"] = "Broadcast"
	return t
}

func (t tags) withExtension() tags {
	t["source"] = "Ext"
	return t
}

type fields map[string]interface{}

func emptyFields() fields {
	f := make(map[string]interface{})
	return f
}

func newFields() fields {
	f := emptyFields()
	f["node"] = rep.id
	return f
}

func (f fields) blockID(id proto.BlockID) fields {
	f["block_id"] = id.String()
	return f
}

func (f fields) source(source string) fields {
	f["source"] = source
	return f
}

func (f fields) referenceID(id proto.BlockID) fields {
	f["reference_id"] = id.String()
	return f
}

func (f fields) error(err error) fields {
	f["error"] = err.Error()
	return f
}

func (f fields) score(score *proto.Score) fields {
	f["score"] = score.String()
	return f
}

func (f fields) blockTS(ts uint64) fields {
	f["block_ts"] = ts
	return f
}

func (f fields) genPK(pk crypto.PublicKey) fields {
	f["gen_pk"] = pk.String()
	return f
}

func (f fields) signature(sig crypto.Signature) fields {
	f["sig"] = sig.String()
	return f
}

func (f fields) withBaseTarget(bt uint64) fields {
	f["bt"] = int(bt)
	return f
}

func (f fields) withComplexity(complexity int) fields {
	f["complexity"] = complexity
	return f
}

func (f fields) withSourceNode(name string) fields {
	f["from"] = name
	return f
}

func (f fields) withTransactionsCount(count int) fields {
	f["txs"] = count
	return f
}

func (f fields) withHeight(height proto.Height) fields {
	f["height"] = int(height)
	return f
}

type reporter struct {
	c         influx.Client
	id        int
	batchConf influx.BatchPointsConfig
	interval  time.Duration
	points    []*influx.Point
	in        chan *influx.Point
}

func Start(ctx context.Context, id int, url string) error {
	cfg, db, err := parseURL(url)
	if err != nil {
		return err
	}
	c, err := influx.NewHTTPClient(cfg)
	if err != nil {
		return err
	}
	d, v, err := c.Ping(defaultTimeout)
	if err != nil {
		return err
	}
	zap.S().Infof("InfluxDB/Telegraf %s replied in %s", v, d)
	if id < 0 {
		return errors.Errorf("invalid metrics ID %d", id)
	}
	once.Do(func() {
		rep = &reporter{
			c:         c,
			id:        id,
			batchConf: influx.BatchPointsConfig{Database: db},
			interval:  reportInterval,
			in:        make(chan *influx.Point, bufferSize),
		}
		go rep.run(ctx)
	})
	return nil
}

func (r *reporter) run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			rep = nil
			err := r.c.Close()
			if err != nil {
				zap.S().Warn("Failed to close connection to InfluxDB: %v", err)
			}
			return
		case <-ticker.C:
			err := r.report()
			if err != nil {
				zap.S().Warnf("Failed to report metrics: %v", err)
			}
			r.points = r.points[:0]
		case p := <-r.in:
			r.points = append(r.points, p)
		}
	}
}

func (r *reporter) report() error {
	batch, err := influx.NewBatchPoints(r.batchConf)
	if err != nil {
		return err
	}
	batch.AddPoints(r.points)
	return r.c.Write(batch)
}

func parseURL(s string) (influx.HTTPConfig, string, error) {
	uri, err := url.Parse(s)
	if err != nil {
		return influx.HTTPConfig{}, "", err
	}
	cfg := influx.HTTPConfig{}
	if uri.User != nil {
		cfg.Username = uri.User.Username()
		password, set := uri.User.Password()
		if set {
			cfg.Password = password
		}
	}
	ps := uri.Port()
	var p int
	if ps != "" {
		p, err = strconv.Atoi(ps)
		if err != nil {
			return influx.HTTPConfig{}, "", errors.Wrap(err, "invalid port number")
		}
		if p <= 0 || p > 65535 {
			return influx.HTTPConfig{}, "", errors.Errorf("invalid port number %d", p)
		}
	} else {
		p = defaultPort
	}

	cfg.Addr = fmt.Sprintf("%s://%s:%d", uri.Scheme, uri.Hostname(), p)
	db := path.Base(path.Clean(uri.Path))
	if db == "." || db == "/" || db == "" {
		return influx.HTTPConfig{}, "", errors.New("empty database")
	}
	return cfg, db, nil
}

func reportBlock(t tags, f fields) {
	p, err := influx.NewPoint("block", t, f, time.Now())
	if err != nil {
		zap.S().Warn("Failed to create metrics point 'block': %v", err)
		return
	}
	rep.in <- p
}

func reportFSM(t tags, f fields) {
	p, err := influx.NewPoint("fsm", t, f, time.Now())
	if err != nil {
		zap.S().Warn("Failed to create metrics point 'fsm': %v", err)
		return
	}
	rep.in <- p
}

func shortID(id proto.BlockID) string {
	return id.String()[:6]
}
