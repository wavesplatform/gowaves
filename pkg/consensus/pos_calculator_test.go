package consensus

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFairPosCalculateDelay(t *testing.T) {
	fpos := FairPosCalculatorV1
	var hit big.Int
	hit.SetString("1", 10)
	delay, err := fpos.CalculateDelay(&hit, 100, 10000000000000)
	if err != nil {
		t.Fatalf("fpos.calculateDelay(): %v\n", err)
	}
	assert.Equal(t, uint64(705491), delay, "invalid fair PoS delay")
	hit.SetString("2", 10)
	delay, err = fpos.CalculateDelay(&hit, 200, 20000000000000)
	if err != nil {
		t.Fatalf("fpos.calculateDelay(): %v\n", err)
	}
	assert.Equal(t, uint64(607358), delay, "invalid fair PoS delay")
	hit.SetString("3", 10)
	delay, err = fpos.CalculateDelay(&hit, 300, 30000000000000)
	if err != nil {
		t.Fatalf("fpos.calculateDelay(): %v\n", err)
	}
	assert.Equal(t, uint64(549956), delay, "invalid fair PoS delay")
}

func TestFairPosCalculateBaseTarget(t *testing.T) {
	fpos := FairPosCalculatorV1
	target, err := fpos.CalculateBaseTarget(100, 30, 100, 100000000000, 99000, 100000)
	if err != nil {
		t.Fatalf("fpos.calculateBaseTarget(): %v\n", err)
	}
	assert.Equal(t, uint64(99), target, "invalid fair PoS target")
	target, err = fpos.CalculateBaseTarget(100, 10, 100, 100000000000, 0, 100000000000)
	if err != nil {
		t.Fatalf("fpos.calculateBaseTarget(): %v\n", err)
	}
	assert.Equal(t, uint64(100), target, "invalid fair PoS target")
	target, err = fpos.CalculateBaseTarget(100, 10, 100, 100000000000, 99999700000, 100000000000)
	if err != nil {
		t.Fatalf("fpos.calculateBaseTarget(): %v\n", err)
	}
	assert.Equal(t, uint64(100), target, "invalid fair PoS target")
	target, err = fpos.CalculateBaseTarget(100, 30, 100, 100000000000, 1, 1000000)
	if err != nil {
		t.Fatalf("fpos.calculateBaseTarget(): %v\n", err)
	}
	assert.Equal(t, uint64(101), target, "invalid fair PoS target")
}

func TestNxtPosCalculateDelay(t *testing.T) {
	nxt := NXTPosCalculator
	var hit big.Int
	hit.SetString("7351874400125134246", 10)
	delay, err := nxt.CalculateDelay(&hit, 334160, 500162462800)
	if err != nil {
		t.Fatalf("nxt.calculateDelay(): %v\n", err)
	}
	assert.Equal(t, uint64(44000), delay, "invalid nxt delay")
	hit.SetString("11824069987143516706", 10)
	delay, err = nxt.CalculateDelay(&hit, 704498270, 100001283)
	if err != nil {
		t.Fatalf("nxt.calculateDelay(): %v\n", err)
	}
	assert.Equal(t, uint64(168000), delay, "invalid nxt delay")
	hit.SetString("1191797677384316995", 10)
	delay, err = nxt.CalculateDelay(&hit, 786689734, 100001062)
	if err != nil {
		t.Fatalf("nxt.calculateDelay(): %v\n", err)
	}
	assert.Equal(t, uint64(16000), delay, "invalid nxt delay")
}

func TestNxtPosCalculateBaseTarget(t *testing.T) {
	nxt := NXTPosCalculator
	target, err := nxt.CalculateBaseTarget(60, 200000, 309209, 1477353355327, 1477353205129, 1477353460467)
	if err != nil {
		t.Fatalf("nxt.calculateBaseTarget(): %v\n", err)
	}
	assert.Equal(t, uint64(345283), target, "invalid nxt base target")
	target, err = nxt.CalculateBaseTarget(60, 7160, 704498270, 1466167572675, 1466167287305, 1466167602106)
	if err != nil {
		t.Fatalf("nxt.calculateBaseTarget(): %v\n", err)
	}
	assert.Equal(t, uint64(786689734), target, "invalid nxt base target")
	target, err = nxt.CalculateBaseTarget(60, 7163, 727950233, 1466167672163, 1466167602106, 1466167703511)
	if err != nil {
		t.Fatalf("nxt.calculateBaseTarget(): %v\n", err)
	}
	assert.Equal(t, uint64(727950233), target, "invalid nxt base target")
}
