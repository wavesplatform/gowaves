package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path"

	"github.com/howeyc/gopass"
	"github.com/mr-tron/base58"
	flag "github.com/spf13/pflag"
	"github.com/wavesplatform/gowaves/pkg/wallet"
)

var usage = `

Usage:
  wallet command [flags]

Available Commands:
  add          Add seed to wallet
  show         Print wallet data

`

type Opts struct {
	Force        bool
	PathToWallet string
	Base58       bool
}

func main() {
	opts := Opts{}

	flag.BoolVarP(&opts.Force, "force", "f", false, "Overwrite existing wallet")
	flag.StringVarP(&opts.PathToWallet, "wallet", "w", "", "Path to wallet")
	flag.BoolVarP(&opts.Base58, "base58", "b", false, "Input seed as Base58 encoded string")

	flag.Parse()

	command := flag.Arg(0)

	switch command {
	case "add":
		addToWallet(opts)
	case "show":
		show(opts)
	default:
		showUsageAndExit()
	}
}

func show(opts Opts) {
	walletPath := getWalletPath(opts.PathToWallet)
	if !exists(walletPath) {
		fmt.Println("Err: wallet not found")
		return
	}

	fmt.Print("Enter password: ")
	pass, err := gopass.GetPasswd()
	if err != nil {
		fmt.Println("Interrupt")
		return
	}

	if len(pass) == 0 {
		fmt.Println("Err: password required")
		return
	}

	b, err := ioutil.ReadFile(walletPath)
	if err != nil {
		fmt.Printf("Err: %s\n", err.Error())
		return
	}

	wlt, err := wallet.Decode(b, pass)
	if err != nil {
		fmt.Printf("Err: %s\n", err.Error())
		return
	}

	for _, s := range wlt.Seeds() {
		fmt.Printf("seed: %s\n", string(s))
	}
}

func showUsageAndExit() {
	fmt.Print(usage)
	flag.PrintDefaults()
	os.Exit(0)
}

func addToWallet(opts Opts) {
	walletPath := getWalletPath(opts.PathToWallet)

	fmt.Print("Enter password: ")
	pass, err := gopass.GetPasswd()
	if err != nil {
		fmt.Println("Interrupt")
		return
	}

	if len(pass) == 0 {
		fmt.Println("Err: Password required")
		return
	}

	var wlt wallet.Wallet
	if exists(walletPath) {
		b, err := ioutil.ReadFile(walletPath)
		if err != nil {
			fmt.Printf("Err: %s\n", err.Error())
			return
		}
		wlt, err = wallet.Decode(b, pass)
		if err != nil {
			fmt.Printf("Err: %s\n", err.Error())
			return
		}
	} else {
		wlt = wallet.NewWallet()
	}

	fmt.Print("Enter seed: ")
	seed, err := gopass.GetPasswd()
	if err != nil {
		fmt.Println("Interrupt")
		return
	}

	if opts.Base58 {
		seed, err = base58.Decode(string(seed))
		if err != nil {
			fmt.Printf("Err: %s\n", err.Error())
			return
		}
	}

	err = wlt.AddSeed(seed)
	if err != nil {
		fmt.Printf("Err: %s\n", err.Error())
		return
	}

	bts, err := wlt.Encode(pass)
	if err != nil {
		fmt.Printf("Err: %s\n", err.Error())
		return
	}

	err = ioutil.WriteFile(walletPath, bts, 0600)
	if err != nil {
		fmt.Printf("Err: %s\n", err.Error())
		return
	}
	fmt.Println("Created!")
}

func userHomeDir() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return u.HomeDir, nil
}

func getWalletPath(userDefinedPath string) string {
	if userDefinedPath != "" {
		return userDefinedPath
	}
	home, err := userHomeDir()
	if err != nil {
		fmt.Printf("Err: %s\n", err.Error())
		os.Exit(0)
	}
	return path.Join(home, ".waves")
}

func exists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
