package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wavesplatform/gowaves/pkg/ride/compiler"
)

var usage = `
Usage:
  compiler -f <script path> [options]

Options:
	-c, --compaction	Compaction mode [not working yet]
`

func main() {
	var (
		scriptPath   string
		isCompaction bool
	)
	flag.StringVar(&scriptPath, "f", "", "Path to script file")
	flag.BoolVar(&isCompaction, "compaction", false, "Compaction mode [not working yet]") // TODO: add compaction mode
	flag.BoolVar(&isCompaction, "c", false, "Compaction mode [not working yet]")

	flag.Usage = func() {
		fmt.Println(usage)
	}
	flag.Parse()

	if scriptPath == "" {
		fmt.Printf("Script path is not specified")
		flag.Usage()
		os.Exit(0)
	}

	b, err := os.ReadFile(filepath.Clean(scriptPath))
	if err != nil {
		fmt.Printf("Failed to open file: %s", err)
		os.Exit(0)
	}

	base64res, errors := compiler.Compile(string(b))
	if errors != nil {
		fmt.Printf("Failed to compile script\n")
		for _, err := range errors {
			fmt.Printf("\t%s", err)
		}
		os.Exit(0)
	}
	fmt.Println(base64res)
}
