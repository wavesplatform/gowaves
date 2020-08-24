package fride

type symbol interface {
	address() int
}

type functionSymbol struct {
	name      string
	addr      int
	arguments []string
}

func (s *functionSymbol) address() int {
	return s.addr
}

type declarationSymbol struct {
	name string
	addr int
}

func (s *declarationSymbol) address() int {
	return s.addr
}

type scriptMeta struct {
	blocks  map[int]int // Number of blocks declared at position (key)
	symbols map[int]symbol
}

func newScriptMeta() *scriptMeta {
	return &scriptMeta{
		blocks:  make(map[int]int),
		symbols: make(map[int]symbol),
	}
}

func (m *scriptMeta) addBlock(pos int) {
	m.blocks[pos]++
}
