package internal

import (
	"fmt"
	"math/rand"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestGraphPaths(t *testing.T) {
	a := buildGraph()
	paths := a.paths([]uint32{5, 7, 10})
	assert.Equal(t, 3, len(paths))
	assert.Equal(t, 10, int(paths[0].top))
	assert.Equal(t, 7, paths[0].length)
	assert.Equal(t, 5, int(paths[1].top))
	assert.Equal(t, 5, paths[1].length)
	assert.Equal(t, 7, int(paths[2].top))
	assert.Equal(t, 5, paths[2].length)
}

func TestGraphPathsIntersection(t *testing.T) {
	a := buildGraph()
	paths := a.paths([]uint32{5, 7, 9, 10})
	assert.Equal(t, 4, len(paths))
	top, lag := paths[0].intersection(paths[1])
	assert.Equal(t, 9, int(top))
	assert.Equal(t, 6, lag)
	top, lag = paths[0].intersection(paths[2])
	assert.Equal(t, 3, int(top))
	assert.Equal(t, 3, lag)
	top, lag = paths[0].intersection(paths[3])
	assert.Equal(t, 6, int(top))
	assert.Equal(t, 4, lag)
}

func TestGraphForks(t *testing.T) {
	a := buildGraph()
	forks := a.forks([]uint32{4, 5, 6, 7, 9, 10})
	assert.Equal(t, 3, len(forks))

	assert.Equal(t, 10, int(forks[0].top))
	assert.Equal(t, 10, int(forks[0].common))
	assert.Equal(t, 2, len(forks[0].lags))

	assert.Equal(t, 5, int(forks[1].top))
	assert.Equal(t, 3, int(forks[1].common))
	assert.Equal(t, 2, len(forks[1].lags))

	assert.Equal(t, 7, int(forks[2].top))
	assert.Equal(t, 6, int(forks[2].common))
	assert.Equal(t, 2, len(forks[2].lags))
}

func BenchmarkPathsSort1M(b *testing.B) {
	g := buildRandomGraph(2000000, 3)
	vertices := make([]uint32, 300)
	for i := 0; i < 300; i++ {
		vertices[i] = uint32(rand.Intn(1000000) + 1)
	}
	for n := 0; n < b.N; n++ {
		g.paths(vertices)
	}
}

func BenchmarkGraphPath2M(b *testing.B) {
	g := buildRandomGraph(2000000, 5)
	var p []uint32
	for n := 0; n < b.N; n++ {
		p = g.path(2000000)
	}
	fmt.Println(len(p))
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
	fmt.Printf("Graph with %d nodes was built in %s\n", nodesCount, time.Since(start))
	PrintMemUsage()

	start = time.Now()
	l := g.length(uint32(nodesCount))
	fmt.Printf("Length of the path from the last node (%d) found in %s\n", l, time.Since(start))
	PrintMemUsage()

	start = time.Now()
	_ = g.path(uint32(nodesCount))
	fmt.Printf("Path from the last node was found in %s\n", time.Since(start))
	PrintMemUsage()

	start = time.Now()
	x := g.intersection(uint32(nodesCount), uint32(nodesCount-1000))
	fmt.Printf("Intersection %d of paths was found in %s\n", int(x), time.Since(start))
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

func buildRandomGraph(size, width int) *graph {
	rand.Seed(time.Now().Unix())
	g := newGraph()
	previous := make([]uint32, 0)
	nodesCount := 0
	for i := 1; i < size; i++ {
		if i > 1 {
			count := rand.Intn(width) + 1
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
