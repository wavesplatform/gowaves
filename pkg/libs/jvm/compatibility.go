package jvm

func ModDivision(x int64, y int64) int64 {
	return x - FloorDiv(x, y)*y
}

func FloorDiv(x int64, y int64) int64 {
	r := x / y
	// if the signs are different and modulo not zero, round down
	if (x^y) < 0 && (r*y != x) {
		r--
	}
	return r
}
