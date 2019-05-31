package settings

type Feature int16

const (
	SmallerMinimalGeneratingBalance Feature = iota + 1
	NG
	MassTransfer
	SmartAccounts
	DataTransaction
	BurnAnyTokens
	FeeSponsorship
	FairPoS
	SmartAssets
	SmartAccountTrading
	Ride4DApps
	OrderV3
	ReduceNFTFee
)

type FeatureInfo struct {
	Implemented bool
	Description string
}

var FeaturesInfo = map[Feature]FeatureInfo{
	SmallerMinimalGeneratingBalance: {true, "Minimum Generating Balance of 1000 WAVES"},
	NG:                              {false, "NG Protocol"},
	MassTransfer:                    {false, "Mass Transfer Transaction"},
	SmartAccounts:                   {false, "Smart Accounts"},
	DataTransaction:                 {false, "Data Transaction"},
	BurnAnyTokens:                   {false, "Burn Any Tokens"},
	FeeSponsorship:                  {false, "Fee Sponsorship"},
	FairPoS:                         {true, "Fair PoS"},
	SmartAssets:                     {false, "Smart Assets"},
	SmartAccountTrading:             {false, "Smart Account Trading"},
	Ride4DApps:                      {false, "RIDE 4 DAPPS"},
	OrderV3:                         {false, "Order Version 3"},
	ReduceNFTFee:                    {false, "Reduce NFT fee"},
}
