package api

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

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

type AddressBalance struct {
	Address       proto.WavesAddress `json:"address"`
	Confirmations uint               `json:"confirmations"`
	Balance       uint64             `json:"balance"`
}

func (a *App) AddressesBalance(addr proto.WavesAddress) (*AddressBalance, error) {
	regularBalance, err := a.state.WavesBalance(proto.NewRecipientFromAddress(addr))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get regular WAVES balance for address %q", addr.String())
	}
	return &AddressBalance{
		Address:       addr,
		Confirmations: 0,
		Balance:       regularBalance,
	}, nil
}
