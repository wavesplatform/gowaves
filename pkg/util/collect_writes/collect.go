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

type CollectInt64 struct {
	err error
	n   int64
}

func (a *CollectInt64) W(n int64, err error) {
	if a.err == nil {
		a.n += n
		a.err = err
	}
}

func (a *CollectInt64) Ret() (int64, error) {
	return a.n, a.err
}
