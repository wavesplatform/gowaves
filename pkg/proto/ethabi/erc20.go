package ethabi

const (
	Erc20TransferSignature Signature = "transfer(address,uint256)"
	ethereumAddressSize    int       = 20
	NumberOfERC20Arguments      int       = 2
)

var erc20Methods = map[Selector]Method{
	Erc20TransferSignature.Selector(): {
		RawName: "transfer",
		Inputs: Arguments{
			Argument{
				Name: "_to",
				Type: Type{
					Size:       ethereumAddressSize,
					T:          AddressTy,
					stringKind: "address",
				},
			},
			Argument{
				Name: "_value",
				Type: Type{
					Size:       256,
					T:          UintTy,
					stringKind: "uint256",
				},
			},
		},
		Payments: nil,
		Sig:      Erc20TransferSignature,
	},
}
