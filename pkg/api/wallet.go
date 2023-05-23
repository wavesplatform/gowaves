package api

import "github.com/mr-tron/base58"

// WalletSeeds returns wallet seeds in base58 encoding.
func (a *App) WalletSeeds() []string {
	seeds := a.services.Wallet.AccountSeeds()

	seeds58 := make([]string, 0, len(seeds))
	for _, seed := range seeds {
		seed58 := base58.Encode(seed)
		seeds58 = append(seeds58, seed58)
	}
	return seeds58
}
