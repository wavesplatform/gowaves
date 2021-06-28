package metamask

const (
	// EthAddressLength is the expected length of the address in bytes
	EthAddressLength = 20
)

// EthAddress represents the 20 byte address of an Ethereum account.
type EthAddress [EthAddressLength]byte

func (a *EthAddress) Bytes() []byte {
	if a != nil {
		return a[:]
	}
	return nil
}

//func (a *EthAddress) unmarshalFromFastRLP(val *fastrlp.Value) error {
//	if err := val.GetAddr(a[:]); err != nil {
//		return errors.Wrap(err, "failed to unmarshal Address from fastRLP value")
//	}
//	return nil
//}
//
//func (a *EthAddress) marshalToFastRLP(arena *fastrlp.Arena) *fastrlp.Value {
//	return arena.NewBytes(a.Bytes())
//}
