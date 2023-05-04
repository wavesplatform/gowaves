package testdata

import (
	"github.com/wavesplatform/gowaves/itests/config"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	SetAssetScriptMaxVersion = 3
	SetAssetScriptMinVersion = 2
)

type SetAssetScriptTestData[T any] struct {
	Account   config.AccountInfo
	AssetID   crypto.Digest
	Script    proto.Script
	Fee       uint64
	Timestamp uint64
	ChainID   proto.Scheme
	Expected  T
}

type SetAssetScriptExpectedValuesPositive struct {
	WavesDiffBalance int64
	AssetDiffBalance int64
	_                struct{}
}

type SetAssetScriptExpectedValuesNegative struct {
	WavesDiffBalance  int64
	AssetDiffBalance  int64
	ErrGoMsg          string
	ErrScalaMsg       string
	ErrBrdCstGoMsg    string
	ErrBrdCstScalaMsg string
	_                 struct{}
}

func NewSetAssetScriptTestData[T any](account config.AccountInfo, assetID crypto.Digest, script proto.Script,
	fee, timestamp uint64, chainID proto.Scheme, expected T) *SetAssetScriptTestData[T] {
	return &SetAssetScriptTestData[T]{
		Account:   account,
		AssetID:   assetID,
		Script:    script,
		Fee:       fee,
		Timestamp: timestamp,
		ChainID:   chainID,
		Expected:  expected,
	}
}

func GetSetAssetScriptPositiveData(suite *f.BaseSuite, assetID crypto.Digest) map[string]SetAssetScriptTestData[SetAssetScriptExpectedValuesPositive] {
	return map[string]SetAssetScriptTestData[SetAssetScriptExpectedValuesPositive]{
		"Valid script, true as expression": *NewSetAssetScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.GetScriptBytes(suite, "BQbtKNoM"),
			utl.MinSetAssetScriptFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SetAssetScriptExpectedValuesPositive{
				WavesDiffBalance: utl.MinSetAssetScriptFeeWaves,
				AssetDiffBalance: 0,
			}),
		"Valid script, size 8192 bytes": *NewSetAssetScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.GetScriptBytes(suite, "BgEEA3N0cgI3VGhpcyB0ZXh0IGlzIG5lY2Vzc2FyeSB0byBnZXQgdGhlIHNjcmlwdCBv"+
				"ZiB0aGUgODE5MiBieQQFc3RyXzECP1RoaXMgdGV4dCBpcyBuZWNlc3NhcnkgdG8gZ2V0IHRoZSBzY3JpcHQgb2YgdGhlIHJlc"+
				"XVpcmVkIHZvbHVtZQQFc3RyXzICgwFBbnkgdXNlciBjYW4gY3JlYXRlIHRoZWlyIG93biB0b2tlbiBvbiB0aGUgV2F2ZXMgYm"+
				"xvY2tjaGFpbiBhbmQgYWxzbyBzZXQgdGhlIHJ1bGVzIGZvciBpdHMgY2lyY3VsYXRpb24gYnkgYXNzaWduaW5nIGEgc2NyaXB"+
				"0IHRvIGl0LgQFc3RyXzMCbkZvciBleGFtcGxlLCBmb3IgaW4tZ2FtZSBjdXJyZW5jeSwgeW91IGNhbiBhbGxvdyBvbmx5IHRy"+
				"YW5zYWN0aW9ucyBiZXR3ZWVuIGNoYXJhY3RlcnMgd2l0aCBjZXJ0YWluIHByb3BlcnRpZXMuBAVzdHJfNAJrQSB0b2tlbiB3a"+
				"XRoIGFuIGFzc2lnbmVkIHNjcmlwdCBpcyBjYWxsZWQgYSBzbWFydCBhc3NldCwgYW5kIHRoZSBhc3NpZ25lZCBzY3JpcHQgaX"+
				"MgY2FsbGVkIGFuIGFzc2V0IHNjcmlwdC4EBnRleHRfMQkAuwkCCQDMCAIFBXN0cl8xCQDMCAIFBXN0cl8yCQDMCAIFBXN0cl8"+
				"zCQDMCAIFBXN0cl80BQNuaWwCAyAmIAQGdGV4dF8yCQC7CQIJAMwIAgUFc3RyXzEJAMwIAgUFc3RyXzIJAMwIAgUFc3RyXzMJ"+
				"AMwIAgUFc3RyXzQFA25pbAIDICYgBAZ0ZXh0XzMJALsJAgkAzAgCBQVzdHJfMQkAzAgCBQVzdHJfMgkAzAgCBQVzdHJfMwkAz"+
				"AgCBQVzdHJfNAUDbmlsAgMgJiAEBnRleHRfNAkAuwkCCQDMCAIFBXN0cl8xCQDMCAIFBXN0cl8yCQDMCAIFBXN0cl8zCQDMCA"+
				"IFBXN0cl80BQNuaWwCAyAmIAQEc3RyMQI/VGhpcyB0ZXh0IGlzIG5lY2Vzc2FyeSB0byBnZXQgdGhlIHNjcmlwdCBvZiB0aGU"+
				"gcmVxdWlyZWQgdm9sdW1lBARzdHIyAoMBQW55IHVzZXIgY2FuIGNyZWF0ZSB0aGVpciBvd24gdG9rZW4gb24gdGhlIFdhdmVz"+
				"IGJsb2NrY2hhaW4gYW5kIGFsc28gc2V0IHRoZSBydWxlcyBmb3IgaXRzIGNpcmN1bGF0aW9uIGJ5IGFzc2lnbmluZyBhIHNjc"+
				"mlwdCB0byBpdC4EBHN0cjMCbkZvciBleGFtcGxlLCBmb3IgaW4tZ2FtZSBjdXJyZW5jeSwgeW91IGNhbiBhbGxvdyBvbmx5IH"+
				"RyYW5zYWN0aW9ucyBiZXR3ZWVuIGNoYXJhY3RlcnMgd2l0aCBjZXJ0YWluIHByb3BlcnRpZXMuBARzdHI0AmtBIHRva2VuIHd"+
				"pdGggYW4gYXNzaWduZWQgc2NyaXB0IGlzIGNhbGxlZCBhIHNtYXJ0IGFzc2V0LCBhbmQgdGhlIGFzc2lnbmVkIHNjcmlwdCBp"+
				"cyBjYWxsZWQgYW4gYXNzZXQgc2NyaXB0LgQFdGV4dDEJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAz"+
				"AgCBQRzdHI0BQNuaWwCAyAmIAQFdGV4dDIJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdH"+
				"I0BQNuaWwCAyAmIAQFdGV4dDMJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdHI0BQNuaWw"+
				"CAyAmIAQFdGV4dDQJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdHI0BQNuaWwCAyAmIAQF"+
				"dGV4dDUJALsJAgkAzAgCBQRzdH"+
				"IxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdHI0BQNuaWwCAyAmIAQFc3RyMTECP1RoaXMgdGV4dCBpcyBuZWNlc3N"+
				"hcnkgdG8gZ2V0IHRoZSBzY3JpcHQgb2YgdGhlIHJlcXVpcmVkIHZvbHVtZQQFc3RyMjICgwFBbnkgdXNlciBjYW4gY3JlYXRl"+
				"IHRoZWlyIG93biB0b2tlbiBvbiB0aGUgV2F2ZXMgYmxvY2tjaGFpbiBhbmQgYWxzbyBzZXQgdGhlIHJ1bGVzIGZvciBpdHMgY"+
				"2lyY3VsYXRpb24gYnkgYXNzaWduaW5nIGEgc2NyaXB0IHRvIGl0LgQFc3RyMzMCbkZvciBleGFtcGxlLCBmb3IgaW4tZ2FtZS"+
				"BjdXJyZW5jeSwgeW91IGNhbiBhbGxvdyBvbmx5IHRyYW5zYWN0aW9ucyBiZXR3ZWVuIGNoYXJhY3RlcnMgd2l0aCBjZXJ0YWl"+
				"uIHByb3BlcnRpZXMuBAVzdHI0NAJrQSB0b2tlbiB3aXRoIGFuIGFzc2lnbmVkIHNjcmlwdCBpcyBjYWxsZWQgYSBzbWFydCBh"+
				"c3NldCwgYW5kIHRoZSBhc3NpZ25lZCBzY3JpcHQgaXMgY2FsbGVkIGFuIGFzc2V0IHNjcmlwdC4EBnRleHQxMQkAuwkCCQDMC"+
				"AIFBHN0cjEJAMwIAgUEc3RyMgkAzAgCBQRzdHIzCQDMCAIFBHN0cjQFA25pbAIDICYgBAZ0ZXh0MjIJALsJAgkAzAgCBQRzdH"+
				"IxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdHI0BQNuaWwCAyAmIAQGdGV4dDMzCQC7CQIJAMwIAgUEc3RyMQkAzAg"+
				"CBQRzdHIyCQDMCAIFBHN0cjMJAMwIAgUEc3RyNAUDbmlsAgMgJiAEBnRleHQ0NAkAuwkCCQDMCAIFBHN0cjEJAMwIAgUEc3Ry"+
				"MgkAzAgCBQRzdHIzCQDMCAIFBHN0cjQFA25pbAIDICYgBAZ0ZXh0NTUJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIA"+
				"gUEc3RyMwkAzAgCBQRzdHI0BQNuaWwCAyAmIAQGc3RyMTExAj9UaGlzIHRleHQgaXMgbmVjZXNzYXJ5IHRvIGdldCB0aGUgc2"+
				"NyaXB0IG9mIHRoZSByZXF1aXJlZCB2b2x1bWUEBnN0cjIyMgKDAUFueSB1c2VyIGNhbiBjcmVhdGUgdGhlaXIgb3duIHRva2V"+
				"uIG9uIHRoZSBXYXZlcyBibG9ja2NoYWluIGFuZCBhbHNvIHNldCB0aGUgcnVsZXMgZm9yIGl0cyBjaXJjdWxhdGlvbiBieSBh"+
				"c3NpZ25pbmcgYSBzY3JpcHQgdG8gaXQuBAZzdHIzMzMCbkZvciBleGFtcGxlLCBmb3IgaW4tZ2FtZSBjdXJyZW5jeSwgeW91I"+
				"GNhbiBhbGxvdyBvbmx5IHRyYW5zYWN0aW9ucyBiZXR3ZWVuIGNoYXJhY3RlcnMgd2l0aCBjZXJ0YWluIHByb3BlcnRpZXMuBA"+
				"ZzdHI0NDQCa0EgdG9rZW4gd2l0aCBhbiBhc3NpZ25lZCBzY3JpcHQgaXMgY2FsbGVkIGEgc21hcnQgYXNzZXQsIGFuZCB0aGU"+
				"gYXNzaWduZWQgc2NyaXB0IGlzIGNhbGxlZCBhbiBhc3NldCBzY3JpcHQuBAd0ZXh0MTExCQC7CQIJAMwIAgUEc3RyMQkAzAgC"+
				"BQRzdHIyCQDMCAIFBHN0cjMJAMwIAgUEc3RyNAUDbmlsAgMgJiAEB3RleHQyMjIJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0c"+
				"jIJAMwIAgUEc3RyMwkAzAgCBQRzdHI0BQNuaWwCAyAmIAQHdGV4dDMzMwkAuwkCCQDMCAIFBHN0cjEJAMwIAgUEc3RyMgkAzA"+
				"gCBQRzdHIzCQDMCAIFBHN0cjQFA25pbAIDICYgBAd0ZXh0NDQ0CQC7CQIJAMwIAgUEc3RyMQkAzAgCBQRzdHIyCQDMCAIFBHN"+
				"0cjMJAMwIAgUEc3RyNAUDbmlsAgMgJiAEB3RleHQ1NTUJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkA"+
				"zAgCBQRzdHI0BQNuaWwCAyAmIAQHc3RyMTExMQI/VGhpcyB0ZXh0IGlzIG5lY2Vzc2FyeSB0byBnZXQgdGhlIHNjcmlwdCBvZ"+
				"iB0aGUgcmVxdWlyZWQgdm9sdW1lBAdzdHIyMjIyAoMBQW55IHVzZXIgY2FuIGNyZWF0ZSB0aGVpciBvd24gdG9rZW4gb24gdG"+
				"hlIFdhdmVzIGJsb2NrY2hhaW4gYW5kIGFsc28gc2V0IHRoZSBydWxlcyBmb3IgaXRzIGNpcmN1bGF0aW9uIGJ5IGFzc2lnbml"+
				"uZyBhIHNjcmlwdCB0byBpdC4EB3N0cjMzMzMCbkZvciBleGFtcGxlLCBmb3IgaW4tZ2FtZSBjdXJyZW5jeSwgeW91IGNhbiBh"+
				"bGxvdyBvbmx5IHRyYW5zYWN0aW9ucyBiZXR3ZWVuIGNoYXJhY3RlcnMgd2l0aCBjZXJ0YWluIHByb3BlcnRpZXMuBAdzdHI0N"+
				"DQ0AmtBIHRva2VuIHdpdGggYW4gYXNzaWduZWQgc2NyaXB0IGlzIGNhbGxlZCBhIHNtYXJ0IGFzc2V0LCBhbmQgdGhlIGFzc2"+
				"lnbmVkIHNjcmlwdCBpcyBjYWxsZWQgYW4gYXNzZXQgc2NyaXB0LgQIdGV4dDExMTEJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN"+
				"0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdHI0BQNuaWwCAyAmIAQIdGV4dDIyMjIJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJ"+
				"AMwIAgUEc3RyMwkAzAgCBQRzdHI0BQNuaWwCAyAmIAQIdGV4dDMzMzMJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIA"+
				"gUEc3RyMwkAzAgCBQRzdHI0BQNuaWwCAyAmIAQIdGV4dDQ0NDQJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3"+
				"RyMwkAzAgCBQRzdHI0BQNuaWwCAyAmIAQIdGV4dDU1NTUJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwk"+
				"AzAgCBQRzdHI0BQNuaWwCAyAmIAQFc3RycjECP1RoaXMgdGV4dCBpcyBuZWNlc3NhcnkgdG8gZ2V0IHRoZSBzY3JpcHQgb2Yg"+
				"dGhlIHJlcXVpcmVkIHZvbHVtZQQFc3RycjICgwFBbnkgdXNlciBjYW4gY3JlYXRlIHRoZWlyIG93biB0b2tlbiBvbiB0aGUgV"+
				"2F2ZXMgYmxvY2tjaGFpbiBhbmQgYWxzbyBzZXQgdGhlIHJ1bGVzIGZvciBpdHMgY2lyY3VsYXRpb24gYnkgYXNzaWduaW5nIG"+
				"Egc2NyaXB0IHRvIGl0LgQFc3RycjMCbkZvciBleGFtcGxlLCBmb3IgaW4tZ2FtZSBjdXJyZW5jeSwgeW91IGNhbiBhbGxvdyB"+
				"vbmx5IHRyYW5zYWN0aW9ucyBiZXR3ZWVuIGNoYXJhY3RlcnMgd2l0aCBjZXJ0YWluIHByb3BlcnRpZXMuBAVzdHJyNAJrQSB0"+
				"b2tlbiB3aXRoIGFuIGFzc2lnbmVkIHNjcmlwdCBpcyBjYWxsZWQgYSBzbWFydCBhc3NldCwgYW5kIHRoZSBhc3NpZ25lZCBzY"+
				"3JpcHQgaXMgY2FsbGVkIGFuIGFzc2V0IHNjcmlwdC4EBnRleHR0MQkAuwkCCQDMCAIFBHN0cjEJAMwIAgUEc3RyMgkAzAgCBQ"+
				"RzdHIzCQDMCAIFBHN0cjQFA25pbAIDICYgBAZ0ZXh0dDIJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwk"+
				"AzAgCBQRzdHI0BQNuaWwCAyAmIAQGdGV4dHQzCQC7CQIJAMwIAgUEc3RyMQkAzAgCBQRzdHIyCQDMCAIFBHN0cjMJAMwIAgUE"+
				"c3RyNAUDbmlsAgMgJiAEBnRleHR0NAkAuwkCCQDMCAIFBHN0cjEJAMwIAgUEc3RyMgkAzAgCBQRzdHIzCQDMCAIFBHN0cjQFA"+
				"25pbAIDICYgBAZ0ZXh0dDUJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdHI0BQNuaWwCAy"+
				"AmIAQGc3RycjExAj9UaGlzIHRleHQgaXMgbmVjZXNzYXJ5IHRvIGdldCB0aGUgc2NyaXB0IG9mIHRoZSByZXF1aXJlZCB2b2x"+
				"1bWUEBnN0cnIyMgKDAUFueSB1c2VyIGNhbiBjcmVhdGUgdGhlaXIgb3duIHRva2VuIG9uIHRoZSBXYXZlcyBibG9ja2NoYWlu"+
				"IGFuZCBhbHNvIHNldCB0aGUgcnVsZXMgZm9yIGl0cyBjaXJjdWxhdGlvbiBieSBhc3NpZ25pbmcgYSBzY3JpcHQgdG8gaXQuB"+
				"AZzdHJyMzMCbkZvciBleGFtcGxlLCBmb3IgaW4tZ2FtZSBjdXJyZW5jeSwgeW91IGNhbiBhbGxvdyBvbmx5IHRyYW5zYWN0aW"+
				"9ucyBiZXR3ZWVuIGNoYXJhY3RlcnMgd2l0aCBjZXJ0YWluIHByb3BlcnRpZXMuBAZzdHJyNDQCa0EgdG9rZW4gd2l0aCBhbiB"+
				"hc3NpZ25lZCBzY3JpcHQgaXMgY2FsbGVkIGEgc21hcnQgYXNzZXQsIGFuZCB0aGUgYXNzaWduZWQgc2NyaXB0IGlzIGNhbGxl"+
				"ZCBhbiBhc3NldCBzY3JpcHQuBAd0ZXh0dDExCQC7CQIJAMwIAgUEc3RyMQkAzAgCBQRzdHIyCQDMCAIFBHN0cjMJAMwIAgUEc"+
				"3RyNAUDbmlsAgMgJiAEB3RleHR0MjIJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdHI0BQ"+
				"NuaWwCAyAmIAQHdGV4dHQzMwkAuwkCCQDMCAIFBHN0cjEJAMwIAgUEc3RyMgkAzAgCBQRzdHIzCQDMCAIFBHN0cjQFA25pbAI"+
				"DICYgBAd0ZXh0dDQ0CQC7CQIJAMwIAgUEc3RyMQkAzAgCBQRzdHIyCQDMCAIFBHN0cjMJAMwIAgUEc3RyNAUDbmlsAgMgJiAE"+
				"B3RleHR0NTUJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdHI0BQNuaWwCAyAmIAQHc3Ryc"+
				"jExMQI/VGhpcyB0ZXh0IGlzIG5lY2Vzc2FyeSB0byBnZXQgdGhlIHNjcmlwdCBvZiB0aGUgcmVxdWlyZWQgdm9sdW1lBAdzdH"+
				"JyMjIyAoMBQW55IHVzZXIgY2FuIGNyZWF0ZSB0aGVpciBvd24gdG9rZW4gb24gdGhlIFdhdmVzIGJsb2NrY2hhaW4gYW5kIGF"+
				"sc28gc2V0IHRoZSBydWxlcyBmb3IgaXRzIGNpcmN1bGF0aW9uIGJ5IGFzc2lnbmluZyBhIHNjcmlwdCB0byBpdC4EB3N0cnIz"+
				"MzMCbkZvciBleGFtcGxlLCBmb3IgaW4tZ2FtZSBjdXJyZW5jeSwgeW91IGNhbiBhbGxvdyBvbmx5IHRyYW5zYWN0aW9ucyBiZ"+
				"XR3ZWVuIGNoYXJhY3RlcnMgd2l0aCBjZXJ0YWluIHByb3BlcnRpZXMuBAdzdHJyNDQ0AmtBIHRva2VuIHdpdGggYW4gYXNzaW"+
				"duZWQgc2NyaXB0IGlzIGNhbGxlZCBhIHNtYXJ0IGFzc2V0LCBhbmQgdGhlIGFzc2lnbmVkIHNjcmlwdCBpcyBjYWxsZWQgYW4"+
				"gYXNzZXQgc2NyaXB0LgQIdGV4dHQxMTEJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdHI0"+
				"BQNuaWwCAyAmIAQIdGV4dHQyMjIJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdHI0BQNua"+
				"WwCAyAmIAQIdGV4dHQzMzMJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdHI0BQNuaWwCAy"+
				"AmIAQIdGV4dHQ0NDQJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdHI0BQNuaWwCAyAmIAQ"+
				"IdGV4dHQ1NTUJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdHI0BQNuaWwCAyAmIAQIc3Ry"+
				"cjExMTECP1RoaXMgdGV4dCBpcyBuZWNlc3NhcnkgdG8gZ2V0IHRoZSBzY3JpcHQgb2YgdGhlIHJlcXVpcmVkIHZvbHVtZQQIc"+
				"3RycjIyMjICgwFBbnkgdXNlciBjYW4gY3JlYXRlIHRoZWlyIG93biB0b2tlbiBvbiB0aGUgV2F2ZXMgYmxvY2tjaGFpbiBhbm"+
				"QgYWxzbyBzZXQgdGhlIHJ1bGVzIGZvciBpdHMgY2lyY3VsYXRpb24gYnkgYXNzaWduaW5nIGEgc2NyaXB0IHRvIGl0LgQIc3R"+
				"ycjMzMzMCbkZvciBleGFtcGxlLCBmb3IgaW4tZ2FtZSBjdXJyZW5jeSwgeW91IGNhbiBhbGxvdyBvbmx5IHRyYW5zYWN0aW9u"+
				"cyBiZXR3ZWVuIGNoYXJhY3RlcnMgd2l0aCBjZXJ0YWluIHByb3BlcnRpZXMuBAhzdHJyNDQ0NAJrQSB0b2tlbiB3aXRoIGFuI"+
				"GFzc2lnbmVkIHNjcmlwdCBpcyBjYWxsZWQgYSBzbWFydCBhc3NldCwgYW5kIHRoZSBhc3NpZ25lZCBzY3JpcHQgaXMgY2FsbG"+
				"VkIGFuIGFzc2V0IHNjcmlwdC4ECXRleHR0MTExMQkAuwkCCQDMCAIFBHN0cjEJAMwIAgUEc3RyMgkAzAgCBQRzdHIzCQDMCAI"+
				"FBHN0cjQFA25pbAIDICYgBAl0ZXh0dDIyMjIJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRz"+
				"dHI0BQNuaWwCAyAmIAQJdGV4dHQzMzMzCQC7CQIJAMwIAgUEc3RyMQkAzAgCBQRzdHIyCQDMCAIFBHN0cjMJAMwIAgUEc3RyN"+
				"AUDbmlsAgMgJiAECXRleHR0NDQ0NAkAuwkCCQDMCAIFBHN0cjEJAMwIAgUEc3RyMgkAzAgCBQRzdHIzCQDMCAIFBHN0cjQFA2"+
				"5pbAIDICYgBAl0ZXh0dDU1NTUJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdHI0BQNuaWw"+
				"CAyAmIAQGYW1vdW50BAckbWF0Y2gwBQJ0eAMJAAECBQckbWF0Y2gwAhdJbnZva2VTY3JpcHRUcmFuc2FjdGlvbgQBaQUHJG1h"+
				"dGNoMAAKAwkAAQIFByRtYXRjaDACD0J1cm5UcmFuc2FjdGlvbgQBYgUHJG1hdGNoMAAAAwkAAQIFByRtYXRjaDACE1RyYW5zZ"+
				"mVyVHJhbnNhY3Rpb24EAXQFByRtYXRjaDAIBQF0BmFtb3VudAMJAAECBQckbWF0Y2gwAhdNYXNzVHJhbnNmZXJUcmFuc2FjdG"+
				"lvbgQBbQUHJG1hdGNoMAgFAW0LdG90YWxBbW91bnQAAAQHYW1vdW50dAQHJG1hdGNoMAUCdHgDCQABAgUHJG1hdGNoMAIXSW5"+
				"2b2tlU2NyaXB0VHJhbnNhY3Rpb24EAWkFByRtYXRjaDAACgMJAAECBQckbWF0Y2gwAg9CdXJuVHJhbnNhY3Rpb24EAWIFByRt"+
				"YXRjaDAAAAMJAAECBQckbWF0Y2gwAhNUcmFuc2ZlclRyYW5zYWN0aW9uBAF0BQckbWF0Y2gwCAUBdAZhbW91bnQDCQABAgUHJ"+
				"G1hdGNoMAIXTWFzc1RyYW5zZmVyVHJhbnNhY3Rpb24EAW0FByRtYXRjaDAIBQFtC3RvdGFsQW1vdW50AAAEByRtYXRjaDAFAn"+
				"R4AwkAAQIFByRtYXRjaDACF0ludm9rZVNjcmlwdFRyYW5zYWN0aW9uBAFpBQckbWF0Y2gwBgMJAAECBQckbWF0Y2gwAhJSZWl"+
				"zc3VlVHJhbnNhY3Rpb24EAXIFByRtYXRjaDAGAwkAAQIFByRtYXRjaDACD0J1cm5UcmFuc2FjdGlvbgQBYgUHJG1hdGNoMAYD"+
				"CQABAgUHJG1hdGNoMAITVHJhbnNmZXJUcmFuc2FjdGlvbgQBdAUHJG1hdGNoMAQBYQkApAgBCAUBdAlyZWNpcGllbnQDCQAAA"+
				"gUBYQUBYQMDAwMDAwMDAwMDAwMDAwMDAwMDAwkA9AMDAQABAAEACQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQD0AwMBAAEAAQ"+
				"AHCQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQD0AwMBAAEAAQA"+
				"HCQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAH"+
				"CQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQDIEwMBAAEAAQAHCQDIEwMBAAEAAQAHC"+
				"QACAQIkU3RyaWN0IHZhbHVlIGlzIG5vdCBlcXVhbCB0byBpdHNlbGYuBtTZQ5E="),
			utl.MinSetAssetScriptFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SetAssetScriptExpectedValuesPositive{
				WavesDiffBalance: utl.MinSetAssetScriptFeeWaves,
				AssetDiffBalance: 0,
			}),
		"Script with complexity 4000": *NewSetAssetScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.GetScriptBytes(suite, "BgEEByRtYXRjaDAFAnR4AwkAAQIFByRtYXRjaDACE1RyYW5zZmVyVHJhbnNhY3Rpb24E"+
				"AXQFByRtYXRjaDAEAWEJAKQIAQgFAXQJcmVjaXBpZW50AwkAAAIFAWEFAWEDAwMDAwMDAwMDAwMDAwMDAwMDAwMDCQD0AwMBA"+
				"AEAAQAJAPQDAwEAAQABAAcJAPQDAwEAAQABAAcJAPQDAwEAAQABAAcJAPQDAwEAAQABAAcJAPQDAwEAAQABAAcJAPQDAwEAAQ"+
				"ABAAcJAPQDAwEAAQABAAcJAPQDAwEAAQABAAcJAPQDAwEAAQABAAcJAPQDAwEAAQABAAcJAPQDAwEAAQABAAcJAPQDAwEAAQA"+
				"BAAcJAPQDAwEAAQABAAcJAPQDAwEAAQABAAcJAPQDAwEAAQABAAcJAPQDAwEAAQABAAcJAPQDAwEAAQABAAcJAPQDAwEAAQAB"+
				"AAcJAPQDAwEAAQABAAcJAMgTAwEAAQABAAcJAMgTAwEAAQABAAcJAMcTAwEAAQABAAcJAAIBAiRTdHJpY3QgdmFsdWUgaXMgb"+
				"m90IGVxdWFsIHRvIGl0c2VsZi4Gh1kbVQ=="),
			utl.MinSetAssetScriptFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SetAssetScriptExpectedValuesPositive{
				WavesDiffBalance: utl.MinSetAssetScriptFeeWaves,
				AssetDiffBalance: 0,
			}),
	}
}

func GetSetAssetScriptNegativeData(suite *f.BaseSuite, assetID crypto.Digest) map[string]SetAssetScriptTestData[SetAssetScriptExpectedValuesNegative] {
	return map[string]SetAssetScriptTestData[SetAssetScriptExpectedValuesNegative]{
		"Complexity more than 4000": *NewSetAssetScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.GetScriptBytes(suite, "BgEEByRtYXRjaDAFAnR4AwkAAQIFByRtYXRjaDACE1RyYW5zZmVyVHJhbnNhY3Rpb24EAXQ"+
				"FByRtYXRjaDAEAWEJAKQIAQgFAXQJcmVjaXBpZW50AwkAAAIFAWEFAWEDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwkA9AMDAQABAA"+
				"EACQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAH"+
				"CQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQD0"+
				"AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQD0AwMB"+
				"AAEAAQAHCQDIEwMBAAEAAQAHCQDIEwMBAAEAAQAHCQDHEwMBAAEAAQAHCQDHEwMBAAEAAQAHCQACAQIkU3RyaWN0IHZhbHVlIGlz"+
				"IG5vdCBlcXVhbCB0byBpdHNlbGYuBgANpkU="),
			utl.MinSetAssetScriptFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SetAssetScriptExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Script is too complex",
			}),
		"Illegal length of script": *NewSetAssetScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.GetScriptBytes(suite, "AA=="),
			utl.MinSetAssetScriptFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SetAssetScriptExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Illegal length of script",
			}),
		//Should it be passed?
		/*"Empty script bytes": *NewSetAssetScriptTestData(
		utl.GetAccount(suite, utl.DefaultSenderNotMiner),
		assetID,
		utl.GetScriptBytes(suite, ""),
		utl.MinSetAssetScriptFeeWaves,
		utl.GetCurrentTimestampInMs(),
		utl.TestChainID,
		SetAssetScriptExpectedValuesNegative{
			WavesDiffBalance:  0,
			AssetDiffBalance:  0,
			ErrGoMsg:          errMsg,
			ErrScalaMsg:       errMsg,
			ErrBrdCstGoMsg:    errBrdCstMsg,
			ErrBrdCstScalaMsg: "",
		}),*/
		//TODO Wait fix for Scala Node
		/*"Script size more 8192 bytes": *NewSetAssetScriptTestData(
		utl.GetAccount(suite, utl.DefaultSenderNotMiner),
		assetID,
		utl.GetScriptBytes(suite, "BgEEA3N0cgI6VGhpcyB0ZXh0IGlzIG5lY2Vzc2FyeSB0byBnZXQgdGhlIHNjcmlwdCBvZ"+
			"iB0aGUgODE5MiBieXRlcwQFc3RyXzECP1RoaXMgdGV4dCBpcyBuZWNlc3NhcnkgdG8gZ2V0IHRoZSBzY3JpcHQgb2YgdGhlIHJ"+
			"lcXVpcmVkIHZvbHVtZQQFc3RyXzICgwFBbnkgdXNlciBjYW4gY3JlYXRlIHRoZWlyIG93biB0b2tlbiBvbiB0aGUgV2F2ZXMgY"+
			"mxvY2tjaGFpbiBhbmQgYWxzbyBzZXQgdGhlIHJ1bGVzIGZvciBpdHMgY2lyY3VsYXRpb24gYnkgYXNzaWduaW5nIGEgc2Nya"+
			"XB0IHRvIGl0LgQFc3RyXzMCbkZvciBleGFtcGxlLCBmb3IgaW4tZ2FtZSBjdXJyZW5jeSwgeW91IGNhbiBhbGxvdyBvbmx5IHR"+
			"yYW5zYWN0aW9ucyBiZXR3ZWVuIGNoYXJhY3RlcnMgd2l0aCBjZXJ0YWluIHByb3BlcnRpZXMuBAVzdHJfNAJrQSB0b2tlbiB3a"+
			"XRoIGFuIGFzc2lnbmVkIHNjcmlwdCBpcyBjYWxsZWQgYSBzbWFydCBhc3NldCwgYW5kIHRoZSBhc3NpZ25lZCBzY3JpcHQgaXM"+
			"gY2FsbGVkIGFuIGFzc2V0IHNjcmlwdC4EBnRleHRfMQkAuwkCCQDMCAIFBXN0cl8xCQDMCAIFBXN0cl8yCQDMCAIFBXN0cl8zC"+
			"QDMCAIFBXN0cl80BQNuaWwCAyAmIAQGdGV4dF8yCQC7CQIJAMwIAgUFc3RyXzEJAMwIAgUFc3RyXzIJAMwIAgUFc3RyXzMJAMw"+
			"IAgUFc3RyXzQFA25pbAIDICYgBAZ0ZXh0XzMJALsJAgkAzAgCBQVzdHJfMQkAzAgCBQVzdHJfMgkAzAgCBQVzdHJfMwkAzAgCB"+
			"QVzdHJfNAUDbmlsAgMgJiAEBnRleHRfNAkAuwkCCQDMCAIFBXN0cl8xCQDMCAIFBXN0cl8yCQDMCAIFBXN0cl8zCQDMCAIFBXN"+
			"0cl80BQNuaWwCAyAmIAQEc3RyMQI/VGhpcyB0ZXh0IGlzIG5lY2Vzc2FyeSB0byBnZXQgdGhlIHNjcmlwdCBvZiB0aGUgcmVxd"+
			"WlyZWQgdm9sdW1lBARzdHIyAoMBQW55IHVzZXIgY2FuIGNyZWF0ZSB0aGVpciBvd24gdG9rZW4gb24gdGhlIFdhdmVzIGJsb2N"+
			"rY2hhaW4gYW5kIGFsc28gc2V0IHRoZSBydWxlcyBmb3IgaXRzIGNpcmN1bGF0aW9uIGJ5IGFzc2lnbmluZyBhIHNjcmlwdCB0b"+
			"yBpdC4EBHN0cjMCbkZvciBleGFtcGxlLCBmb3IgaW4tZ2FtZSBjdXJyZW5jeSwgeW91IGNhbiBhbGxvdyBvbmx5IHRyYW5zYWN"+
			"0aW9ucyBiZXR3ZWVuIGNoYXJhY3RlcnMgd2l0aCBjZXJ0YWluIHByb3BlcnRpZXMuBARzdHI0AmtBIHRva2VuIHdpdGggYW4gY"+
			"XNzaWduZWQgc2NyaXB0IGlzIGNhbGxlZCBhIHNtYXJ0IGFzc2V0LCBhbmQgdGhlIGFzc2lnbmVkIHNjcmlwdCBpcyBjYWxsZWQ"+
			"gYW4gYXNzZXQgc2NyaXB0LgQFdGV4dDEJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdHI0B"+
			"QNuaWwCAyAmIAQFdGV4dDIJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdHI0BQNuaWwCAyA"+
			"mIAQFdGV4dDMJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdHI0BQNuaWwCAyAmIAQFdGV4d"+
			"DQJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdHI0BQNuaWwCAyAmIAQFdGV4dDUJALsJAgk"+
			"AzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdHI0BQNuaWwCAyAmIAQFc3RyMTECP1RoaXMgdGV4dCBpc"+
			"yBuZWNlc3NhcnkgdG8gZ2V0IHRoZSBzY3JpcHQgb2YgdGhlIHJlcXVpcmVkIHZvbHVtZQQFc3RyMjICgwFBbnkgdXNlciBjYW4"+
			"gY3JlYXRlIHRoZWlyIG93biB0b2tlbiBvbiB0aGUgV2F2ZXMgYmxvY2tjaGFpbiBhbmQgYWxzbyBzZXQgdGhlIHJ1bGVzIGZvc"+
			"iBpdHMgY2lyY3VsYXRpb24gYnkgYXNzaWduaW5nIGEgc2NyaXB0IHRvIGl0LgQFc3RyMzMCbkZvciBleGFtcGxlLCBmb3IgaW4"+
			"tZ2FtZSBjdXJyZW5jeSwgeW91IGNhbiBhbGxvdyBvbmx5IHRyYW5zYWN0aW9ucyBiZXR3ZWVuIGNoYXJhY3RlcnMgd2l0aCBjZ"+
			"XJ0YWluIHByb3BlcnRpZXMuBAVzdHI0NAJrQSB0b2tlbiB3aXRoIGFuIGFzc2lnbmVkIHNjcmlwdCBpcyBjYWxsZWQgYSBzbWF"+
			"ydCBhc3NldCwgYW5kIHRoZSBhc3NpZ25lZCBzY3JpcHQgaXMgY2FsbGVkIGFuIGFzc2V0IHNjcmlwdC4EBnRleHQxMQkAuwkCC"+
			"QDMCAIFBHN0cjEJAMwIAgUEc3RyMgkAzAgCBQRzdHIzCQDMCAIFBHN0cjQFA25pbAIDICYgBAZ0ZXh0MjIJALsJAgkAzAgCBQR"+
			"zdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdHI0BQNuaWwCAyAmIAQGdGV4dDMzCQC7CQIJAMwIAgUEc3RyMQkAz"+
			"AgCBQRzdHIyCQDMCAIFBHN0cjMJAMwIAgUEc3RyNAUDbmlsAgMgJiAEBnRleHQ0NAkAuwkCCQDMCAIFBHN0cjEJAMwIAgUEc3R"+
			"yMgkAzAgCBQRzdHIzCQDMCAIFBHN0cjQFA25pbAIDICYgBAZ0ZXh0NTUJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIA"+
			"gUEc3RyMwkAzAgCBQRzdHI0BQNuaWwCAyAmIAQGc3RyMTExAj9UaGlzIHRleHQgaXMgbmVjZXNzYXJ5IHRvIGdldCB0aGUgc2N"+
			"yaXB0IG9mIHRoZSByZXF1aXJlZCB2b2x1bWUEBnN0cjIyMgKDAUFueSB1c2VyIGNhbiBjcmVhdGUgdGhlaXIgb3duIHRva2VuI"+
			"G9uIHRoZSBXYXZlcyBibG9ja2NoYWluIGFuZCBhbHNvIHNldCB0aGUgcnVsZXMgZm9yIGl0cyBjaXJjdWxhdGlvbiBieSBhc3N"+
			"pZ25pbmcgYSBzY3JpcHQgdG8gaXQuBAZzdHIzMzMCbkZvciBleGFtcGxlLCBmb3IgaW4tZ2FtZSBjdXJyZW5jeSwgeW91IGNhb"+
			"iBhbGxvdyBvbmx5IHRyYW5zYWN0aW9ucyBiZXR3ZWVuIGNoYXJhY3RlcnMgd2l0aCBjZXJ0YWluIHByb3BlcnRpZXMuBAZzdHI"+
			"0NDQCa0EgdG9rZW4gd2l0aCBhbiBhc3NpZ25lZCBzY3JpcHQgaXMgY2FsbGVkIGEgc21hcnQgYXNzZXQsIGFuZCB0aGUgYXNza"+
			"WduZWQgc2NyaXB0IGlzIGNhbGxlZCBhbiBhc3NldCBzY3JpcHQuBAd0ZXh0MTExCQC7CQIJAMwIAgUEc3RyMQkAzAgCBQRzdHI"+
			"yCQDMCAIFBHN0cjMJAMwIAgUEc3RyNAUDbmlsAgMgJiAEB3RleHQyMjIJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIA"+
			"gUEc3RyMwkAzAgCBQRzdHI0BQNuaWwCAyAmIAQHdGV4dDMzMwkAuwkCCQDMCAIFBHN0cjEJAMwIAgUEc3RyMgkAzAgCBQRzdHI"+
			"zCQDMCAIFBHN0cjQFA25pbAIDICYgBAd0ZXh0NDQ0CQC7CQIJAMwIAgUEc3RyMQkAzAgCBQRzdHIyCQDMCAIFBHN0cjMJAMwIA"+
			"gUEc3RyNAUDbmlsAgMgJiAEB3RleHQ1NTUJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdHI"+
			"0BQNuaWwCAyAmIAQHc3RyMTExMQI/VGhpcyB0ZXh0IGlzIG5lY2Vzc2FyeSB0byBnZXQgdGhlIHNjcmlwdCBvZiB0aGUgcmVxd"+
			"WlyZWQgdm9sdW1lBAdzdHIyMjIyAoMBQW55IHVzZXIgY2FuIGNyZWF0ZSB0aGVpciBvd24gdG9rZW4gb24gdGhlIFdhdmVzIGJ"+
			"sb2NrY2hhaW4gYW5kIGFsc28gc2V0IHRoZSBydWxlcyBmb3IgaXRzIGNpcmN1bGF0aW9uIGJ5IGFzc2lnbmluZyBhIHNjcmlwd"+
			"CB0byBpdC4EB3N0cjMzMzMCbkZvciBleGFtcGxlLCBmb3IgaW4tZ2FtZSBjdXJyZW5jeSwgeW91IGNhbiBhbGxvdyBvbmx5IHR"+
			"yYW5zYWN0aW9ucyBiZXR3ZWVuIGNoYXJhY3RlcnMgd2l0aCBjZXJ0YWluIHByb3BlcnRpZXMuBAdzdHI0NDQ0AmtBIHRva2VuI"+
			"HdpdGggYW4gYXNzaWduZWQgc2NyaXB0IGlzIGNhbGxlZCBhIHNtYXJ0IGFzc2V0LCBhbmQgdGhlIGFzc2lnbmVkIHNjcmlwdCB"+
			"pcyBjYWxsZWQgYW4gYXNzZXQgc2NyaXB0LgQIdGV4dDExMTEJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyM"+
			"wkAzAgCBQRzdHI0BQNuaWwCAyAmIAQIdGV4dDIyMjIJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAg"+
			"CBQRzdHI0BQNuaWwCAyAmIAQIdGV4dDMzMzMJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzd"+
			"HI0BQNuaWwCAyAmIAQIdGV4dDQ0NDQJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdHI0BQN"+
			"uaWwCAyAmIAQIdGV4dDU1NTUJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdHI0BQNuaWwCA"+
			"yAmIAQFc3RycjECP1RoaXMgdGV4dCBpcyBuZWNlc3NhcnkgdG8gZ2V0IHRoZSBzY3JpcHQgb2YgdGhlIHJlcXVpcmVkIHZvbHV"+
			"tZQQFc3RycjICgwFBbnkgdXNlciBjYW4gY3JlYXRlIHRoZWlyIG93biB0b2tlbiBvbiB0aGUgV2F2ZXMgYmxvY2tjaGFpbiBhb"+
			"mQgYWxzbyBzZXQgdGhlIHJ1bGVzIGZvciBpdHMgY2lyY3VsYXRpb24gYnkgYXNzaWduaW5nIGEgc2NyaXB0IHRvIGl0LgQFc3R"+
			"ycjMCbkZvciBleGFtcGxlLCBmb3IgaW4tZ2FtZSBjdXJyZW5jeSwgeW91IGNhbiBhbGxvdyBvbmx5IHRyYW5zYWN0aW9ucyBiZ"+
			"XR3ZWVuIGNoYXJhY3RlcnMgd2l0aCBjZXJ0YWluIHByb3BlcnRpZXMuBAVzdHJyNAJrQSB0b2tlbiB3aXRoIGFuIGFzc2lnbmV"+
			"kIHNjcmlwdCBpcyBjYWxsZWQgYSBzbWFydCBhc3NldCwgYW5kIHRoZSBhc3NpZ25lZCBzY3JpcHQgaXMgY2FsbGVkIGFuIGFzc"+
			"2V0IHNjcmlwdC4EBnRleHR0MQkAuwkCCQDMCAIFBHN0cjEJAMwIAgUEc3RyMgkAzAgCBQRzdHIzCQDMCAIFBHN0cjQFA25pbAI"+
			"DICYgBAZ0ZXh0dDIJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdHI0BQNuaWwCAyAmIAQGd"+
			"GV4dHQzCQC7CQIJAMwIAgUEc3RyMQkAzAgCBQRzdHIyCQDMCAIFBHN0cjMJAMwIAgUEc3RyNAUDbmlsAgMgJiAEBnRleHR0NAk"+
			"AuwkCCQDMCAIFBHN0cjEJAMwIAgUEc3RyMgkAzAgCBQRzdHIzCQDMCAIFBHN0cjQFA25pbAIDICYgBAZ0ZXh0dDUJALsJAgkA"+
			"zAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdHI0BQNuaWwCAyAmIAQGc3RycjExAj9UaGlzIHRleHQga"+
			"XMgbmVjZXNzYXJ5IHRvIGdldCB0aGUgc2NyaXB0IG9mIHRoZSByZXF1aXJlZCB2b2x1bWUEBnN0cnIyMgKDAUFueSB1c2VyIG"+
			"NhbiBjcmVhdGUgdGhlaXIgb3duIHRva2VuIG9uIHRoZSBXYXZlcyBibG9ja2NoYWluIGFuZCBhbHNvIHNldCB0aGUgcnVsZXM"+
			"gZm9yIGl0cyBjaXJjdWxhdGlvbiBieSBhc3NpZ25pbmcgYSBzY3JpcHQgdG8gaXQuBAZzdHJyMzMCbkZvciBleGFtcGxlLCBm"+
			"b3IgaW4tZ2FtZSBjdXJyZW5jeSwgeW91IGNhbiBhbGxvdyBvbmx5IHRyYW5zYWN0aW9ucyBiZXR3ZWVuIGNoYXJhY3RlcnMgd"+
			"2l0aCBjZXJ0YWluIHByb3BlcnRpZXMuBAZzdHJyNDQCa0EgdG9rZW4gd2l0aCBhbiBhc3NpZ25lZCBzY3JpcHQgaXMgY2FsbG"+
			"VkIGEgc21hcnQgYXNzZXQsIGFuZCB0aGUgYXNzaWduZWQgc2NyaXB0IGlzIGNhbGxlZCBhbiBhc3NldCBzY3JpcHQuBAd0ZXh"+
			"0dDExCQC7CQIJAMwIAgUEc3RyMQkAzAgCBQRzdHIyCQDMCAIFBHN0cjMJAMwIAgUEc3RyNAUDbmlsAgMgJiAEB3RleHR0MjIJ"+
			"ALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdHI0BQNuaWwCAyAmIAQHdGV4dHQzMwkAuwkCC"+
			"QDMCAIFBHN0cjEJAMwIAgUEc3RyMgkAzAgCBQRzdHIzCQDMCAIFBHN0cjQFA25pbAIDICYgBAd0ZXh0dDQ0CQC7CQIJAMwIAg"+
			"UEc3RyMQkAzAgCBQRzdHIyCQDMCAIFBHN0cjMJAMwIAgUEc3RyNAUDbmlsAgMgJiAEB3RleHR0NTUJALsJAgkAzAgCBQRzdHI"+
			"xCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdHI0BQNuaWwCAyAmIAQHc3RycjExMQI/VGhpcyB0ZXh0IGlzIG5lY2Vz"+
			"c2FyeSB0byBnZXQgdGhlIHNjcmlwdCBvZiB0aGUgcmVxdWlyZWQgdm9sdW1lBAdzdHJyMjIyAoMBQW55IHVzZXIgY2FuIGNyZ"+
			"WF0ZSB0aGVpciBvd24gdG9rZW4gb24gdGhlIFdhdmVzIGJsb2NrY2hhaW4gYW5kIGFsc28gc2V0IHRoZSBydWxlcyBmb3IgaX"+
			"RzIGNpcmN1bGF0aW9uIGJ5IGFzc2lnbmluZyBhIHNjcmlwdCB0byBpdC4EB3N0cnIzMzMCbkZvciBleGFtcGxlLCBmb3IgaW4"+
			"tZ2FtZSBjdXJyZW5jeSwgeW91IGNhbiBhbGxvdyBvbmx5IHRyYW5zYWN0aW9ucyBiZXR3ZWVuIGNoYXJhY3RlcnMgd2l0aCBj"+
			"ZXJ0YWluIHByb3BlcnRpZXMuBAdzdHJyNDQ0AmtBIHRva2VuIHdpdGggYW4gYXNzaWduZWQgc2NyaXB0IGlzIGNhbGxlZCBhI"+
			"HNtYXJ0IGFzc2V0LCBhbmQgdGhlIGFzc2lnbmVkIHNjcmlwdCBpcyBjYWxsZWQgYW4gYXNzZXQgc2NyaXB0LgQIdGV4dHQxMT"+
			"EJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdHI0BQNuaWwCAyAmIAQIdGV4dHQyMjIJALs"+
			"JAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdHI0BQNuaWwCAyAmIAQIdGV4dHQzMzMJALsJAgkA"+
			"zAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdHI0BQNuaWwCAyAmIAQIdGV4dHQ0NDQJALsJAgkAzAgCB"+
			"QRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdHI0BQNuaWwCAyAmIAQIdGV4dHQ1NTUJALsJAgkAzAgCBQRzdH"+
			"IxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdHI0BQNuaWwCAyAmIAQIc3RycjExMTECP1RoaXMgdGV4dCBpcyBuZ"+
			"WNlc3NhcnkgdG8gZ2V0IHRoZSBzY3JpcHQgb2YgdGhlIHJlcXVpcmVkIHZvbHVtZQQIc3RycjIyMjICgwFBbnkgdXNlciBjYW"+
			"4gY3JlYXRlIHRoZWlyIG93biB0b2tlbiBvbiB0aGUgV2F2ZXMgYmxvY2tjaGFpbiBhbmQgYWxzbyBzZXQgdGhlIHJ1bGVzIGZv"+
			"ciBpdHMgY2lyY3VsYXRpb24gYnkgYXNzaWduaW5nIGEgc2NyaXB0IHRvIGl0LgQIc3RycjMzMzMCbkZvciBleGFtcGxlLCBmb3"+
			"IgaW4tZ2FtZSBjdXJyZW5jeSwgeW91IGNhbiBhbGxvdyBvbmx5IHRyYW5zYWN0aW9ucyBiZXR3ZWVuIGNoYXJhY3RlcnMgd2l0"+
			"aCBjZXJ0YWluIHByb3BlcnRpZXMuBAhzdHJyNDQ0NAJrQSB0b2tlbiB3aXRoIGFuIGFzc2lnbmVkIHNjcmlwdCBpcyBjYWxsZW"+
			"QgYSBzbWFydCBhc3NldCwgYW5kIHRoZSBhc3NpZ25lZCBzY3JpcHQgaXMgY2FsbGVkIGFuIGFzc2V0IHNjcmlwdC4ECXRleHR0"+
			"MTExMQkAuwkCCQDMCAIFBHN0cjEJAMwIAgUEc3RyMgkAzAgCBQRzdHIzCQDMCAIFBHN0cjQFA25pbAIDICYgBAl0ZXh0dDIyMj"+
			"IJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdHI0BQNuaWwCAyAmIAQJdGV4dHQzMzMzCQC7"+
			"CQIJAMwIAgUEc3RyMQkAzAgCBQRzdHIyCQDMCAIFBHN0cjMJAMwIAgUEc3RyNAUDbmlsAgMgJiAECXRleHR0NDQ0NAkAuwkCCQ"+
			"DMCAIFBHN0cjEJAMwIAgUEc3RyMgkAzAgCBQRzdHIzCQDMCAIFBHN0cjQFA25pbAIDICYgBAl0ZXh0dDU1NTUJALsJAgkAzAgC"+
			"BQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdHI0BQNuaWwCAyAmIAQGYW1vdW50BAckbWF0Y2gwBQJ0eAMJAA"+
			"ECBQckbWF0Y2gwAhdJbnZva2VTY3JpcHRUcmFuc2FjdGlvbgQBaQUHJG1hdGNoMAAKAwkAAQIFByRtYXRjaDACD0J1cm5UcmFu"+
			"c2FjdGlvbgQBYgUHJG1hdGNoMAAAAwkAAQIFByRtYXRjaDACE1RyYW5zZmVyVHJhbnNhY3Rpb24EAXQFByRtYXRjaDAIBQF0Bm"+
			"Ftb3VudAMJAAECBQckbWF0Y2gwAhdNYXNzVHJhbnNmZXJUcmFuc2FjdGlvbgQBbQUHJG1hdGNoMAgFAW0LdG90YWxBbW91bnQA"+
			"AAQHYW1vdW50dAQHJG1hdGNoMAUCdHgDCQABAgUHJG1hdGNoMAIXSW52b2tlU2NyaXB0VHJhbnNhY3Rpb24EAWkFByRtYXRjaD"+
			"AACgMJAAECBQckbWF0Y2gwAg9CdXJuVHJhbnNhY3Rpb24EAWIFByRtYXRjaDAAAAMJAAECBQckbWF0Y2gwAhNUcmFuc2ZlclRy"+
			"YW5zYWN0aW9uBAF0BQckbWF0Y2gwCAUBdAZhbW91bnQDCQABAgUHJG1hdGNoMAIXTWFzc1RyYW5zZmVyVHJhbnNhY3Rpb24EAW"+
			"0FByRtYXRjaDAIBQFtC3RvdGFsQW1vdW50AAAEByRtYXRjaDAFAnR4AwkAAQIFByRtYXRjaDACF0ludm9rZVNjcmlwdFRyYW5z"+
			"YWN0aW9uBAFpBQckbWF0Y2gwBgMJAAECBQckbWF0Y2gwAhJSZWlzc3VlVHJhbnNhY3Rpb24EAXIFByRtYXRjaDAGAwkAAQIFBy"+
			"RtYXRjaDACD0J1cm5UcmFuc2FjdGlvbgQBYgUHJG1hdGNoMAYDCQABAgUHJG1hdGNoMAITVHJhbnNmZXJUcmFuc2FjdGlvbgQB"+
			"dAUHJG1hdGNoMAQBYQkApAgBCAUBdAlyZWNpcGllbnQDCQAAAgUBYQUBYQMDAwMDAwMDAwMDAwMDAwMDAwMDAwkA9AMDAQABAA"+
			"EACQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAH"+
			"CQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQ"+
			"D0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQD0AwMBAAEAAQAHCQD0"+
			"AwMBAAEAAQAHCQDIEwMBAAEAAQAHCQDIEwMBAAEAAQAHCQACAQIkU3RyaWN0IHZhbHVlIGlzIG5vdCBlcXVhbCB0byBpdHNlbG"+
			"YuBuFg8mQ="),
		utl.MinSetAssetScriptFeeWaves,
		utl.GetCurrentTimestampInMs(),
		utl.TestChainID,
		SetAssetScriptExpectedValuesNegative{
			WavesDiffBalance:  0,
			AssetDiffBalance:  0,
			ErrGoMsg:          errMsg,
			ErrScalaMsg:       errMsg,
			ErrBrdCstGoMsg:    errBrdCstMsg,
			ErrBrdCstScalaMsg: "",
		}),*/
		"Invalid content type of script": *NewSetAssetScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.GetScriptBytes(suite, "AAQB"),
			utl.MinSetAssetScriptFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SetAssetScriptExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Invalid content type of script",
			}),
		"Invalid script version": *NewSetAssetScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.GetScriptBytes(suite, "CAEF"),
			utl.MinSetAssetScriptFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SetAssetScriptExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Invalid version of script",
			}),
		"Asset was issued by other Account": *NewSetAssetScriptTestData(
			utl.GetAccount(suite, utl.DefaultRecipientNotMiner),
			assetID,
			utl.GetScriptBytes(suite, "BQbtKNoM"),
			utl.MinSetAssetScriptFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SetAssetScriptExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Asset was issued by other address",
			}),
		"Invalid fee (fee > max)": *NewSetAssetScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.GetScriptBytes(suite, "BQbtKNoM"),
			utl.MaxAmount+1,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SetAssetScriptExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "failed to parse json message",
			}),
		"Invalid fee (0 < fee < min)": *NewSetAssetScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.GetScriptBytes(suite, "BQbtKNoM"),
			10,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SetAssetScriptExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "(10 in WAVES) does not exceed minimal value",
			}),
		"Invalid fee (fee = 0)": *NewSetAssetScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.GetScriptBytes(suite, "BQbtKNoM"),
			0,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SetAssetScriptExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "insufficient fee",
			}),
		"Timestamp more than 7200000ms in the past relative to previous block timestamp": *NewSetAssetScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.GetScriptBytes(suite, "BQbtKNoM"),
			utl.MinSetAssetScriptFeeWaves,
			utl.GetCurrentTimestampInMs()-7260000,
			utl.TestChainID,
			SetAssetScriptExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "is more than 7200000ms in the past relative to previous block timestamp",
			}),
		"Timestamp more than 5400000ms in the future relative to previous block timestamp": *NewSetAssetScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.GetScriptBytes(suite, "BQbtKNoM"),
			utl.MinSetAssetScriptFeeWaves,
			utl.GetCurrentTimestampInMs()+54160000,
			utl.TestChainID,
			SetAssetScriptExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "is more than 5400000ms in the future relative to block timestamp",
			}),
		"Try to do sponsorship when fee more than funds on the sender balance": *NewSetAssetScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.GetScriptBytes(suite, "BQbtKNoM"),
			uint64(100000000+utl.GetAvailableBalanceInWavesGo(suite, utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address)),
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SetAssetScriptExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "negative waves balance",
			}),
		"Invalid asset ID (asset ID not exist)": *NewSetAssetScriptTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandDigest(suite.T(), 32, utl.LettersAndDigits),
			utl.GetScriptBytes(suite, "BQbtKNoM"),
			utl.MinSetAssetScriptFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SetAssetScriptExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Referenced assetId not found",
			}),
	}
}

func GetSimpleSmartAssetNegativeData(suite *f.BaseSuite, assetID crypto.Digest) SetAssetScriptTestData[SetAssetScriptExpectedValuesNegative] {
	return *NewSetAssetScriptTestData(
		utl.GetAccount(suite, utl.DefaultSenderNotMiner),
		assetID,
		utl.GetScriptBytes(suite, "BQbtKNoM"),
		utl.MinSetAssetScriptFeeWaves,
		utl.GetCurrentTimestampInMs(),
		utl.TestChainID,
		SetAssetScriptExpectedValuesNegative{
			WavesDiffBalance:  0,
			AssetDiffBalance:  0,
			ErrGoMsg:          errMsg,
			ErrScalaMsg:       errMsg,
			ErrBrdCstGoMsg:    errBrdCstMsg,
			ErrBrdCstScalaMsg: "",
		})
}
