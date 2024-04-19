package proto

//go:generate stringer -type=TransactionType
type TransactionType byte

// All transaction types supported.
const (
	GenesisTransaction          TransactionType = iota + 1 // 1 - Genesis transaction
	PaymentTransaction                                     // 2 - Payment transaction
	IssueTransaction                                       // 3 - Issue transaction
	TransferTransaction                                    // 4 - Transfer transaction
	ReissueTransaction                                     // 5 - Reissue transaction
	BurnTransaction                                        // 6 - Burn transaction
	ExchangeTransaction                                    // 7 - Exchange transaction
	LeaseTransaction                                       // 8 - Lease transaction
	LeaseCancelTransaction                                 // 9 - LeaseCancel transaction
	CreateAliasTransaction                                 // 10 - CreateAlias transaction
	MassTransferTransaction                                // 11 - MassTransfer transaction
	DataTransaction                                        // 12 - Data transaction
	SetScriptTransaction                                   // 13 - SetScript transaction
	SponsorshipTransaction                                 // 14 - Sponsorship transaction
	SetAssetScriptTransaction                              // 15 - SetAssetScript transaction
	InvokeScriptTransaction                                // 16 - InvokeScript transaction
	UpdateAssetInfoTransaction                             // 17 - UpdateAssetInfo transaction
	EthereumMetamaskTransaction                            // 18 - EthereumMetamask transaction: received from MetaMask
	InvokeExpressionTransaction                            // 19 - InvokeExpression transaction
)
