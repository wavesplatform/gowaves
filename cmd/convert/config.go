package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type config struct {
	scheme   proto.Scheme
	sk       *crypto.SecretKey
	in       io.ReadCloser
	out      io.WriteCloser
	toJSON   bool
	toBinary bool
	base64   bool
	validate bool
}

func (c *config) parse() error {
	var (
		scheme, privateKey, in, out string
	)
	flag.StringVar(&scheme, "scheme", "W", "Specifies the network scheme byte. Defaults to 'W' (MainNet).")
	flag.BoolVar(&c.toJSON, "to-json", false,
		"Converts the transaction to JSON format. Signs the transaction if a private key is provided.")
	flag.BoolVar(&c.toBinary, "to-binary", false,
		"Converts the transaction to binary format. Signs the transaction if a private key is provided.")
	flag.BoolVar(&c.base64, "base64", false, "Encodes the binary transaction in Base64.")
	flag.StringVar(&privateKey, "private-key", "",
		"Private key for signing the transaction. Provide the key as a Base58 string.")
	flag.StringVar(&in, "in", "",
		"Specifies the input file path. Defaults to an empty string. If empty, reads from STDIN.")
	flag.StringVar(&out, "out", "",
		"Specifies the output file path. Defaults to an empty string. If empty, writes to STDOUT.")
	flag.BoolVar(&c.validate, "validate", false, "Validates the transaction after deserialization.")
	flag.Parse()

	if len(scheme) != 1 {
		return fmt.Errorf("invalid network scheme %q", scheme)
	}
	c.scheme = []byte(scheme)[0]

	if len(privateKey) != 0 {
		sk, err := crypto.NewSecretKeyFromBase58(privateKey)
		if err != nil {
			return fmt.Errorf("failed to parse private key: %w", err)
		}
		c.sk = &sk
	}
	if inErr := c.setInput(in); inErr != nil {
		return inErr
	}
	if outErr := c.setOutput(out); outErr != nil {
		return outErr
	}
	return nil
}

func (c *config) setInput(str string) error {
	if len(str) == 0 {
		c.in = os.Stdin
		return nil
	}
	fi, err := os.Stat(str)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("file %q does not exist", str)
	}
	if err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}
	if fi.IsDir() {
		return fmt.Errorf("path %q is not a file", str)
	}
	c.in, err = os.Open(path.Clean(str))
	if err != nil {
		return fmt.Errorf("failed to open input file %q: %w", str, err)
	}
	return nil
}

func (c *config) createOutputFile(fn string) error {
	f, err := os.Create(path.Clean(fn))
	if err != nil {
		return fmt.Errorf("failed to open output file %q: %w", fn, err)
	}
	c.out = f
	return nil
}

func (c *config) setOutput(str string) error {
	if len(str) == 0 {
		c.out = os.Stdout
		return nil
	}
	fi, err := os.Stat(str)
	if errors.Is(err, os.ErrNotExist) {
		return c.createOutputFile(str)
	}
	if err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}
	if fi.IsDir() {
		return fmt.Errorf("path %q is not a file", str)
	}
	return c.createOutputFile(str)
}

func (c *config) close() {
	if c.in != nil {
		if err := c.in.Close(); err != nil {
			log.Printf("Failed to close input: %v", err)
		}
	}
	if c.out != nil {
		if err := c.out.Close(); err != nil {
			log.Printf("Failed to close output: %v", err)
		}
	}
}
