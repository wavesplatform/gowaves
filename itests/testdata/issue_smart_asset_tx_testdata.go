package testdata

import (
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
)

func GetPositiveAssetScriptData(suite *f.BaseSuite) map[string]IssueTestData[ExpectedValuesPositive] {
	return map[string]IssueTestData[ExpectedValuesPositive]{
		"Valid script, true as expression": *NewIssueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(5, utl.CommonSymbolSet),
			utl.RandStringBytes(20, utl.CommonSymbolSet),
			100000000000,
			8,
			true,
			utl.GetScriptBytes(suite, "BQbtKNoM"),
			utl.MinIssueFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			ExpectedValuesPositive{
				WavesDiffBalance: utl.MinIssueFeeWaves,
				AssetBalance:     100000000000,
			}),
		"Valid script, size 8192 bytes": *NewIssueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(5, utl.CommonSymbolSet),
			utl.RandStringBytes(20, utl.CommonSymbolSet),
			100000000000,
			8,
			true,
			utl.GetScriptBytes(suite, "BgEEA3N0cgI3VGhpcyB0ZXh0IGlzIG5lY2Vzc2FyeSB0byBnZXQgdGhlIHNjcmlwdCBvZiB0aGUgODE5MiBieQQFc3RyXz"+
				"ECP1RoaXMgdGV4dCBpcyBuZWNlc3NhcnkgdG8gZ2V0IHRoZSBzY3JpcHQgb2YgdGhlIHJlcXVpcmVkIHZvbHVtZQQFc3RyXzI"+
				"CgwFBbnkgdXNlciBjYW4gY3JlYXRlIHRoZWlyIG93biB0b2tlbiBvbiB0aGUgV2F2ZXMgYmxvY2tjaGFpbiBhbmQgYWxzbyBz"+
				"ZXQgdGhlIHJ1bGVzIGZvciBpdHMgY2lyY3VsYXRpb24gYnkgYXNzaWduaW5nIGEgc2NyaXB0IHRvIGl0LgQFc3RyXzMCbkZvc"+
				"iBleGFtcGxlLCBmb3IgaW4tZ2FtZSBjdXJyZW5jeSwgeW91IGNhbiBhbGxvdyBvbmx5IHRyYW5zYWN0aW9ucyBiZXR3ZWVuIG"+
				"NoYXJhY3RlcnMgd2l0aCBjZXJ0YWluIHByb3BlcnRpZXMuBAVzdHJfNAJrQSB0b2tlbiB3aXRoIGFuIGFzc2lnbmVkIHNjcml"+
				"wdCBpcyBjYWxsZWQgYSBzbWFydCBhc3NldCwgYW5kIHRoZSBhc3NpZ25lZCBzY3JpcHQgaXMgY2FsbGVkIGFuIGFzc2V0IHNj"+
				"cmlwdC4EBnRleHRfMQkAuwkCCQDMCAIFBXN0cl8xCQDMCAIFBXN0cl8yCQDMCAIFBXN0cl8zCQDMCAIFBXN0cl80BQNuaWwCA"+
				"yAmIAQGdGV4dF8yCQC7CQIJAMwIAgUFc3RyXzEJAMwIAgUFc3RyXzIJAMwIAgUFc3RyXzMJAMwIAgUFc3RyXzQFA25pbAIDIC"+
				"YgBAZ0ZXh0XzMJALsJAgkAzAgCBQVzdHJfMQkAzAgCBQVzdHJfMgkAzAgCBQVzdHJfMwkAzAgCBQVzdHJfNAUDbmlsAgMgJiA"+
				"EBnRleHRfNAkAuwkCCQDMCAIFBXN0cl8xCQDMCAIFBXN0cl8yCQDMCAIFBXN0cl8zCQDMCAIFBXN0cl80BQNuaWwCAyAmIAQE"+
				"c3RyMQI/VGhpcyB0ZXh0IGlzIG5lY2Vzc2FyeSB0byBnZXQgdGhlIHNjcmlwdCBvZiB0aGUgcmVxdWlyZWQgdm9sdW1lBARzd"+
				"HIyAoMBQW55IHVzZXIgY2FuIGNyZWF0ZSB0aGVpciBvd24gdG9rZW4gb24gdGhlIFdhdmVzIGJsb2NrY2hhaW4gYW5kIGFsc2"+
				"8gc2V0IHRoZSBydWxlcyBmb3IgaXRzIGNpcmN1bGF0aW9uIGJ5IGFzc2lnbmluZyBhIHNjcmlwdCB0byBpdC4EBHN0cjMCbkZ"+
				"vciBleGFtcGxlLCBmb3IgaW4tZ2FtZSBjdXJyZW5jeSwgeW91IGNhbiBhbGxvdyBvbmx5IHRyYW5zYWN0aW9ucyBiZXR3ZWVu"+
				"IGNoYXJhY3RlcnMgd2l0aCBjZXJ0YWluIHByb3BlcnRpZXMuBARzdHI0AmtBIHRva2VuIHdpdGggYW4gYXNzaWduZWQgc2Nya"+
				"XB0IGlzIGNhbGxlZCBhIHNtYXJ0IGFzc2V0LCBhbmQgdGhlIGFzc2lnbmVkIHNjcmlwdCBpcyBjYWxsZWQgYW4gYXNzZXQgc2"+
				"NyaXB0LgQFdGV4dDEJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdHI0BQNuaWwCAyAmIAQ"+
				"FdGV4dDIJALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdHI0BQNuaWwCAyAmIAQFdGV4dDMJ"+
				"ALsJAgkAzAgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdHI0BQNuaWwCAyAmIAQFdGV4dDQJALsJAgkAz"+
				"AgCBQRzdHIxCQDMCAIFBHN0cjIJAMwIAgUEc3RyMwkAzAgCBQRzdHI0BQNuaWwCAyAmIAQFdGV4dDUJALsJAgkAzAgCBQRzdH"+
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
			utl.MinIssueFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			ExpectedValuesPositive{
				WavesDiffBalance: utl.MinIssueFeeWaves,
				AssetBalance:     100000000000,
			}),
		"Simple Script with match...case": *NewIssueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(5, utl.CommonSymbolSet),
			utl.RandStringBytes(20, utl.CommonSymbolSet),
			100000000000,
			8,
			true,
			utl.GetScriptBytes(suite, "BQQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAGVNldEFz"+
				"c2V0U2NyaXB0VHJhbnNhY3Rpb24EAAAAAXQFAAAAByRtYXRjaDAGBm3Fhro="),
			utl.MinIssueFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			ExpectedValuesPositive{
				WavesDiffBalance: utl.MinIssueFeeWaves,
				AssetBalance:     100000000000,
			}),
		"Script with complexity 4000": *NewIssueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(5, utl.CommonSymbolSet),
			utl.RandStringBytes(20, utl.CommonSymbolSet),
			100000000000,
			8,
			true,
			utl.GetScriptBytes(suite, "BgEEByRtYXRjaDAFAnR4AwkAAQIFByRtYXRjaDACE1RyYW5zZmVyVHJhbnNhY3Rpb24E"+
				"AXQFByRtYXRjaDAEAWEJAKQIAQgFAXQJcmVjaXBpZW50AwkAAAIFAWEFAWEDAwMDAwMDAwMDAwMDAwMDAwMDAwMDCQD0AwMBA"+
				"AEAAQAJAPQDAwEAAQABAAcJAPQDAwEAAQABAAcJAPQDAwEAAQABAAcJAPQDAwEAAQABAAcJAPQDAwEAAQABAAcJAPQDAwEAAQ"+
				"ABAAcJAPQDAwEAAQABAAcJAPQDAwEAAQABAAcJAPQDAwEAAQABAAcJAPQDAwEAAQABAAcJAPQDAwEAAQABAAcJAPQDAwEAAQA"+
				"BAAcJAPQDAwEAAQABAAcJAPQDAwEAAQABAAcJAPQDAwEAAQABAAcJAPQDAwEAAQABAAcJAPQDAwEAAQABAAcJAPQDAwEAAQAB"+
				"AAcJAPQDAwEAAQABAAcJAMgTAwEAAQABAAcJAMgTAwEAAQABAAcJAMcTAwEAAQABAAcJAAIBAiRTdHJpY3QgdmFsdWUgaXMgb"+
				"m90IGVxdWFsIHRvIGl0c2VsZi4Gh1kbVQ=="),
			utl.MinIssueFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			ExpectedValuesPositive{
				WavesDiffBalance: utl.MinIssueFeeWaves,
				AssetBalance:     100000000000,
			}),
		/*
			{-# STDLIB_VERSION 5 #-}
			{-# CONTENT_TYPE EXPRESSION #-}
			{-# SCRIPT_TYPE ASSET #-}

			...
		*/
		/*"": *NewIssueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(5, utl.CommonSymbolSet),
			utl.RandStringBytes(20, utl.CommonSymbolSet),
			100000000000,
			8,
			true,
			Script(""),
			utl.MinIssueFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			ExpectedValuesPositive{
				WavesDiffBalance: utl.MinIssueFeeWaves,
				AssetBalance:     100000000000,
			}),
		/*
			{-# STDLIB_VERSION 5 #-}
			{-# CONTENT_TYPE EXPRESSION #-}
			{-# SCRIPT_TYPE ASSET #-}

			...
		*/
		/*"": *NewIssueTestData(
		utl.GetAccount(suite, utl.DefaultSenderNotMiner),
		utl.RandStringBytes(5, utl.CommonSymbolSet),
		utl.RandStringBytes(20, utl.CommonSymbolSet),
		100000000000,
		8,
		true,
		Script(""),
		utl.MinIssueFeeWaves,
		utl.GetCurrentTimestampInMs(),
		utl.TestChainID,
		ExpectedValuesPositive{
			WavesDiffBalance: utl.MinIssueFeeWaves,
			AssetBalance:     100000000000,
		}),*/
	}
}

//данные блокчейна, которые может использовать скрипт

//скрипт написан на Ride

//скрипт скомпилирован в base64

func GetNegativeAssetScriptData(suite *f.BaseSuite) map[string]IssueTestData[ExpectedValuesNegative] {
	return map[string]IssueTestData[ExpectedValuesNegative]{
		/*
			{-# STDLIB_VERSION 5 #-}
			{-# CONTENT_TYPE EXPRESSION #-}
			{-# SCRIPT_TYPE ASSET #-}

			...
		*/
		"Account script as negative data": *NewIssueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(5, utl.CommonSymbolSet),
			utl.RandStringBytes(20, utl.CommonSymbolSet),
			100000000000,
			8,
			true,
			//GetScriptBytes(suite, "BQkAAfQAAAADCAUAAAACdHgAAAAJYm9keUJ5dGVzCQABkQAAAAIIBQAAAAJ0eAAAAAZwcm9vZnMAAAAAAAAAAAAIBQAAAAJ0eAAAAA9zZW5kZXJQdWJsaWNLZXlzTh3b"),
			utl.GetScriptBytes(suite, ""),
			utl.MinIssueFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			ExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "",
				WavesDiffBalance:  0,
				AssetBalance:      0,
			}),
	}
}

/*
	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE EXPRESSION #-}
	{-# SCRIPT_TYPE ASSET #-}

	...
*/
/*"": *NewIssueTestData(
	utl.GetAccount(suite, utl.DefaultSenderNotMiner),
	utl.RandStringBytes(5, utl.CommonSymbolSet),
	utl.RandStringBytes(20, utl.CommonSymbolSet),
	100000000000,
	8,
	true,
	Script(""),
	utl.MinIssueFeeWaves,
	utl.GetCurrentTimestampInMs(),
	utl.TestChainID,
	ExpectedValuesNegative{
		ErrGoMsg:          errMsg,
		ErrScalaMsg:       errMsg,
		ErrBrdCstGoMsg:    errBrdCstMsg,
		ErrBrdCstScalaMsg: "",
		WavesDiffBalance:  0,
		AssetBalance:      0,
	}),
/*
	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE EXPRESSION #-}
	{-# SCRIPT_TYPE ASSET #-}

	...
*/
/*"": *NewIssueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(5, utl.CommonSymbolSet),
			utl.RandStringBytes(20, utl.CommonSymbolSet),
			100000000000,
			8,
			true,
			Script(""),
			utl.MinIssueFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			ExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "",
				WavesDiffBalance:  0,
				AssetBalance:      0,
			}),
	}
}*/
