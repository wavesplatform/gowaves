package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/mr-tron/base58"

	"github.com/pkg/errors"

	"github.com/howeyc/gopass"
	"github.com/tyler-smith/go-bip39"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/wallet"
)

const (
	newOpt               = "new"
	showOpt              = "show"
	seedPhraseOpt        = "seed-phrase"
	seedPhraseBase58Opt  = "seed-phrase-base58"
	accountSeedBase58Opt = "account-seed-base58"

	schemeOpt = "scheme"
)

var primaryFlags = []string{newOpt, showOpt, seedPhraseOpt, seedPhraseBase58Opt, accountSeedBase58Opt}

const (
	defaultBitSize    = 160
	walletDefaultName = ".waves"
)

var usage = `
Usage:
  wallet [flags]

Flags:
`

var examples = `
Examples:
	./wallet -show					Show existing wallet credentials
	./wallet -new					Generate a seed phrase and a wallet 
	./wallet -seed-phrase "..."			Import a seed phrase
	./wallet -seed-phrase-base58 "..."		Import a Base58 encoded seed phrase
	./wallet -account-seed-base58 "..."		Import a Base58 encoded account seed
`

func schemeFromString(s string) (proto.Scheme, error) {
	val := []byte(s)
	if len(val) != 1 {
		return byte(0), errors.New("failed to parse scheme: one letter should be provided")
	}

	return val[0], nil
}

type Opts struct {
	seedPhrase        string
	base58SeedPhrase  string
	base58AccountSeed string
}

func main() {
	var (
		show          bool
		newWallet     bool
		walletPath    string
		accountNumber int
		sch           string
		opts          Opts
	)
	flag.BoolVar(&newWallet, newOpt, false, "Generate and add a new seed phrase (Primary flag)")
	flag.BoolVar(&show, showOpt, false, "Show existing wallet credentials (Primary flag)")
	flag.StringVar(&opts.seedPhrase, seedPhraseOpt, "", "Import a seed phrase (Primary flag)")
	flag.StringVar(&opts.base58SeedPhrase, seedPhraseBase58Opt, "", "Import a base58-encoded seed phrase (Primary flag)")
	flag.StringVar(&opts.base58AccountSeed, accountSeedBase58Opt, "", "Import a base58-encoded account seed (Primary flag)")
	flag.StringVar(&walletPath, "wallet", "", "Path to the wallet file")
	flag.IntVar(&accountNumber, "number", 0, "Account number. 0 is default")
	flag.StringVar(&sch, schemeOpt, "W", "Network scheme: MainNet=W, TestNet=T, StageNet=S, CustomNet=E. MainNet is default")

	flag.Parse()

	if moreThanOnePrimaryFlagPassed() {
		fmt.Print("Failed to handle more than one primary flag")
		showUsageAndExit()
	}

	var scheme proto.Scheme
	var err error
	if isFlagPassed(schemeOpt) {
		scheme, err = schemeFromString(sch)
		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}
	} else {
		scheme = proto.MainNetScheme
	}

	command, err := passedCommand()
	if err != nil {
		fmt.Println(err)
		showUsageAndExit()
	}

	switch command {
	case showOpt:
		err = showWallet(walletPath, scheme)
		if err != nil {
			log.Printf("Failed to show wallet's credentials: %v", err)
		}
	case newOpt, seedPhraseOpt, seedPhraseBase58Opt, accountSeedBase58Opt:
		err = createWallet(command, walletPath, accountNumber, scheme, opts)
		if err != nil {
			log.Printf("Failed to create a new wallet: %v", err)
		}
	default:
		showUsageAndExit()
	}
}

func ReadWallet(walletPath string) (wallet.Wallet, []byte, error) {
	fmt.Print("Enter password to decode your wallet: ")
	pass, err := gopass.GetPasswd()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get the input password")
	}
	if len(pass) == 0 {
		return nil, nil, errors.New("empty password")
	}

	b, err := os.ReadFile(walletPath) // #nosec: in this case check for prevent G304 (CWE-22) is not necessary
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to read the wallet")
	}
	wlt, err := wallet.Decode(b, pass)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to decode the wallet")

	}

	return wlt, pass, nil
}

func showWallet(walletPath string, scheme proto.Scheme) error {
	walletPath, err := getWalletPath(walletPath)
	if err != nil {
		return errors.Wrap(err, "failed to handle wallet's path")
	}
	if !exists(walletPath) {
		return errors.New("wallet does not exist")
	}

	wlt, _, err := ReadWallet(walletPath)
	if err != nil {
		return errors.Errorf("failed to read the wallet, %v", err)
	}

	for i, s := range wlt.AccountSeeds() {
		accountSeedDigest, err := crypto.NewDigestFromBytes(s)
		if err != nil {
			return errors.Wrap(err, "failed to receive digest from account seed bytes")
		}
		pk, sk, address, err := generateOnAccountSeed(accountSeedDigest, scheme)
		if err != nil {
			return errors.Wrap(err, "failed to receive wallet's credentials")
		}
		fmt.Println()
		fmt.Printf("Account number: %d\n", i)
		fmt.Printf("Account seed:   %s\n", accountSeedDigest.String())
		fmt.Printf("Public Key:     %s\n", pk.String())
		fmt.Printf("Secret Key:     %s\n", sk.String())
		fmt.Printf("Address:        %s\n", address.String())
	}
	return nil
}

func showUsageAndExit() {
	fmt.Print(usage)
	flag.PrintDefaults()
	fmt.Print(examples)
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
	accountSeed crypto.Digest
	pk          crypto.PublicKey
	sk          crypto.SecretKey
	address     proto.Address
}

var wrongProgramArguments = errors.New("wrong program arguments were provided")

func generateWalletCredentials(
	choice string,
	accountNumber int,
	scheme proto.Scheme,
	opts Opts) (*WalletCredentials, error) {

	var walletCredentials *WalletCredentials

	switch choice {
	case newOpt:
		newSeedPhrase, err := generateMnemonic()
		if err != nil {
			return nil, errors.Wrap(err, "failed to generate seed phrase")
		}
		accountSeed, pk, sk, address, err := generateOnSeedPhrase(newSeedPhrase, accountNumber, scheme)
		if err != nil {
			return nil, err
		}
		walletCredentials = &WalletCredentials{
			accountSeed: accountSeed,
			pk:          pk,
			sk:          sk,
			address:     address,
		}

		fmt.Printf("Seed Phrase: '%s'\n", newSeedPhrase)
	case seedPhraseOpt:
		if opts.seedPhrase == "" {
			return nil, errors.Wrap(wrongProgramArguments, "no seed phrase was provided")
		}

		accountSeed, pk, sk, address, err := generateOnSeedPhrase(opts.seedPhrase, accountNumber, scheme)
		if err != nil {
			return nil, err
		}
		walletCredentials = &WalletCredentials{
			accountSeed: accountSeed,
			pk:          pk,
			sk:          sk,
			address:     address,
		}
	case seedPhraseBase58Opt:
		if opts.base58SeedPhrase == "" {
			return nil, errors.Wrap(wrongProgramArguments, "no base58 encoded seed phrase was provided")
		}
		b, err := base58.Decode(opts.base58SeedPhrase)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode base58-encoded seed phrase")
		}
		decodedSeedPhrase := string(b)

		accountSeed, pk, sk, address, err := generateOnSeedPhrase(decodedSeedPhrase, accountNumber, scheme)
		if err != nil {
			return nil, err
		}
		walletCredentials = &WalletCredentials{
			accountSeed: accountSeed,
			pk:          pk,
			sk:          sk,
			address:     address,
		}
	case accountSeedBase58Opt:
		if opts.base58AccountSeed == "" {
			return nil, errors.Wrap(wrongProgramArguments, "no base58 account seed was provided")
		}
		accountSeed, err := crypto.NewDigestFromBase58(opts.base58AccountSeed)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode base58-encoded account seed")
		}
		pk, sk, address, err := generateOnAccountSeed(accountSeed, scheme)
		if err != nil {
			return nil, err
		}
		walletCredentials = &WalletCredentials{
			accountSeed: accountSeed,
			pk:          pk,
			sk:          sk,
			address:     address,
		}

	default:
		showUsageAndExit()
	}

	return walletCredentials, nil
}

func createWallet(
	command string,
	walletPath string,
	accountNumber int,
	scheme proto.Scheme, opts Opts) error {
	walletPath, err := getWalletPath(walletPath)
	if err != nil {
		return errors.Wrap(err, "failed to handle wallet's path")
	}

	var walletCredentials *WalletCredentials

	walletCredentials, err = generateWalletCredentials(command, accountNumber, scheme, opts)
	if err != nil {
		return errors.Wrap(err, "failed to generate wallet's credentials")
	}
	if walletCredentials == nil {
		return errors.New("failed to generate wallet's credentials")
	}

	var oldWallet bool
	var password []byte
	var wlt wallet.Wallet
	if exists(walletPath) {
		fmt.Print("Wallet already exists. Do you want to [A]dd / [O]verwrite / [C]ancel? ")
		var a string
		_, err := fmt.Scanf("%s", &a)
		if err != nil {
			return errors.Wrap(err, "failed to get the answer on rewriting the existing wallet")
		}
		answer := strings.ToLower(a)
		switch answer {
		case "o":
			wlt = wallet.NewWallet()
		case "a":
			oldWallet = true
			wlt, password, err = ReadWallet(walletPath)
			if err != nil {
				return err
			}
		case "c":
			return nil
		default:
			return nil
		}
	} else {
		wlt = wallet.NewWallet()
	}

	err = wlt.AddAccountSeed(walletCredentials.accountSeed.Bytes())
	if err != nil {
		return errors.Wrap(err, "failed to add the account seed to the wallet")
	}

	if !oldWallet {
		fmt.Print("Enter password to encode your account seed: ")
		password, err = gopass.GetPasswd()
		if err != nil {
			return errors.Wrap(err, "failed to get the password to encode the account seed")
		}

		if len(password) == 0 {
			return errors.Wrap(err, "the password's length is zero")
		}
	}

	bts, err := wlt.Encode(password)
	if err != nil {
		return errors.Wrap(err, "failed to encode the wallet with the provided password")

	}

	err = os.WriteFile(walletPath, bts, 0600)
	if err != nil {
		return errors.Wrap(err, "failed to write the wallet's data to the wallet")

	}
	fmt.Printf("New account has been to wallet successfully %s\n", walletPath)
	fmt.Printf("Account Seed:   %s\n", walletCredentials.accountSeed.String())
	fmt.Printf("Public Key:     %s\n", walletCredentials.pk.String())
	fmt.Printf("Secret Key:     %s\n", walletCredentials.sk.String())
	fmt.Printf("Address:        %s\n", walletCredentials.address.String())
	return nil
}

func userHomeDir() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return u.HomeDir, nil
}

func getWalletPath(userDefinedPath string) (string, error) {
	if userDefinedPath != "" {
		return filepath.Clean(userDefinedPath), nil
	}
	home, err := userHomeDir()
	if err != nil {
		return "", errors.Wrap(err, "failed to get user's home directory")
	}
	return filepath.Join(home, walletDefaultName), nil
}

func exists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false
		}
	}
	return true
}

func isFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func moreThanOnePrimaryFlagPassed() bool {
	passedFlagsNum := 0
	for _, f := range primaryFlags {
		if isFlagPassed(f) {
			passedFlagsNum++
		}
	}

	return passedFlagsNum > 1
}

func passedCommand() (string, error) {
	for _, f := range primaryFlags {
		if isFlagPassed(f) {
			return f, nil
		}
	}
	return "", errors.New("no flag was provided")
}
