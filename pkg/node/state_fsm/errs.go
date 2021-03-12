package state_fsm

import (
	"errors"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

var TimeoutErr = proto.NewInfoMsg(errors.New("timeout"))
