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

type WavesRegularBalance struct {
	Address       proto.WavesAddress `json:"address"`
	Balance       uint64             `json:"balance"`
	Confirmations uint64             `json:"confirmations"`
}

func (a *App) WavesRegularBalanceByAddress(addr proto.WavesAddress) (WavesRegularBalance, error) {
	regularBalance, err := a.state.WavesBalance(proto.NewRecipientFromAddress(addr))
	if err != nil {
		return WavesRegularBalance{}, err
	}
	return WavesRegularBalance{
		Address:       addr,
		Balance:       regularBalance,
		Confirmations: 0, // nickeskov: always 0 confirmations because this method returns the latest balance
	}, nil
}
