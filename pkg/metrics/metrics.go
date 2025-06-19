package metrics

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/ccoveille/go-safecast"
	influx "github.com/influxdata/influxdb1-client/v2"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

const (
	defaultTimeout = 5 * time.Second
	defaultPort    = 8086
	reportInterval = time.Second
	bufferSize     = 2000

	eventInv           = "Inv"
	eventReceived      = "Received"
	eventApplied       = "Applied"
	eventAppended      = "Appended"
	eventDeclined      = "Declined"
	eventMined         = "Mined"
	eventScore         = "Score"
	eventUtx           = "Utx"
	eventFSMChannelLen = "FSMChannelLength"
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

func MicroBlockDeclined(mb *proto.MicroBlock) {
	if rep == nil {
		return
	}
	t := newTags().withMicro().withEvent(eventDeclined).withID(mb.TotalBlockID).withParentID(mb.Reference)
	f := newFields()
	reportBlock(t, f)
}

func MicroBlockApplied(mb *proto.MicroBlock) {
	if rep == nil {
		return
	}
	t := newTags().withMicro().withEvent(eventApplied).withID(mb.TotalBlockID).withParentID(mb.Reference)
	f := newFields().withTransactionsCount(int(mb.TransactionCount))
	reportBlock(t, f)
}

func BlockReceived(block *proto.Block, source string) {
	if rep == nil {
		return
	}
	t := newTags().withBlock().withEvent(eventReceived).withID(block.ID).withBroadcast().withParentID(block.Parent)
	f := newFields().withSourceNode(source).withBaseTarget(block.BaseTarget).withID(block.ID)
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

// BlockAppended TODO remove it?
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
	t := newTags().withBlock().withEvent(eventApplied).withID(block.ID).withParentID(block.Parent).withBroadcast()
	f := newFields().withHeight(height).withTransactionsCount(block.TransactionCount).withID(block.ID)
	reportBlock(t, f)
}

func SnapshotBlockApplied(block *proto.Block, height proto.Height) {
	if rep == nil {
		return
	}
	t := newTags().withSnapshot().withEvent(eventApplied).withID(block.ID).withParentID(block.Parent).withBroadcast()
	f := newFields().withHeight(height).withTransactionsCount(block.TransactionCount).withID(block.ID)
	reportBlock(t, f)
}

func BlockDeclined(block *proto.Block) {
	if rep == nil {
		return
	}
	t := newTags().withBlock().withEvent(eventDeclined).withID(block.ID).withParentID(block.Parent).withBroadcast()
	f := newFields()
	reportBlock(t, f)
}

func BlockDeclinedFromExtension(block *proto.Block) {
	if rep == nil {
		return
	}
	t := newTags().withBlock().withEvent(eventDeclined).withID(block.ID).withParentID(block.Parent).withExtension()
	f := newFields()
	reportBlock(t, f)
}

func BlockAppliedFromExtension(block *proto.Block, height proto.Height) {
	if rep == nil {
		return
	}
	t := newTags().withBlock().withEvent(eventApplied).withID(block.ID).withParentID(block.Parent).withExtension()
	f := newFields().withHeight(height).withTransactionsCount(block.TransactionCount)
	reportBlock(t, f)
}

func BlockMined(block *proto.Block) {
	if rep == nil {
		return
	}
	t := newTags().withBlock().withEvent(eventMined).withID(block.ID).withParentID(block.Parent).withBroadcast()
	f := newFields().withTransactionsCount(block.TransactionCount).withBaseTarget(block.BaseTarget).
		withID(block.ID)
	reportBlock(t, f)
}

// MicroBlockMined must show the total tx count in block.
func MicroBlockMined(mb *proto.MicroBlock, totalTxCount int) {
	if rep == nil {
		return
	}
	t := newTags().withMicro().withEvent(eventMined).withID(mb.TotalBlockID).withParentID(mb.Reference)
	f := newFields().withTransactionsCount(totalTxCount)
	reportBlock(t, f)
}

func Score(score *proto.Score, source string) {
	if rep == nil {
		return
	}
	t := emptyTags().node().withEvent(eventScore)
	f := emptyFields().score(score).source(source)
	reportBlock(t, f)
}

func Utx(utxCount int) {
	if rep == nil {
		return
	}
	t := emptyTags().node().withEvent(eventUtx)
	f := emptyFields().withUtxCount(utxCount)
	reportUtx(t, f)
}

func FSMChannelLength(length int) {
	if rep == nil {
		return
	}
	t := emptyTags().node().withEvent(eventFSMChannelLen)
	f := emptyFields().withChannelLength(length)
	reportFSMChannelLength(t, f)
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

func (t tags) withSnapshot() tags {
	t["type"] = "Snapshot"
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

func (f fields) withID(id proto.BlockID) fields {
	f["block_id"] = id.String()
	return f
}

func (f fields) source(source string) fields {
	f["source"] = source
	return f
}

func (f fields) score(score *proto.Score) fields {
	f["score"] = score.String()
	return f
}

func (f fields) withUtxCount(utxCount int) fields {
	f["utx_count"] = utxCount
	return f
}

func (f fields) withBaseTarget(bt uint64) fields {
	baseTarget, err := safecast.ToInt64(bt)
	if err != nil {
		zap.S().Errorf("failed to execute withBaseTarget, %v", err)
	}
	f["bt"] = baseTarget
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

func (f fields) withChannelLength(chLength int) fields {
	f["channel-size"] = chLength
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
				zap.S().Warnf("Failed to close connection to InfluxDB: %v", err)
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
		zap.S().Warnf("Failed to create metrics point 'block': %v", err)
		return
	}
	rep.in <- p
}

func reportUtx(t tags, f fields) {
	p, err := influx.NewPoint("utx", t, f, time.Now())
	if err != nil {
		zap.S().Warnf("Failed to create metrics point 'utx': %v", err)
		return
	}
	rep.in <- p
}

func reportFSMChannelLength(t tags, f fields) {
	p, err := influx.NewPoint("fsm-channel", t, f, time.Now())
	if err != nil {
		zap.S().Warnf("Failed to create metrics point 'fsm-channel': %v", err)
		return
	}
	rep.in <- p
}

func shortID(id proto.BlockID) string {
	return id.String()[:6]
}
