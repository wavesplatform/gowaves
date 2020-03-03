package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path"
	"time"

	flag "github.com/spf13/pflag"
	"github.com/wavesplatform/gowaves/pkg/client"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func main() {
	if len(os.Args) < 2 {
		showUsageAndExit()
	}

	switch os.Args[1] {
	case "transfer":
		transferTransaction()
	default:
		showUsageAndExit()
	}
}

func transferTransaction() {
	type Opts struct {
		PathToWallet string
		Recipient    string
		Amount       uint64
		Fee          uint64
		CustomSecret string
		Scheme       byte
	}

	opts := Opts{}

	f := flag.NewFlagSet("transfer", flag.ExitOnError)
	f.StringVarP(&opts.PathToWallet, "wallet", "w", "", "Path to wallet")
	f.Uint64VarP(&opts.Amount, "amount", "a", 0, "Amount of waves to send")
	f.Uint64Var(&opts.Fee, "fee", 100000, "Fee, optional")
	f.StringVarP(&opts.Recipient, "recipient", "r", "", "Address of recipient")
	f.StringVarP(&opts.CustomSecret, "secret", "s", "", "Use this secret key instead of wallet, optional")
	opts.Scheme = byte(*f.Uint8P("scheme", "", 'W', "Network byte scheme"))

	if err := f.Parse(os.Args[1:]); err != nil {
		fmt.Printf("Parse error: %q", err)
		return
	}

	pathToWallet := getWalletPath(opts.PathToWallet)
	if !exists(pathToWallet) {
		f.PrintDefaults()
		return
	}

	if opts.Recipient == "" {
		fmt.Println("Err: no recipient provided")
		f.PrintDefaults()
		return
	}

	if opts.Amount == 0 {
		fmt.Println("Err: amount should be positive")
		f.PrintDefaults()
		return
	}

	address, err := proto.NewAddressFromString(opts.Recipient)
	if err != nil {
		fmt.Printf("Err: %q", err)
		return
	}

	secretKey, err := getSecretKey(opts.CustomSecret, pathToWallet)
	if err != nil {
		fmt.Printf("Err: %q", err)
		return
	}

	publicKey := crypto.GeneratePublicKey(secretKey)

	timestamp := client.NewTimestampFromTime(time.Now())
	transfer := proto.NewUnsignedTransferWithSig(
		publicKey,
		proto.OptionalAsset{},
		proto.OptionalAsset{},
		timestamp,
		opts.Amount,
		opts.Fee,
		proto.NewRecipientFromAddress(address),
		&proto.LegacyAttachment{},
	)

	err = transfer.Sign(opts.Scheme, secretKey)
	if err != nil {
		fmt.Printf("Err: %q", err)
		os.Exit(2)
	}

	jsoned, err := json.Marshal(transfer)
	if err != nil {
		fmt.Printf("Err: %q", err)
		os.Exit(2)
	}

	fmt.Printf("\n%s\n\n", string(jsoned))
}

func getSecretKey(s string, pathToWallet string) (crypto.SecretKey, error) {
	panic("not implemented")
	//if s != "" {
	//	return crypto.NewSecretKeyFromBase58(s)
	//}
	//
	//body, err := ioutil.ReadFile(pathToWallet)
	//if err != nil {
	//	fmt.Printf("Err: %s\n", err)
	//	return crypto.SecretKey{}, err
	//}
	//
	//fmt.Print("Enter password: ")
	//pass, err := gopass.GetPasswd()
	//if err != nil {
	//	return crypto.SecretKey{}, errors.New("Interrupt")
	//}
	//wlt, err := wallet.Decode(body, pass)
	//if err != nil {
	//	return crypto.SecretKey{}, err
	//}
	//
	//secretKey, _, err := wlt.GenPair()
	//if err != nil {
	//	return crypto.SecretKey{}, err
	//}
	//
	//return secretKey, nil
}

func showUsageAndExit() {
	fmt.Println("usage: sign transfer [<args>]")
	os.Exit(0)
}

func getWalletPath(userDefinedPath string) string {
	if userDefinedPath != "" {
		return userDefinedPath
	}
	home, err := userHomeDir()
	if err != nil {
		fmt.Printf("Err: %s\n", err)
		os.Exit(0)
	}
	return path.Join(home, ".waves")
}

func userHomeDir() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return u.HomeDir, nil
}

func exists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
