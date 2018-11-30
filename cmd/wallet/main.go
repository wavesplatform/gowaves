package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path"

	"github.com/howeyc/gopass"
	flag "github.com/spf13/pflag"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/wallet"
)

var usage = `

Usage:
  wallet command [flags]

Available Commands:
  create       Create wallet
  show         Print wallet data

`

type Opts struct {
	Force        bool
	PathToWallet string
}

func main() {
	opts := Opts{}

	flag.BoolVarP(&opts.Force, "force", "f", false, "Overwrite existing wallet")
	flag.StringVarP(&opts.PathToWallet, "wallet", "w", "", "Path to wallet")

	flag.Parse()

	command := flag.Arg(0)

	switch command {
	case "create":
		createWallet(opts)
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

	priv, pub, err := wlt.GenPair()
	if err != nil {
		fmt.Printf("Err: %s\n", err.Error())
		return
	}

	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, pub)
	if err != nil {
		fmt.Printf("Err: %s\n", err.Error())
		return
	}

	fmt.Printf("private: %s\n", priv.String())
	fmt.Printf("public: %s\n", pub.String())
	fmt.Printf("addr: %s\n", addr.String())
}

func showUsageAndExit() {
	fmt.Print(usage)
	flag.PrintDefaults()
	os.Exit(0)
}

func createWallet(opts Opts) {
	walletPath := getWalletPath(opts.PathToWallet)
	if exists(walletPath) {
		if !opts.Force {
			fmt.Println("Err: Wallet exists, use --force to overwrite")
			return
		}
	}

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

	fmt.Print("Enter seed: ")
	seed, err := gopass.GetPasswd()
	if err != nil {
		fmt.Println("Interrupt")
		return
	}

	wlt, err := wallet.NewWalletFromSeed(seed)
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
