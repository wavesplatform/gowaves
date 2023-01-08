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
	./wallet -show -wallet <wallet path> // Show existing wallet credentials
	./wallet -new // Generate a seed phrase and a wallet 
	./wallet -seed-phrase "..." // Import a seed phrase
	./wallet -seed-phrase-base58 "..." // Import a Base58 encoded seed phrase
	./wallet -account-seed "..." // Import a Base58 encoded account seed
`

type Opts struct {
	PathToWallet       string
	AccountNumber      int
	Scheme             proto.Scheme
	SeedPhrase         string
	IsSeedPhraseBase58 bool
	Base58AccountSeed  string
	New                bool
	Show               bool
}

type SchemeFlag byte

func (sch *SchemeFlag) String() string {
	str := string(*sch)
	return str
}
func (sch *SchemeFlag) Set(s string) error {
	val := []byte(s)
	if len(val) > 1 {
		return errors.New("failed to parse scheme: more than one letter was provided")
	}

	*sch = SchemeFlag(val[0])
	return nil
}

func main() {
	opts := Opts{}
	var scheme SchemeFlag
	flag.BoolVar(&opts.New, newOpt, false, "Generate and add a new seed phrase (Primary flag)")
	flag.BoolVar(&opts.New, showOpt, false, "Show existing wallet credentials (Primary flag)")
	flag.StringVar(&opts.SeedPhrase, seedPhraseOpt, "", "Import a seed phrase (Primary flag)")
	flag.BoolVar(&opts.IsSeedPhraseBase58, seedPhraseBase58Opt, false, "Import a base58-encoded seed phrase (Primary flag)")
	flag.StringVar(&opts.Base58AccountSeed, accountSeedBase58Opt, "", "Import a base58-encoded account seed (Primary flag)")
	flag.StringVar(&opts.PathToWallet, "wallet", "", "Path to the wallet file")
	flag.IntVar(&opts.AccountNumber, "number", 0, "Account number. 0 is default")
	flag.Var(&scheme, schemeOpt, "Network scheme: MainNet=W, TestNet=T, StageNet=S, CustomNet=E. MainNet is default")

	flag.Parse()

	possibleFlags := []string{newOpt, showOpt, seedPhraseOpt, seedPhraseBase58Opt, accountSeedBase58Opt}
	if moreThanOnePrimaryFlagPassed(possibleFlags) {
		fmt.Print("failed to handle more than one primary flag")
		showUsageAndExit()
	}

	if isFlagPassed(schemeOpt) {
		opts.Scheme = byte(scheme)
	} else {
		opts.Scheme = proto.MainNetScheme
	}

	command, err := passedCommand(possibleFlags)
	if err != nil {
		fmt.Println(err)
		showUsageAndExit()
	}

	switch command {
	case showOpt:
		err = show(opts)
		if err != nil {
			log.Printf("failed to show wallet's credentials: %v", err)
		}
	case newOpt, seedPhraseOpt, seedPhraseBase58Opt, accountSeedBase58Opt:
		err = createWallet(opts, command)
		if err != nil {
			log.Printf("failed to create a new wallet: %v", err)
		}
	default:
		showUsageAndExit()
	}
}

func show(opts Opts) error {
	walletPath, err := getWalletPath(opts.PathToWallet)
	if err != nil {
		return errors.Wrap(err, "failed to handle wallet's path")
	}
	if !exists(walletPath) {
		return errors.New("wallet does not exist")
	}

	fmt.Print("Enter password to decode your wallet: ")
	pass, err := gopass.GetPasswd()
	if err != nil {
		return errors.Wrap(err, "failed to get the input password")
	}
	if len(pass) == 0 {
		return errors.New("empty password")
	}

	b, err := os.ReadFile(walletPath) // #nosec: in this case check for prevent G304 (CWE-22) is not necessary
	if err != nil {
		return errors.Wrap(err, "failed to read the wallet")
	}
	wlt, err := wallet.Decode(b, pass)
	if err != nil {
		return errors.Wrap(err, "failed to decode the wallet")

	}

	for i, s := range wlt.AccountSeeds() {
		accountSeedDigest, err := crypto.NewDigestFromBytes(s)
		if err != nil {
			return errors.Wrap(err, "failed to receive digest from account seed bytes")
		}
		pk, sk, address, err := generateOnAccountSeed(accountSeedDigest, opts.Scheme)
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

func generateWalletCredentials(choice string, opts Opts) (*WalletCredentials, error) {
	var walletCredentials *WalletCredentials

	switch choice {
	case newOpt:
		seedPhrase, err := generateMnemonic()
		if err != nil {
			return nil, errors.Wrap(err, "failed to generate seed phrase")
		}
		accountSeed, pk, sk, address, err := generateOnSeedPhrase(seedPhrase, opts.AccountNumber, opts.Scheme)
		if err != nil {
			return nil, err
		}
		walletCredentials = &WalletCredentials{
			accountSeed: accountSeed,
			pk:          pk,
			sk:          sk,
			address:     address,
		}

		fmt.Printf("Seed Phrase: '%s'\n", seedPhrase)
	case seedPhraseOpt:
		if opts.SeedPhrase == "" {
			return nil, errors.Wrap(wrongProgramArguments, "no seed phrase was provided")
		}
		if opts.IsSeedPhraseBase58 {
			return nil, errors.Wrap(wrongProgramArguments, "seed phrase base-58-encoding flag was true, but non-base-58 option was chosen")
		}

		accountSeed, pk, sk, address, err := generateOnSeedPhrase(opts.SeedPhrase, opts.AccountNumber, opts.Scheme)
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
		if !opts.IsSeedPhraseBase58 {
			return nil, errors.Wrap(wrongProgramArguments, "seed phrase base-58-encoding flag was false, but base-58 option was chosen")
		}
		b, err := base58.Decode(opts.SeedPhrase)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode base58-encoded seed phrase")
		}
		decodedSeedPhrase := string(b)

		accountSeed, pk, sk, address, err := generateOnSeedPhrase(decodedSeedPhrase, opts.AccountNumber, opts.Scheme)
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
		if opts.Base58AccountSeed == "" {
			return nil, errors.Wrap(wrongProgramArguments, "no base58 account seed was provided")
		}
		accountSeed, err := crypto.NewDigestFromBase58(opts.Base58AccountSeed)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode base58-encoded account seed")
		}
		pk, sk, address, err := generateOnAccountSeed(accountSeed, opts.Scheme)
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

func createWallet(opts Opts, command string) error {
	walletPath, err := getWalletPath(opts.PathToWallet)
	if err != nil {
		return errors.Wrap(err, "failed to handle wallet's path")
	}

	var walletCredentials *WalletCredentials

	walletCredentials, err = generateWalletCredentials(command, opts)
	if err != nil {
		return errors.Wrap(err, "failed to generate wallet's credentials")
	}
	if walletCredentials == nil {
		return errors.New("failed to generate wallet's credentials")
	}

	fmt.Print("Enter password to encode your account seed: ")
	pass, err := gopass.GetPasswd()
	if err != nil {
		return errors.Wrap(err, "failed to get the password to encode the account seed")
	}

	if len(pass) == 0 {
		return errors.Wrap(err, "the password's length is zero")
	}

	var wlt wallet.Wallet
	if exists(walletPath) {
		fmt.Print("Wallet already exists on the provided path. Rewrite? Y/N: ")
		var a string
		_, err := fmt.Scanf("%s", &a)
		if err != nil {
			return errors.Wrap(err, "failed to get the answer on rewriting the existing wallet")
		}
		answer := strings.ToLower(a)
		switch answer {
		case "y":
			wlt = wallet.NewWallet()
		case "n":
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

	bts, err := wlt.Encode(pass)
	if err != nil {
		return errors.Wrap(err, "failed to encode the wallet with the provided password")

	}

	err = os.WriteFile(walletPath, bts, 0600)
	if err != nil {
		return errors.Wrap(err, "failed to write the wallet's data to the wallet")

	}
	fmt.Println()
	log.Printf("Account Number: %d\n", opts.AccountNumber)
	log.Printf("Account Seed:   %s\n", walletCredentials.accountSeed.String())
	log.Printf("Public Key:     %s\n", walletCredentials.pk.String())
	log.Printf("Secret Key:     %s\n", walletCredentials.sk.String())
	log.Printf("Address:        %s\n", walletCredentials.address.String())
	fmt.Printf("Your wallet has been successfully created in %s\n", walletPath)
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

func moreThanOnePrimaryFlagPassed(flags []string) bool {
	passedFlagsNum := 0
	for _, f := range flags {
		if isFlagPassed(f) {
			passedFlagsNum++
		}
	}

	return passedFlagsNum > 1
}

func passedCommand(flags []string) (string, error) {
	for _, f := range flags {
		if isFlagPassed(f) {
			return f, nil
		}
	}
	return "", errors.New("no flag was provided")
}
