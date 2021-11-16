package ethabi

const (
	erc20TransferSignature Signature = "transfer(address,uint256)"
	ethereumAddressSize    int       = 20
)

var erc20Methods = map[Selector]Method{
	erc20TransferSignature.Selector(): {
		RawName: "transfer",
		Inputs: Arguments{
			Argument{
				Name: "_to",
				Type: Type{
					Size:       ethereumAddressSize,
					T:          AddressType,
					stringKind: "address",
				},
			},
			Argument{
				Name: "_value",
				Type: Type{
					Size:       256,
					T:          UintType,
					stringKind: "uint256",
				},
			},
		},
		Payments: nil,
		Sig:      erc20TransferSignature,
	},
}
