package proto

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMicroBlock_UnmarshalBinary(t *testing.T) {
	mBytes := []byte{
		0, 0, 0, 237,
		18, 52, 86, 120,
		26,
		0, 0, 0, 224,
		19, 34, 78, 86,
		182, 35, 49, 86, 42, 200, 213, 253, 3, 193, 156, 84, 158, 6, 85, 144, 196, 169, 150, 129, 143, 56, 215, 43, 166, 227, 252, 84, 68, 23, 232, 0, 204, 6, 204, 215, 132, 126, 216, 110, 30, 18, 77, 21, 50, 29, 218, 224, 26, 112, 42, 200, 210, 90, 76, 27, 22, 155, 42, 143, 86, 142, 151, 16, 37, 32, 184, 175, 222, 165, 144, 9, 155, 185, 187, 216, 180, 34, 144, 144, 111, 212, 131, 161, 41, 94, 101, 13, 115, 237, 150, 158, 72, 214, 240, 129, 223, 110, 144, 45, 100, 108, 63, 104, 38, 27, 197, 164, 45, 44, 148, 67, 241, 109, 198, 136, 236, 169, 237, 98, 21, 252, 71, 28, 1, 205, 195, 21, 112, 95, 143, 186, 183, 4, 37, 187, 226, 91, 25, 138, 66, 82, 167, 199, 249, 201, 152, 1, 95, 152, 236, 164, 21, 150, 132, 33, 102, 24, 125, 143, 1, 201, 43, 50, 177, 73, 186, 246, 142, 20, 209, 126, 243, 169, 70, 151, 240, 93, 225, 54, 142, 137, 133, 247, 5, 134, 30, 185, 199, 189, 3, 73, 246, 23, 231, 94, 130, 111, 216, 191, 163, 15, 59, 202, 40, 55, 37}

	m := MicroBlock{}
	require.NoError(t, m.UnmarshalBinary(mBytes[17:]))

}
