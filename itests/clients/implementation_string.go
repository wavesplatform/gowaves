// Code generated by "stringer -type Implementation -trimprefix Node"; DO NOT EDIT.

package clients

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[NodeGo-0]
	_ = x[NodeScala-1]
}

const _Implementation_name = "GoScala"

var _Implementation_index = [...]uint8{0, 2, 7}

func (i Implementation) String() string {
	if i >= Implementation(len(_Implementation_index)-1) {
		return "Implementation(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Implementation_name[_Implementation_index[i]:_Implementation_index[i+1]]
}
