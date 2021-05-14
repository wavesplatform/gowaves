package api

import "github.com/pkg/errors"

func (a *App) Addresses() ([]string, error) {
	accounts, err := a.Accounts()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get wallet accounts")
	}

	addresses := make([]string, 0, len(accounts))
	for i := range accounts {
		addresses = append(addresses, accounts[i].Address.String())
	}

	return addresses, nil
}
