package main

import (
	"encoding/binary"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/user"
	"path/filepath"

	flag "github.com/spf13/pflag"

	"github.com/mr-tron/base58"
	"github.com/pkg/errors"

	"github.com/howeyc/gopass"
	"github.com/tyler-smith/go-bip39"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/wallet"
)

type WalletGenerationChoice string

const (
	newSeedPhrase             = WalletGenerationChoice("1")
	existingSeedPhrase        = WalletGenerationChoice("2")
	existingBase58SeedPhrase  = WalletGenerationChoice("3")
	existingBase58AccountSeed = WalletGenerationChoice("4")
)

const (
	defaultBitSize = 160
)

var usage = `

Usage:
  wallet <command> [flags]

Available Commands:
  create       Create a wallet
  show         Print the wallet data

Flags:
 --wallet <wallet path>
 --number <account number>
 --scheme <mainnet | testnet | stagenet>
 --seedPhraseBase58 <true | false>
 --seedPhrase <seed phrase> (base-58 or non-base-58)
 --accountSeed <account seed>  (base-58 only)

Examples:
 1) Create a new wallet based on new generated seed phrase:
	./wallet create --scheme mainnet

 2) Create a new wallet based on an existing seed phrase:
	./wallet create --scheme mainnet --seedPhrase "one two three one two three one two three one two three one two three"

 3) Show the credentials of an existing wallet
	./wallet show --wallet "/home/user/wallet/.waves"
`

type Opts struct {
	PathToWallet  string
	AccountNumber int
	Scheme        proto.Scheme

	SeedPhrase         string
	IsSeedPhraseBase58 bool

	Base58AccountSeed string
}

func main() {
	opts := Opts{}
	var scheme string
	flag.StringVar(&opts.PathToWallet, "wallet", "", "Path to wallet")
	flag.IntVar(&opts.AccountNumber, "number", 0, "Input account number. 0 is default")
	flag.StringVar(&scheme, "scheme", "mainnet", "Input the network scheme: mainnet, testnet, stagenet")
	flag.StringVar(&opts.SeedPhrase, "seedPhrase", "", "Input your seed phrase")
	flag.BoolVar(&opts.IsSeedPhraseBase58, "seedPhraseBase58", false, "Seed phrase is written in Base58 format")

	flag.StringVar(&opts.Base58AccountSeed, "accountSeed", "", "Input your account seed in Base58 format")

	flag.Parse()

	command := flag.Arg(0)

	schemeByte, err := proto.ParseSchemeFromStr(scheme)
	if err != nil {
		fmt.Printf("failed to parse network scheme: %v", err)
		return
	}
	opts.Scheme = schemeByte

	switch command {
	case "create":
		err := createWallet(opts)
		if err != nil {
			fmt.Printf("failed to create a new wallet: %v", err)
		}
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

	fmt.Print("Enter password to decode your wallet: ")
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

	for _, s := range wlt.AccountSeeds() {
		accountSeedDigest, err := crypto.NewDigestFromBytes(s)
		if err != nil {
			fmt.Printf("err: %v", err)
			return
		}

		sk, pk, err := crypto.GenerateKeyPair(accountSeedDigest.Bytes())
		if err != nil {
			fmt.Printf("err: %v", err)
			return
		}
		address, err := proto.NewAddressFromPublicKey(opts.Scheme, pk)
		if err != nil {
			fmt.Printf("err: %v", err)
			return
		}

		fmt.Printf("Account seed: %s\n", accountSeedDigest.String())
		fmt.Printf("Public Key: %s\n", pk.String())
		fmt.Printf("Secret Key: %s\n", sk.String())
		fmt.Printf("Address: %s\n", address.String())
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

func createWallet(opts Opts) error {
	walletPath := getWalletPath(opts.PathToWallet)
	fmt.Println(`Available options: 
		 1: Generate new seed phrase and wallet
		 2: Create a wallet based on an existing seed phrase. Requires seed phrase argument to be provided
		 3: Create a wallet based on an existing base58-encoded seed phrase. Requires seed phrase argument to be provided and seed-base58 flag marked "true"
		 4: Create a wallet based on an existing base58-encoded account seed. Requires account seed argument in Base58 format`)

	var walletCredentials WalletCredentials
	var choice WalletGenerationChoice
	_, err := fmt.Scanf("%s", &choice)
	if err != nil {
		fmt.Println("Interrupt")
		return err
	}
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
			accountSeed: accountSeed,
			pk:          pk,
			sk:          sk,
			address:     address,
		}

		fmt.Printf("Seed Phrase: '%s'\n", seedPhrase)

	case existingSeedPhrase:
		if opts.SeedPhrase == "" {
			return errors.Wrap(wrongProgramArguments, "no seed phrase was provided")
		}
		if opts.IsSeedPhraseBase58 {
			return errors.Wrap(wrongProgramArguments, "seed phrase base-58-encoding flag was true, but non-base-58 option was chosen")
		}

		accountSeed, pk, sk, address, err := generateOnSeedPhrase(opts.SeedPhrase, opts.AccountNumber, opts.Scheme)
		if err != nil {
			return err
		}
		walletCredentials = WalletCredentials{
			accountSeed: accountSeed,
			pk:          pk,
			sk:          sk,
			address:     address,
		}
	case existingBase58SeedPhrase:
		if !opts.IsSeedPhraseBase58 {
			return errors.Wrap(wrongProgramArguments, "seed phrase base-58-encoding flag was false, but base-58 option was chosen")
		}
		b, err := base58.Decode(opts.SeedPhrase)
		if err != nil {
			return errors.Wrap(err, "failed to decode base58-encoded seed phrase")
		}
		decodedSeedPhrase := string(b)

		accountSeed, pk, sk, address, err := generateOnSeedPhrase(decodedSeedPhrase, opts.AccountNumber, opts.Scheme)
		if err != nil {
			return err
		}
		walletCredentials = WalletCredentials{
			accountSeed: accountSeed,
			pk:          pk,
			sk:          sk,
			address:     address,
		}

	case existingBase58AccountSeed:
		if opts.Base58AccountSeed == "" {
			return errors.Wrap(wrongProgramArguments, "no base58 account seed was provided")
		}
		accountSeed, err := crypto.NewDigestFromBase58(opts.Base58AccountSeed)
		if err != nil {
			return errors.Wrap(err, "failed to decode base58-encoded account seed")
		}
		pk, sk, address, err := generateOnAccountSeed(accountSeed, opts.Scheme)
		if err != nil {
			return err
		}
		walletCredentials = WalletCredentials{
			accountSeed: accountSeed,
			pk:          pk,
			sk:          sk,
			address:     address,
		}

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
		fmt.Print("Wallet already exists on the provided path. Rewrite? Y/N")
		var answer string
		_, err := fmt.Scanf("%s", &answer)
		if err != nil {
			fmt.Println("Interrupt")
			return err
		}
		switch answer {
		case "Y":
			wlt = wallet.NewWallet()
		case "N":
			return errors.New("program interrupted")
		default:
			return errors.New("unknown command")
		}

	} else {
		wlt = wallet.NewWallet()
	}

	err = wlt.AddAccountSeed(walletCredentials.accountSeed.Bytes())
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
