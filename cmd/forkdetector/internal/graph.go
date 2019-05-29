package internal

type graph struct {
	adjacencies map[uint32]uint32
	paths       map[uint32][]uint32
}

func newGraph() *graph {
	g := &graph{
		adjacencies: make(map[uint32]uint32),
		paths:       make(map[uint32][]uint32),
	}
	g.adjacencies[1] = 0
	return g
}

func (g *graph) edge(from, to uint32) bool {
	if from == 0 || to == 0 || to >= from {
		return false
	}
	if _, ok := g.adjacencies[to]; !ok {
		g.adjacencies[to] = 0
	}
	if a, ok := g.adjacencies[from]; ok && a != 0 {
		return false
	}
	g.adjacencies[from] = to
	return true
}

func (g *graph) length(v uint32) int {
	l := 0
	for a, ok := g.adjacencies[v]; ok && a != 0; a, ok = g.adjacencies[v] {
		l++
		v = a
	}
	return l
}

func (g *graph) path(v uint32) []uint32 {
	path := []uint32{v}
	for a, ok := g.adjacencies[v]; ok && a != 0; a, ok = g.adjacencies[v] {
		path = append(path, a)
		v = a
	}
	for left, right := 0, len(path)-1; left < right; left, right = left+1, right-1 {
		path[left], path[right] = path[right], path[left]
	}
	return path
}

func (g *graph) intersection(a, b uint32) uint32 {
	pa := g.path(a)
	pb := g.path(b)
	l := len(pa)
	lb := len(pb)
	if lb < l {
		l = lb
	}
	i := 0
	for ; i < l; i++ {
		if pa[i] != pb[i] {
			break
		}
	}
	return pa[i-1]
}
