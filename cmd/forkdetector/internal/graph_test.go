package internal

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math/rand"
	"runtime"
	"testing"
	"time"
)

func TestNewGraph(t *testing.T) {
	g := newGraph()
	require.NotNil(t, g)

	assert.True(t, g.edge(2, 1))
	assert.True(t, g.edge(3, 2))
	assert.True(t, g.edge(4, 2))

	assert.False(t, g.edge(1, 0))
	assert.False(t, g.edge(10, 20))
	assert.False(t, g.edge(4, 3))
}

func TestGraphLength(t *testing.T) {
	g := buildGraph()
	assert.Equal(t, 4, g.length(5))
	assert.Equal(t, 4, g.length(7))
	assert.Equal(t, 4, g.length(8))
	assert.Equal(t, 6, g.length(10))
}

func TestGraphPath(t *testing.T) {
	g := buildGraph()
	assert.ElementsMatch(t, []uint32{5, 4, 3, 2, 1}, g.path(5))
	assert.ElementsMatch(t, []uint32{7, 6, 3, 2, 1}, g.path(7))
	assert.ElementsMatch(t, []uint32{10, 9, 8, 6, 3, 2, 1}, g.path(10))
}

func TestGraphIntersection(t *testing.T) {
	g := buildGraph()
	assert.Equal(t, uint32(3), g.intersection(10, 5))
	assert.Equal(t, uint32(6), g.intersection(10, 7))
	assert.Equal(t, uint32(3), g.intersection(5, 7))
	assert.Equal(t, uint32(8), g.intersection(10, 8))
}

func TestHugeRandomGraph(t *testing.T) {
	start := time.Now()
	PrintMemUsage()
	rand.Seed(time.Now().Unix())
	g := newGraph()
	previous := make([]uint32, 0)
	nodesCount := 0
	for i := 1; i < 2000000; i++ {
		if i > 1 {
			count := rand.Intn(3) + 1
			current := make([]uint32, count)
			for j := 0; j < count; j++ {
				nodesCount++
				n := uint32(nodesCount)
				current[j] = uint32(n)
				p := previous[rand.Intn(len(previous))]
				g.edge(n, p)
			}
			previous = current
		} else {
			nodesCount++
			previous = []uint32{uint32(nodesCount)}
		}
	}
	fmt.Printf("Graph with %d nodes was built in %s\n", nodesCount, time.Now().Sub(start))
	PrintMemUsage()

	start = time.Now()
	l := g.length(uint32(nodesCount))
	fmt.Printf("Length of the path from the last node (%d) found in %s\n", l, time.Now().Sub(start))
	PrintMemUsage()

	start = time.Now()
	_ = g.path(uint32(nodesCount))
	fmt.Printf("Path from the last node was found in %s\n", time.Now().Sub(start))
	PrintMemUsage()

	start = time.Now()
	x := g.intersection(uint32(nodesCount), uint32(nodesCount-1000))
	fmt.Printf("Intersection %d of paths was found in %s\n", int(x), time.Now().Sub(start))
	PrintMemUsage()
}

func buildGraph() *graph {
	g := newGraph()
	g.edge(2, 1)
	g.edge(3, 2)
	g.edge(4, 3)
	g.edge(5, 4)
	g.edge(6, 3)
	g.edge(7, 6)
	g.edge(8, 6)
	g.edge(9, 8)
	g.edge(10, 9)
	return g
}

func PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
