package api

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func (a *App) scheme() proto.Scheme {
	return a.services.Scheme
}

func (a *App) AddrByAlias(alias proto.Alias) (proto.Address, error) {
	addr, err := a.state.AddrByAlias(alias)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find addr by alias %q", alias.String())
	}
	return addr, nil
}

func (a *App) AliasesByAddr(addr proto.Address) ([]string, error) {
	wavesAddr, err := addr.ToWavesAddress(a.services.Scheme)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid waves address %s", addr.String())
	}

	aliases, err := a.state.AliasesByAddr(&wavesAddr)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find aliases  by addr %s", addr.String())
	}
	return aliases, err
}
