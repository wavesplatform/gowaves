package bn254

func square(c, a *fe) {
	mul(c, a, a)
}

func neg(c, a *fe) {
	if a.isZero() {
		c.set(a)
	} else {
		_neg(c, a)
	}
}

//go:noescape
func add(c, a, b *fe)

//go:noescape
func addAssign(a, b *fe)

//go:noescape
func laddAssign(a, b *fe)

//go:noescape
func sub(c, a, b *fe)

//go:noescape
func subAssign(a, b *fe)

//go:noescape
func lsubAssign(a, b *fe) uint64

//go:noescape
func _neg(c, a *fe)

//go:noescape
func double(c, a *fe)

//go:noescape
func doubleAssign(a *fe)

//go:noescape
func mul(c, a, b *fe)
