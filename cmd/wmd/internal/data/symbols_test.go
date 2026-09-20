package data

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestNewSymbolsFromFileReturnsScannerError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "symbols.txt")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", 70*1024)), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := NewSymbolsFromFile(path, proto.WavesAddress{}, 'W')
	if err == nil {
		t.Fatal("NewSymbolsFromFile() error = nil, want scanner error")
	}
}
