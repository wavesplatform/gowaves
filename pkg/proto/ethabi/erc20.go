package ethabi

import (
	"github.com/pkg/errors"
)

const (
	erc20TransferSignature         Signature = "transfer(address,uint256)"
	EthereumAddressSize            int       = 20
	NumberOfERC20TransferArguments int       = 2
)

var (
	erc20TransferSelector = erc20TransferSignature.Selector()

	erc20Methods = map[Selector]Method{
		erc20TransferSelector: {
			RawName: "transfer",
			Inputs: Arguments{
				Argument{
					Name: "_to",
					Type: Type{
						Size:       EthereumAddressSize,
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
)

type ERC20TransferArguments struct {
	Recipient [EthereumAddressSize]byte
	Amount    int64
}

// GetERC20TransferArguments parses DecodedCallData to ERC20TransferArguments
func GetERC20TransferArguments(decodedData *DecodedCallData) (ERC20TransferArguments, error) {
	if len(decodedData.Inputs) != NumberOfERC20TransferArguments {
		return ERC20TransferArguments{}, errors.Errorf("invalid DecodedCallData.Inputs count: want %d, got %d",
			NumberOfERC20TransferArguments, len(decodedData.Inputs),
		)
	}

	// get recipient
	ethRecipientAddressBytes, ok := decodedData.Inputs[0].Value.(Bytes)
	if !ok {
		return ERC20TransferArguments{}, errors.New(
			"failed to cast first argument of DecodedCallData to Bytes",
		)
	}
	if len(ethRecipientAddressBytes) != EthereumAddressSize {
		return ERC20TransferArguments{}, errors.Errorf("invalid reccipient size: want %d, got %d",
			EthereumAddressSize, len(ethRecipientAddressBytes),
		)
	}
	var ethRecipient [EthereumAddressSize]byte
	copy(ethRecipient[:], ethRecipientAddressBytes)

	// get amount
	transferAmount, ok := decodedData.Inputs[1].Value.(BigInt)
	if !ok {
		return ERC20TransferArguments{}, errors.New(
			"failed to cast first argument of DecodedCallData to BigInt",
		)
	}
	if ok := transferAmount.V.IsInt64(); !ok {
		return ERC20TransferArguments{}, errors.Errorf(
			"failed to convert BigInt value to int64 (overflow), value is %s",
			transferAmount.V.String(),
		)
	}

	return ERC20TransferArguments{Recipient: ethRecipient, Amount: transferAmount.V.Int64()}, nil
}
