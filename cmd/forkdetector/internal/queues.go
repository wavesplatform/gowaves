package internal

import (
	"bytes"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

type requestQueue struct {
	picked      int
	blocks      []crypto.Signature
	connections map[crypto.Signature][]*Conn
	once        sync.Once
	rnd         *rand.Rand
}

func (q *requestQueue) init() {
	q.picked = -1
	q.blocks = make([]crypto.Signature, 0)
	q.connections = make(map[crypto.Signature][]*Conn)
	q.rnd = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func (q *requestQueue) String() string {
	q.once.Do(q.init)

	sb := strings.Builder{}
	sb.WriteRune('(')
	sb.WriteString(strconv.Itoa(len(q.connections)))
	sb.WriteRune(')')
	sb.WriteRune('[')
	for i, s := range q.blocks {
		if i != 0 {
			sb.WriteRune(' ')
		}
		ss := s.String()
		sb.WriteString(ss[:6])
		sb.WriteRune('.')
		sb.WriteRune('.')
		sb.WriteString(ss[len(ss)-6:])
		if i == q.picked {
			sb.WriteRune(' ')
			sb.WriteRune('|')
		}
	}
	sb.WriteRune(']')
	return sb.String()
}

func (q *requestQueue) enqueue(block crypto.Signature, conn *Conn) {
	q.once.Do(q.init)

	if conn == nil {
		zap.S().Fatalf("Attempt to insert NIL connection into queue")
	}

	list, ok := q.connections[block]
	if ok {
		list = append(list, conn)
		q.connections[block] = list
		return
	}
	q.blocks = append(q.blocks, block)
	list = []*Conn{conn}
	q.connections[block] = list
}

func (q *requestQueue) pickRandomly(exclusion []*Conn) (crypto.Signature, *Conn, bool) {
	q.once.Do(q.init)

	if q.picked == len(q.blocks)-1 {
		return crypto.Signature{}, nil, false
	}
	q.picked++
	sig := q.blocks[q.picked]
	connections, ok := q.connections[sig]
	if !ok {
		zap.S().Fatalf("Failure to locate enqueued connection")
	}
	filtered := q.exclude(connections, exclusion)
	if len(filtered) == 0 {
		filtered = connections
	}
	conn := filtered[q.rnd.Intn(len(filtered))]
	return sig, conn, true
}

func (q *requestQueue) dequeue(block crypto.Signature) {
	q.once.Do(q.init)

	ok, pos := contains(q.blocks, block)
	if !ok {
		return
	}
	q.blocks = q.blocks[:pos+copy(q.blocks[pos:], q.blocks[pos+1:])]
	delete(q.connections, block)
	q.picked--
}

func (q *requestQueue) reset() {
	q.picked = -1
}

func (q *requestQueue) unpick() {
	q.picked--
}

func (q *requestQueue) exclude(connections, exclusion []*Conn) []*Conn {
	r := make([]*Conn, 0)
	for _, c := range connections {
		if !q.contains(exclusion, c) {
			r = append(r, c)
		}
	}
	return r
}

func (q *requestQueue) contains(connections []*Conn, conn *Conn) bool {
	for _, c := range connections {
		if c == conn {
			return true
		}
	}
	return false
}

type pendingItem struct {
	sig   crypto.Signature
	conn  *Conn
	block *proto.Block
}

type pendingQueue struct {
	items []pendingItem
}

func (q *pendingQueue) String() string {
	sb := strings.Builder{}
	sb.WriteRune('[')
	for i, pi := range q.items {
		if i != 0 {
			sb.WriteRune(' ')
		}
		sb.WriteString(pi.sig.ShortString())
		if pi.block != nil {
			sb.WriteRune('(')
			sb.WriteRune(0x2713)
			sb.WriteRune(')')
		} else {
			sb.WriteRune('(')
			sb.WriteRune(' ')
			sb.WriteRune(')')
		}
	}
	sb.WriteRune(']')
	return sb.String()
}

func (q *pendingQueue) len() int {
	return len(q.items)
}

func (q *pendingQueue) connections() []*Conn {
	r := make([]*Conn, 0)
	for _, i := range q.items {
		r = append(r, i.conn)
	}
	return r
}

func (q *pendingQueue) enqueue(sig crypto.Signature, conn *Conn) {
	q.items = append(q.items, pendingItem{sig: sig, conn: conn})
}

func (q *pendingQueue) dequeue() (*proto.Block, bool) {
	if len(q.items) == 0 || q.items[0].block == nil {
		return nil, false
	}
	var i pendingItem
	i, q.items = q.items[0], q.items[1:]
	return i.block, true
}

func (q *pendingQueue) update(block *proto.Block) {
	for i := 0; i < len(q.items); i++ {
		if bytes.Equal(q.items[i].sig[:], block.BlockSignature[:]) {
			q.items[i].block = block
			break
		}
	}
}
