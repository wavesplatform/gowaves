package api

func (a *App) Addresses() ([]string, error) {
	accounts, err := a.Accounts()
	if err != nil {
		return nil, err
	}

	addresses := make([]string, 0, len(accounts))
	for i := range accounts {
		addresses = append(addresses, accounts[i].Address.String())
	}

	return addresses, nil
}
