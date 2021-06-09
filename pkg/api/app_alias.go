package api

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
)

func (a *App) AddrByAlias(alias proto.Alias) (proto.Address, error) {
	addr, err := a.state.AddrByAlias(alias)
	if err != nil {
		if state.IsNotFound(err) {
			return proto.Address{}, err
		}
		return proto.Address{}, errors.Wrapf(err, "failed to find addr by alias %q", alias.String())
	}
	return addr, nil
}
