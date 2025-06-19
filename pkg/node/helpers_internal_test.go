package node

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func testContext(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return ctx, cancel
}

func filledMsgChan(values ...proto.Message) chan peer.ProtoMessage {
	ch := make(chan peer.ProtoMessage, len(values))
	for _, v := range values {
		ch <- peer.ProtoMessage{
			ID:      nil, // ID is not used in this test.
			Message: v,
		}
	}
	return ch
}

func readProtoMessages(t *testing.T, ch <-chan peer.ProtoMessage, expectedMsgCount int) []proto.Message {
	timeout := time.NewTimer(5 * time.Second)
	messages := make([]proto.Message, 0, expectedMsgCount)
	for range expectedMsgCount {
		select {
		case <-timeout.C:
			t.Fatalf("timed out waiting for messages, check expectedMsgCount=%d", expectedMsgCount)
		case msg := <-ch:
			messages = append(messages, msg.Message)
		}
	}
	select {
	case msg := <-ch:
		t.Fatalf("unexpected message=%+v received, expected exactly %d messages", msg, expectedMsgCount)
	default:
		// No more messages expected, this is fine.
	}
	return messages
}

func TestDeduplicateProtoTxMessages(t *testing.T) {
	messages := filledMsgChan(
		&proto.TransactionMessage{Transaction: nil},
		&proto.TransactionMessage{Transaction: nil},   // duplicate
		&proto.PBTransactionMessage{Transaction: nil}, // considered as duplicate because payloads are equal
		&proto.TransactionMessage{Transaction: []byte{1, 2, 3}},
		&proto.TransactionMessage{Transaction: []byte{1, 2, 3}},   // duplicate
		&proto.PBTransactionMessage{Transaction: []byte{1, 2, 3}}, // duplicate, payloads are equal
		&proto.PBTransactionMessage{Transaction: []byte{42}},
		&proto.TransactionMessage{Transaction: []byte{42}}, // duplicate, payloads are equal
		// -- non-transaction messages
		&proto.ScoreMessage{Score: nil},
		&proto.ScoreMessage{Score: nil},
		&proto.ScoreMessage{Score: []byte{1, 2, 3}},
		&proto.ScoreMessage{Score: []byte{1, 2, 3}},
		&proto.ScoreMessage{Score: []byte{42}},
		&proto.ScoreMessage{Score: []byte{42}},
		&proto.ScoreMessage{Score: []byte{21}},
	)

	expected := []proto.Message{
		&proto.TransactionMessage{Transaction: nil},
		&proto.TransactionMessage{Transaction: []byte{1, 2, 3}},
		&proto.PBTransactionMessage{Transaction: []byte{42}},
		// -- non-transaction messages
		&proto.ScoreMessage{Score: nil},
		&proto.ScoreMessage{Score: nil},
		&proto.ScoreMessage{Score: []byte{1, 2, 3}},
		&proto.ScoreMessage{Score: []byte{1, 2, 3}},
		&proto.ScoreMessage{Score: []byte{42}},
		&proto.ScoreMessage{Score: []byte{42}},
		&proto.ScoreMessage{Score: []byte{21}},
	}

	ctx, cancel := testContext(t)
	deduplicated, _, wg := deduplicateProtoTxMessages(ctx, messages)
	defer wg.Wait()
	defer cancel()

	actual := readProtoMessages(t, deduplicated, len(expected))
	assert.ElementsMatch(t, expected, actual)
}
