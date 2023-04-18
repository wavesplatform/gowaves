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

func (a *App) AliasesByAddr(addr proto.WavesAddress) ([]proto.Alias, error) {
	aliases, err := a.state.AliasesByAddr(addr)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find aliases by addr %q", addr.String())
	}
	if len(aliases) == 0 {
		return nil, nil
	}
	out := make([]proto.Alias, len(aliases))
	for i := range aliases {
		out[i] = *proto.NewAlias(a.scheme(), aliases[i])
	}
	return out, err
}
