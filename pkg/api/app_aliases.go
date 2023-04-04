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

func (a *App) AliasesByAddr(addr proto.WavesAddress) ([]string, error) {
	aliases, err := a.state.AliasesByAddr(addr)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find aliases by addr %q", addr.String())
	}
	return aliases, err
}
