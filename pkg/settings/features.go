package settings

type Feature int16

// Feature IDs.
// Order is important, do not change implemented features IDs.
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
	ReducedNFTFee
	BlockReward             // 14
	BlockV5                 // 15
	RideV5                  // 16
	RideV6                  // 17
	ConsensusImprovements   // 18
	BlockRewardDistribution // 19
	CappedRewards           // 20
	XTNBuyBackCessation     // 21
	LightNode               // 22
	BoostBlockReward        // 23
	ecrecoverFix            // 24, intentionally package private
	InvokeExpression        // 25
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
	Ride4DApps:                      {true, "Ride For DApp"},
	OrderV3:                         {true, "Order Version 3"},
	ReducedNFTFee:                   {true, "Reduced NFT Fee"},
	BlockReward:                     {true, "Block Reward and Community Driven Monetary Policy"},
	BlockV5:                         {true, "Ride V4, VRF, Protobuf, Failed Transactions"},
	RideV5:                          {true, "Ride V5, DApp-to-DApp Invocations"},
	RideV6:                          {true, "Ride V6, MetaMask Support"},
	ConsensusImprovements:           {true, "Consensus and MetaMask Updates"},
	BlockRewardDistribution:         {true, "Block Reward Distribution"},
	CappedRewards:                   {true, "Capped XTN Buy-back and DAO Amounts"},
	XTNBuyBackCessation:             {true, "XTN Buy-back Cessation"},
	LightNode:                       {true, "Light Node"},
	BoostBlockReward:                {true, "Boost Block Reward"},
	ecrecoverFix:                    {true, "Fixed ecrecover function"},
	InvokeExpression:                {false, "InvokeExpression"},
}

// LastFeature returns the last implemented feature.
func LastFeature() Feature {
	var f Feature
	for feature, info := range FeaturesInfo {
		if info.Implemented {
			f = max(f, feature)
		}
	}
	return f
}
