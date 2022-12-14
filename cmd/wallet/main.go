package main

import (
	"encoding/binary"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/user"
	"path/filepath"

	"github.com/mr-tron/base58"
	"github.com/pkg/errors"

	"github.com/howeyc/gopass"
	flag "github.com/spf13/pflag"
	"github.com/tyler-smith/go-bip39"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/wallet"
)

type WalletGenerationChoice string

const (
	newSeedPhrase WalletGenerationChoice = WalletGenerationChoice(rune(iota + 1))
	existingSeedPhrase
	existingBase58SeedPhrase
	existingAccountSeed
	existingBase58AccountSeed
)

const (
	defaultBitSize = 160
)

var usage = `

Usage:
  wallet <command> [flags]

Available Commands:
  generate     Generate a wallet
  show         Print the wallet data

Flags:
	

`

type Opts struct {
	Force         bool
	PathToWallet  string
	AccountNumber int
	Scheme        proto.Scheme

	SeedPhrase         *string
	IsSeedPhraseBase58 bool

	Base58AccountSeed *string
}

func main() {
	opts := Opts{}
	var scheme string
	flag.BoolVarP(&opts.Force, "force", "f", false, "Overwrite existing wallet")
	flag.StringVarP(&opts.PathToWallet, "wallet", "w", "", "Path to wallet")
	flag.IntVarP(&opts.AccountNumber, "number", "i", 0, "Input account number. 0 is default")
	flag.StringVarP(&scheme, "scheme", "sch", "mainnet", "Input the network scheme: mainnet, testnet, stagenet")
	flag.StringVarP(opts.SeedPhrase, "seedPhrase", "sp", "", "Input your seed phrase")
	flag.BoolVarP(&opts.IsSeedPhraseBase58, "seedBase58", "seedB58", false, "Seed phrase is written in Base58 format")

	flag.StringVarP(opts.Base58AccountSeed, "accountSeed", "as", "", "Input your account seed in Base58 format")

	flag.Parse()

	command := flag.Arg(0)

	schemeByte, err := proto.ParseSchemeFromStr(scheme)
	if err != nil {
		fmt.Printf("failed to parse network scheme: %v", err)
		return
	}
	opts.Scheme = schemeByte

	switch command {
	case "add":
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

	fmt.Print("Enter password which will be used to encode your seed: ")
	pass, err := gopass.GetPasswd()
	if err != nil {
		fmt.Println("Interrupt")
		return
	}

	if len(pass) == 0 {
		fmt.Println("Err: password required")
		return
	}

	b, err := os.ReadFile(walletPath) // #nosec: in this case check for prevent G304 (CWE-22) is not necessary
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

func generateMnemonic() (string, error) {
	entropy, err := bip39.NewEntropy(defaultBitSize)
	if err != nil {
		return "", errors.Wrap(err, "failed to generate random entropy")
	}
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", errors.Wrap(err, "failed to generate mnemonic phrase")
	}
	return mnemonic, nil
}

//func generate(n int, scheme byte) (crypto.Digest, crypto.PublicKey, crypto.SecretKey, proto.Address, error) {
//	seedPhrase, err := generateMnemonic()
//	if err != nil {
//		return crypto.Digest{}, crypto.PublicKey{}, crypto.SecretKey{}, nil, err
//	}
//
//	return generateOnSeedPhrase(seedPhrase, n, scheme)
//}

func generateOnSeedPhrase(seedPhrase string, n int, scheme byte) (crypto.Digest, crypto.PublicKey, crypto.SecretKey, proto.Address, error) {
	iv := make([]byte, 4)
	binary.BigEndian.PutUint32(iv, uint32(n))
	s := append(iv, seedPhrase...)
	accountSeed, err := crypto.SecureHash(s)
	if err != nil {
		return crypto.Digest{}, crypto.PublicKey{}, crypto.SecretKey{}, nil, errors.Wrap(err, "failed to generate account seed")
	}
	pk, sk, a, err := generateOnAccountSeed(accountSeed, scheme)
	if err != nil {
		return crypto.Digest{}, crypto.PublicKey{}, crypto.SecretKey{}, nil, err
	}
	return accountSeed, pk, sk, a, nil
}

func generateOnAccountSeed(accountSeed crypto.Digest, scheme proto.Scheme) (crypto.PublicKey, crypto.SecretKey, proto.Address, error) {
	sk, pk, err := crypto.GenerateKeyPair(accountSeed.Bytes())
	if err != nil {
		return crypto.PublicKey{}, crypto.SecretKey{}, nil, errors.Wrap(err, "failed to generate key pair")
	}
	a, err := proto.NewAddressFromPublicKey(scheme, pk)
	if err != nil {
		return crypto.PublicKey{}, crypto.SecretKey{}, nil, errors.Wrap(err, "failed to generate address")
	}
	return pk, sk, a, nil
}

type WalletCredentials struct {
	seedPhrase  string
	accountSeed crypto.Digest
	pk          crypto.PublicKey
	sk          crypto.SecretKey
	address     proto.Address
}

var wrongProgramArguments = errors.New("wrong program arguments were provided")

func createWallet(opts Opts) error {
	walletPath := getWalletPath(opts.PathToWallet)
	fmt.Print(`Available options: 
		 1: Generate new seed phrase and wallet
		 2: Create a wallet based on an existing seed phrase. Requires seed phrase argument to be provided
		 3: Create a wallet based on an existing base58-encoded seed phrase. Requires seed phrase argument to be provided and seed-base58 flag marked "true"
		 4: Create a wallet based on an existing base58-encoded account seed. Requires account seed argument in Base58 format`)

	var walletCredentials WalletCredentials
	var choice WalletGenerationChoice
	switch choice {
	case newSeedPhrase:
		seedPhrase, err := generateMnemonic()
		if err != nil {
			return errors.Wrap(err, "failed to generate seed phrase")
		}
		accountSeed, pk, sk, address, err := generateOnSeedPhrase(seedPhrase, opts.AccountNumber, opts.Scheme)
		if err != nil {
			return err
		}
		walletCredentials = WalletCredentials{
			seedPhrase:  seedPhrase,
			accountSeed: accountSeed,
			pk:          pk,
			sk:          sk,
			address:     address,
		}

	case existingSeedPhrase:
		if opts.SeedPhrase == nil {
			return errors.Wrap(wrongProgramArguments, "no seed phrase was provided")
		}
		if opts.IsSeedPhraseBase58 != false {
			return errors.Wrap(wrongProgramArguments, "seed phrase base-58-encoding flag was true, but non-base-58 option was chosen")
		}

		accountSeed, pk, sk, address, err := generateOnSeedPhrase(*opts.SeedPhrase, opts.AccountNumber, opts.Scheme)
		if err != nil {
			return err
		}
		walletCredentials = WalletCredentials{
			seedPhrase:  *opts.SeedPhrase,
			accountSeed: accountSeed,
			pk:          pk,
			sk:          sk,
			address:     address,
		}
	case existingBase58SeedPhrase:
		if opts.IsSeedPhraseBase58 != true {
			return errors.Wrap(wrongProgramArguments, "seed phrase base-58-encoding flag was false, but base-58 option was chosen")
		}
		b, err := base58.Decode(*opts.SeedPhrase)
		if err != nil {
			return errors.Wrap(err, "failed to decode base58-encoded seed phrase")
		}
		decodedSeedPhrase := string(b)

		accountSeed, pk, sk, address, err := generateOnSeedPhrase(decodedSeedPhrase, opts.AccountNumber, opts.Scheme)
		if err != nil {
			return err
		}
		walletCredentials = WalletCredentials{
			seedPhrase:  decodedSeedPhrase,
			accountSeed: accountSeed,
			pk:          pk,
			sk:          sk,
			address:     address,
		}

	case existingBase58AccountSeed:
		if opts.Base58AccountSeed == nil {
			return errors.Wrap(wrongProgramArguments, "no base58 account seed was provided")
		}
		accountSeed, err := crypto.NewDigestFromBase58(*opts.Base58AccountSeed)
		if err != nil {
			return errors.Wrap(err, "failed to decode base58-encoded account seed")
		}
		pk, sk, address, err := generateOnAccountSeed(accountSeed, opts.Scheme)

	default:
		showUsageAndExit()
	}

	fmt.Print("Enter password that will be used to encode your account seed: ")
	pass, err := gopass.GetPasswd()
	if err != nil {
		fmt.Println("Interrupt")
		return err
	}

	if len(pass) == 0 {
		fmt.Println("Err: Password required")
		return err
	}

	var wlt wallet.Wallet
	if exists(walletPath) {
		b, err := os.ReadFile(walletPath) // #nosec: in this case check for prevent G304 (CWE-22) is not necessary
		if err != nil {
			fmt.Printf("Err: %s\n", err.Error())
			return err
		}
		wlt, err = wallet.Decode(b, pass)
		if err != nil {
			fmt.Printf("Err: %s\n", err.Error())
			return err
		}
	} else {
		wlt = wallet.NewWallet()
	}

	//fmt.Print("Enter seed: ")
	//seed, err := gopass.GetPasswd()
	//if err != nil {
	//	fmt.Println("Interrupt")
	//	return err
	//}

	//if opts.Base58 {
	//	seed, err = base58.Decode(string(seed))
	//	if err != nil {
	//		fmt.Printf("Err: %s\n", err.Error())
	//		return err
	//	}
	//}
	seedPhraseBytes := []byte(walletCredentials.seedPhrase)
	err = wlt.AddSeed(seedPhraseBytes)
	if err != nil {
		fmt.Printf("Err: %s\n", err.Error())
		return err
	}

	bts, err := wlt.Encode(pass)
	if err != nil {
		fmt.Printf("Err: %s\n", err.Error())
		return err
	}

	err = os.WriteFile(walletPath, bts, 0600)
	if err != nil {
		fmt.Printf("Err: %s\n", err.Error())
		return err
	}

	log.Printf("Seed Phrase: '%s'\n", walletCredentials.seedPhrase)
	log.Printf("Account Number: %d\n", opts.AccountNumber)
	log.Printf("Account Seed: %s\n", walletCredentials.accountSeed.String())
	log.Printf("Public Key: %s\n", walletCredentials.pk.String())
	log.Printf("Secret Key: %s\n", walletCredentials.sk.String())
	log.Printf("Address: %s\n", walletCredentials.address.String())

	fmt.Println("Your wallet has been successfully created")
	return nil
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
		return filepath.Clean(userDefinedPath)
	}
	home, err := userHomeDir()
	if err != nil {
		fmt.Printf("Err: %s\n", err.Error())
		os.Exit(0)
	}
	return filepath.Join(home, ".waves")
}

func exists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false
		}
	}
	return true
}
