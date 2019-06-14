package internal

import "sort"

type graph struct {
	adjacencies map[uint32]uint32
}

func newGraph() *graph {
	g := &graph{
		adjacencies: make(map[uint32]uint32),
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
	return g.pathsIntersection(pa, pb)
}

func (g *graph) pathsIntersection(pa, pb []uint32) uint32 {
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

func (g *graph) paths(vertices []uint32) []path {
	paths := pathsByLengthAscending(make([]path, len(vertices)))
	for i, v := range vertices {
		p := g.path(v)
		paths[i] = path{
			top:      v,
			vertices: p,
			length:   len(p),
		}
	}
	sort.Sort(sort.Reverse(paths))
	return paths
}

func (g *graph) forks(vertices []uint32) []fork {
	paths := g.paths(vertices)
	var longest path
	for i, p := range paths {
		if i == 0 {
			longest = p
			continue
		}
		p.intersection(longest)
	}
	return nil
}

type path struct {
	top      uint32
	length   int
	vertices []uint32
}

func (p *path) intersection(other path) uint32 {
	l := p.length
	if other.length < l {
		l = other.length
	}
	for i := 0; i < l; i++ {
		if p.vertices[i] != other.vertices[i] {
			return p.vertices[i-1]
		}
	}
	return p.vertices[l-1]
}

type pathsByLengthAscending []path

func (a pathsByLengthAscending) Len() int {
	return len(a)
}

func (a pathsByLengthAscending) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a pathsByLengthAscending) Less(i, j int) bool {
	return a[i].length < a[j].length
}

type fork struct {
	top    uint32
	common uint32
}
