package collect_writes

type CollectInt struct {
	err error
	n   int
}

func (a *CollectInt) W(n int, err error) {
	if a.err == nil {
		a.n += n
		a.err = err
	}
}

func (a *CollectInt) Ret() (int, error) {
	return a.n, a.err
}
