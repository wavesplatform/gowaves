package serialization

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// base64Literal matches quoted base64 strings that are long enough to be a script.
var base64Literal = regexp.MustCompile("[\"`]([A-Za-z0-9+/]{8,}={0,2})[\"`]")

// collectScripts extracts base64 encoded scripts from the test files of this and the parent package.
// Only the strings that pass the checksum verification are returned.
func collectScripts(t *testing.T) [][]byte {
	files, err := filepath.Glob("*_test.go")
	require.NoError(t, err)
	parentFiles, err := filepath.Glob(filepath.Join("..", "*_test.go"))
	require.NoError(t, err)
	files = append(files, parentFiles...)
	seen := make(map[string]struct{})
	var r [][]byte
	for _, f := range files {
		data, rErr := os.ReadFile(f)
		require.NoError(t, rErr)
		for _, m := range base64Literal.FindAllSubmatch(data, -1) {
			src, dErr := base64.StdEncoding.DecodeString(string(m[1]))
			if dErr != nil {
				continue
			}
			if ok, _ := verifyCheckSum(src); !ok {
				continue
			}
			if _, dup := seen[string(src)]; dup {
				continue
			}
			seen[string(src)] = struct{}{}
			r = append(r, src)
		}
	}
	return r
}

func assertSameParseResult(t *testing.T, src []byte) {
	expected, expectedErr := legacyParse(src)
	actual, actualErr := Parse(src)
	if expectedErr != nil {
		assert.Error(t, actualErr)
		assert.Nil(t, actual)
		return
	}
	require.NoError(t, actualErr)
	assert.Equal(t, expected, actual)
	assert.Equal(t, legacyCheckHeader(src) == nil, CheckHeader(src) == nil)
}

func TestParseEqualsLegacyParser(t *testing.T) {
	scripts := collectScripts(t)
	require.Greater(t, len(scripts), 100) // Make sure that the corpus is not empty.
	parsed := 0
	for _, src := range scripts {
		if _, err := legacyParse(src); err == nil {
			parsed++
		}
		assertSameParseResult(t, src)
		// Every truncation of a valid script must fail in the same way.
		for n := 1; n < len(src)-4 && n < 64; n++ {
			assertSameParseResult(t, withCheckSum(t, src[:n]))
		}
	}
	require.Greater(t, parsed, 100)
}

func TestNativeFunctionName(t *testing.T) {
	for _, id := range []int16{0, 1, 100, 1000, nativeFunctionNamesCount - 1, nativeFunctionNamesCount, -1, -32768,
		32767} {
		assert.Equal(t, legacyNativeFunctionName(id), nativeFunctionName(id))
	}
}

func FuzzParseEqualsLegacyParser(f *testing.F) {
	for _, s := range fuzzRideSeeds() {
		src, err := base64.StdEncoding.DecodeString(s)
		require.NoError(f, err)
		f.Add(src[:len(src)-4])
	}
	f.Fuzz(func(t *testing.T, body []byte) {
		if len(body) == 0 {
			return
		}
		assertSameParseResult(t, withCheckSum(t, body))
	})
}
