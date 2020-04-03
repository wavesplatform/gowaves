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
	Ride4DApps // RIDE V3
	OrderV3
	ReduceNFTFee
	BlockReward // 14
	BlockV5
	MultiPaymentInvokeScript // RIDE V4
	LeaseExpiration
)

type FeatureInfo struct {
	Implemented bool
	Description string
}

var FeaturesInfo = map[Feature]FeatureInfo{
	SmallerMinimalGeneratingBalance: {true, "Minimum Generating Balance of 1000 WAVES"},
	NG:                              {true, "NG Protocol"},
	MassTransfer:                    {true, "Mass Transfer Transaction"},
	SmartAccounts:                   {true, "Smart Accounts"},
	DataTransaction:                 {true, "Data Transaction"},
	BurnAnyTokens:                   {true, "Burn Any Tokens"},
	FeeSponsorship:                  {true, "Fee Sponsorship"},
	FairPoS:                         {true, "Fair PoS"},
	SmartAssets:                     {true, "Smart Assets"},
	SmartAccountTrading:             {true, "Smart Account Trading"},
	Ride4DApps:                      {true, "RIDE 4 DAPPS"},
	OrderV3:                         {true, "Order Version 3"},
	ReduceNFTFee:                    {true, "Reduce NFT fee"},
	BlockReward:                     {true, "Block Reward and Community Driven Monetary Policy"},
	BlockV5:                         {true, "VRF and Protobuf"},
	MultiPaymentInvokeScript:        {true, "Ride V4 and multiple attached payments for Invoke Script Transaction"},
	LeaseExpiration:                 {false, "Lease Expiration"},
}
