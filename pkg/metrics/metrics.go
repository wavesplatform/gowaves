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

type tags map[string]string

func newTags() tags {
	t := make(map[string]string)
	t["node"] = strconv.Itoa(rep.id)
	return t
}

func (t tags) withHost() tags {
	t["host"] = strconv.Itoa(rep.id)
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

func newFields() fields {
	f := make(map[string]interface{})
	f["node"] = rep.id
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
	ticker    *time.Ticker
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
			ticker:    time.NewTicker(reportInterval),
			in:        make(chan *influx.Point, bufferSize),
		}
		go rep.run(ctx)
	})
	return nil
}

func (r *reporter) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			rep = nil
			err := r.c.Close()
			if err != nil {
				zap.S().Warn("Failed to close connection to InfluxDB: %v", err)
			}
			return
		case <-r.ticker.C:
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
	err = r.c.Write(batch)
	if err != nil {
		return err
	}
	return nil
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

func shortID(id proto.BlockID) string {
	return id.String()[:6]
}
