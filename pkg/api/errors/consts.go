package errors

const (
	UnknownErrorID   ErrorID = 0
	WrongJsonErrorID ErrorID = 1
)

// API Auth
const (
	ApiKeyNotValidErrorID        ApiAuthErrorID = 2
	TooBigArrayAllocationErrorID ApiAuthErrorID = 10
)

// VALIDATION
const (
	InvalidSignatureErrorID                     ValidationErrorID = 101
	InvalidAddressErrorID                       ValidationErrorID = 102
	InvalidPublicKeyErrorID                     ValidationErrorID = 108
	InvalidMessageErrorID                       ValidationErrorID = 110
	InvalidNameErrorID                          ValidationErrorID = 111
	StateCheckFailedErrorID                     ValidationErrorID = 112
	OverflowErrorID                             ValidationErrorID = 113
	ToSelfErrorID                               ValidationErrorID = 114
	MissingSenderPrivateKeyErrorID              ValidationErrorID = 115
	InvalidIdsErrorID                           ValidationErrorID = 116
	CustomValidationErrorErrorID                ValidationErrorID = 199
	BlockDoesNotExistErrorID                    ValidationErrorID = 301
	AliasDoesNotExistErrorID                    ValidationErrorID = 302
	MistimingErrorID                            ValidationErrorID = 303
	DataKeyDoesNotExistErrorID                  ValidationErrorID = 304
	ScriptCompilerErrorID                       ValidationErrorID = 305
	ScriptExecutionErrorErrorID                 ValidationErrorID = 306
	TransactionNotAllowedByAccountScriptErrorID ValidationErrorID = 307
	TransactionNotAllowedByAssetScriptErrorID   ValidationErrorID = 308
)

// TRANSACTIONS
const (
	TransactionDoesNotExistErrorID    TransactionErrorID = 311
	UnsupportedTransactionTypeErrorID TransactionErrorID = 312
	AssetDoesNotExistErrorID          TransactionErrorID = 313
	NegativeAmountErrorID             TransactionErrorID = 111
	InsufficientFeeErrorID            TransactionErrorID = 112
	NegativeMinFeeErrorID             TransactionErrorID = 114
	NonPositiveAmountErrorID          TransactionErrorID = 115
	AlreadyInStateErrorID             TransactionErrorID = 400
	AccountBalanceErrorsErrorID       TransactionErrorID = 402
	OrderInvalidErrorID               TransactionErrorID = 403
	InvalidChainIdErrorID             TransactionErrorID = 404
	InvalidProofsErrorID              TransactionErrorID = 405
	InvalidTransactionIdErrorID       TransactionErrorID = 4001
	InvalidBlockIdErrorID             TransactionErrorID = 4002
	InvalidAssetIdErrorID             TransactionErrorID = 4007
)

var errorNames = map[Identifier]string{
	UnknownErrorID:   "UnknownError",
	WrongJsonErrorID: "WrongJsonError",

	ApiKeyNotValidErrorID:                       "ApiKeyNotValidError",
	TooBigArrayAllocationErrorID:                "TooBigArrayAllocationError",
	InvalidSignatureErrorID:                     "InvalidSignatureError",
	InvalidAddressErrorID:                       "InvalidAddressError",
	InvalidPublicKeyErrorID:                     "InvalidPublicKeyError",
	InvalidMessageErrorID:                       "InvalidMessageError",
	InvalidNameErrorID:                          "InvalidNameError",
	StateCheckFailedErrorID:                     "StateCheckFailedError",
	OverflowErrorID:                             "OverflowError",
	ToSelfErrorID:                               "ToSelfError",
	MissingSenderPrivateKeyErrorID:              "MissingSenderPrivateKeyError",
	InvalidIdsErrorID:                           "InvalidIdsError",
	CustomValidationErrorErrorID:                "CustomValidationErrorError",
	BlockDoesNotExistErrorID:                    "BlockDoesNotExistError",
	AliasDoesNotExistErrorID:                    "AliasDoesNotExistError",
	MistimingErrorID:                            "MistimingError",
	DataKeyDoesNotExistErrorID:                  "DataKeyDoesNotExistError",
	ScriptCompilerErrorID:                       "ScriptCompilerError",
	ScriptExecutionErrorErrorID:                 "ScriptExecutionErrorError",
	TransactionNotAllowedByAccountScriptErrorID: "TransactionNotAllowedByAccountScriptError",
	TransactionNotAllowedByAssetScriptErrorID:   "TransactionNotAllowedByAssetScriptError",

	TransactionDoesNotExistErrorID:    "TransactionDoesNotExistError",
	UnsupportedTransactionTypeErrorID: "UnsupportedTransactionTypeError",
	AssetDoesNotExistErrorID:          "AssetDoesNotExistError",
	NegativeAmountErrorID:             "NegativeAmountError",
	InsufficientFeeErrorID:            "InsufficientFeeError",
	NegativeMinFeeErrorID:             "NegativeMinFeeError",
	NonPositiveAmountErrorID:          "NonPositiveAmountError",
	AlreadyInStateErrorID:             "AlreadyInStateError",
	AccountBalanceErrorsErrorID:       "AccountBalanceErrorsError",
	OrderInvalidErrorID:               "OrderInvalidError",
	InvalidChainIdErrorID:             "InvalidChainIdError",
	InvalidProofsErrorID:              "InvalidProofsError",
	InvalidTransactionIdErrorID:       "InvalidTransactionIdError",
	InvalidBlockIdErrorID:             "InvalidBlockIdError",
	InvalidAssetIdErrorID:             "InvalidAssetIdError",
}
