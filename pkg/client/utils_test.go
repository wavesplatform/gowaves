package client

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewUtils(t *testing.T) {
	assert.NotNil(t, NewUtils(defaultOptions))
}

var utilsSeedJson = `
{
  "seed": "33sJ3mEWyeZ3w004CPChfJvgapbPr88e6XV01Wd2cjyy"
}
`

func TestUtils_Seed(t *testing.T) {
	client, err := NewClient(Options{
		Client:  NewMockHttpRequestFromString(utilsSeedJson, 200),
		BaseUrl: "https://testnode1.wavesnodes.com",
	})
	require.NoError(t, err)
	body, resp, err :=
		client.Utils.Seed(context.Background())
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "33sJ3mEWyeZ3w004CPChfJvgapbPr88e6XV01Wd2cjyy", body)
	assert.Equal(t, "https://testnode1.wavesnodes.com/utils/seed", resp.Request.URL.String())
}

var utilsHashSecureJson = `
{
  "message": "xxx",
  "hash": "FhRKMmvP4qq3ZSQVSpu7QRY9xruYUc9adsxTg56SZhFE"
}
`

func TestUtils_HashSecure(t *testing.T) {
	client, err := NewClient(Options{
		Client:  NewMockHttpRequestFromString(utilsHashSecureJson, 200),
		BaseUrl: "https://testnode1.wavesnodes.com",
	})
	require.NoError(t, err)
	body, resp, err :=
		client.Utils.HashSecure(context.Background(), "xxx")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, &UtilsHashSecure{
		Message: "xxx",
		Hash:    "FhRKMmvP4qq3ZSQVSpu7QRY9xruYUc9adsxTg56SZhFE",
	}, body)
	assert.Equal(t, "https://testnode1.wavesnodes.com/utils/hash/secure", resp.Request.URL.String())
}

func TestUtils_HashFast(t *testing.T) {
	client, err := NewClient(Options{
		Client:  NewMockHttpRequestFromString(utilsHashSecureJson, 200),
		BaseUrl: "https://testnode1.wavesnodes.com",
	})
	require.NoError(t, err)
	body, resp, err :=
		client.Utils.HashFast(context.Background(), "xxx")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, &UtilsHashFast{
		Message: "xxx",
		Hash:    "FhRKMmvP4qq3ZSQVSpu7QRY9xruYUc9adsxTg56SZhFE",
	}, body)
	assert.Equal(t, "https://testnode1.wavesnodes.com/utils/hash/fast", resp.Request.URL.String())
}

var utilsTimeJson = `
{
  "system": 1540980020056,
  "NTP": 1540980020055
}
`

func TestUtils_Time(t *testing.T) {
	client, err := NewClient(Options{
		Client:  NewMockHttpRequestFromString(utilsTimeJson, 200),
		BaseUrl: "https://testnode1.wavesnodes.com",
	})
	require.NoError(t, err)
	body, resp, err :=
		client.Utils.Time(context.Background())
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, &UtilsTime{
		System: 1540980020056,
		NTP:    1540980020055,
	}, body)
	assert.Equal(t, "https://testnode1.wavesnodes.com/utils/time", resp.Request.URL.String())
}

func TestUtils_SeedByLength(t *testing.T) {
	client, err := NewClient(Options{
		Client:  NewMockHttpRequestFromString(utilsSeedJson, 200),
		BaseUrl: "https://testnode1.wavesnodes.com",
	})
	require.NoError(t, err)
	body, resp, err :=
		client.Utils.SeedByLength(context.Background(), 44)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "33sJ3mEWyeZ3w004CPChfJvgapbPr88e6XV01Wd2cjyy", body)
	assert.Equal(t, "https://testnode1.wavesnodes.com/utils/seed/44", resp.Request.URL.String())
}

var utilsScriptCompileJson = `{
"script": "base64:AQa3b8tH",
"complexity": 1,
"extraFee": 400000
}`

func TestUtils_ScriptCompile(t *testing.T) {
	client, err := NewClient(Options{
		Client:  NewMockHttpRequestFromString(utilsScriptCompileJson, 200),
		BaseUrl: "https://testnode1.wavesnodes.com",
	})
	require.NoError(t, err)
	body, resp, err :=
		client.Utils.ScriptCompile(context.Background(), "true")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, &UtilsScriptCompile{
		Script:     "base64:AQa3b8tH",
		Complexity: 1,
		ExtraFee:   400000,
	}, body)
	assert.Equal(t, "https://testnode1.wavesnodes.com/utils/script/compileCode?compact=false", resp.Request.URL.String())
}

const utilsScriptCompileCodeJSON = `{
  "script": "base64:AAIFAAAAAAAAACMIAhIAIgFpIgVhc3NldCIHYXNzZXRJZCICdHgiBnZlcmlmeQAAAAAAAAABAAAAAWEBAAAABGNhbGwAAAAABAAAAAFiCQAEQwAAAAcCAAAABUFzc2V0AgAAAAAAAAAAAAAAAAEAAAAAAAAAAAAGBQAAAAR1bml0AAAAAAAAAAAABAAAAAFjCQAEOAAAAAEFAAAAAWIJAARMAAAAAgkBAAAAC0JpbmFyeUVudHJ5AAAAAgIAAAADYmluAQAAAAAJAARMAAAAAgkBAAAADEJvb2xlYW5FbnRyeQAAAAICAAAABGJvb2wGCQAETAAAAAIJAQAAAAxJbnRlZ2VyRW50cnkAAAACAgAAAANpbnQAAAAAAAAAAAEJAARMAAAAAgkBAAAAC1N0cmluZ0VudHJ5AAAAAgIAAAADc3RyAgAAAAAJAARMAAAAAgkBAAAAC0RlbGV0ZUVudHJ5AAAAAQIAAAADc3RyCQAETAAAAAIFAAAAAWIJAARMAAAAAgkBAAAAB1JlaXNzdWUAAAADBQAAAAFjAAAAAAAAAAABBwkABEwAAAACCQEAAAAEQnVybgAAAAIFAAAAAWMAAAAAAAAAAAEJAARMAAAAAgkBAAAADlNjcmlwdFRyYW5zZmVyAAAAAwgFAAAAAWEAAAAGY2FsbGVyAAAAAAAAAAABBQAAAAFjBQAAAANuaWwAAAABAAAAAWQBAAAAAWUAAAAACQAB9AAAAAMIBQAAAAFkAAAACWJvZHlCeXRlcwkAAZEAAAACCAUAAAABZAAAAAZwcm9vZnMAAAAAAAAAAAAIBQAAAAFkAAAAD3NlbmRlclB1YmxpY0tleS+y6UM=",
  "complexity": 202,
  "verifierComplexity": 202,
  "callableComplexities": {
    "call": 37
  },
  "extraFee": 400000
}`

func TestUtils_ScriptCompileCode(t *testing.T) {
	client, err := NewClient(Options{
		Client:  NewMockHttpRequestFromString(utilsScriptCompileCodeJSON, 200),
		BaseUrl: "https://testnode1.wavesnodes.com",
	})
	require.NoError(t, err)
	body, resp, err :=
		client.Utils.ScriptCompileCode(context.Background(), "true", true)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, &UtilsScriptCompile{
		Script:               "base64:AAIFAAAAAAAAACMIAhIAIgFpIgVhc3NldCIHYXNzZXRJZCICdHgiBnZlcmlmeQAAAAAAAAABAAAAAWEBAAAABGNhbGwAAAAABAAAAAFiCQAEQwAAAAcCAAAABUFzc2V0AgAAAAAAAAAAAAAAAAEAAAAAAAAAAAAGBQAAAAR1bml0AAAAAAAAAAAABAAAAAFjCQAEOAAAAAEFAAAAAWIJAARMAAAAAgkBAAAAC0JpbmFyeUVudHJ5AAAAAgIAAAADYmluAQAAAAAJAARMAAAAAgkBAAAADEJvb2xlYW5FbnRyeQAAAAICAAAABGJvb2wGCQAETAAAAAIJAQAAAAxJbnRlZ2VyRW50cnkAAAACAgAAAANpbnQAAAAAAAAAAAEJAARMAAAAAgkBAAAAC1N0cmluZ0VudHJ5AAAAAgIAAAADc3RyAgAAAAAJAARMAAAAAgkBAAAAC0RlbGV0ZUVudHJ5AAAAAQIAAAADc3RyCQAETAAAAAIFAAAAAWIJAARMAAAAAgkBAAAAB1JlaXNzdWUAAAADBQAAAAFjAAAAAAAAAAABBwkABEwAAAACCQEAAAAEQnVybgAAAAIFAAAAAWMAAAAAAAAAAAEJAARMAAAAAgkBAAAADlNjcmlwdFRyYW5zZmVyAAAAAwgFAAAAAWEAAAAGY2FsbGVyAAAAAAAAAAABBQAAAAFjBQAAAANuaWwAAAABAAAAAWQBAAAAAWUAAAAACQAB9AAAAAMIBQAAAAFkAAAACWJvZHlCeXRlcwkAAZEAAAACCAUAAAABZAAAAAZwcm9vZnMAAAAAAAAAAAAIBQAAAAFkAAAAD3NlbmRlclB1YmxpY0tleS+y6UM=",
		Complexity:           202,
		VerifierComplexity:   202,
		ExtraFee:             400000,
		CallableComplexities: map[string]uint64{"call": 37},
	}, body)
	assert.Equal(t, "https://testnode1.wavesnodes.com/utils/script/compileCode?compact=true", resp.Request.URL.String())
}

const utilsScriptEstimateJSON = `{
  "script": "base64:AAIFAAAAAAAAAAQIAhIAAAAAAAAAAAEAAAABaQEAAAAEY2FsbAAAAAAEAAAABWFzc2V0CQAEQwAAAAcCAAAABUFzc2V0AgAAAAAAAAAAAAAAAAEAAAAAAAAAAAAGBQAAAAR1bml0AAAAAAAAAAAABAAAAAdhc3NldElkCQAEOAAAAAEFAAAABWFzc2V0CQAETAAAAAIJAQAAAAtCaW5hcnlFbnRyeQAAAAICAAAAA2JpbgEAAAAACQAETAAAAAIJAQAAAAxCb29sZWFuRW50cnkAAAACAgAAAARib29sBgkABEwAAAACCQEAAAAMSW50ZWdlckVudHJ5AAAAAgIAAAADaW50AAAAAAAAAAABCQAETAAAAAIJAQAAAAtTdHJpbmdFbnRyeQAAAAICAAAAA3N0cgIAAAAACQAETAAAAAIJAQAAAAtEZWxldGVFbnRyeQAAAAECAAAAA3N0cgkABEwAAAACBQAAAAVhc3NldAkABEwAAAACCQEAAAAHUmVpc3N1ZQAAAAMFAAAAB2Fzc2V0SWQAAAAAAAAAAAEHCQAETAAAAAIJAQAAAARCdXJuAAAAAgUAAAAHYXNzZXRJZAAAAAAAAAAAAQkABEwAAAACCQEAAAAOU2NyaXB0VHJhbnNmZXIAAAADCAUAAAABaQAAAAZjYWxsZXIAAAAAAAAAAAEFAAAAB2Fzc2V0SWQFAAAAA25pbAAAAAEAAAACdHgBAAAABnZlcmlmeQAAAAAJAAH0AAAAAwgFAAAAAnR4AAAACWJvZHlCeXRlcwkAAZEAAAACCAUAAAACdHgAAAAGcHJvb2ZzAAAAAAAAAAAACAUAAAACdHgAAAAPc2VuZGVyUHVibGljS2V5ZN0QxA==",
  "scriptText": "DApp(DAppMeta(2,Vector(CallableFuncSignature(<ByteString@5137f21 size=0 contents=\"\">,UnknownFieldSet(Map()))),Vector(),Vector(),UnknownFieldSet(Map())),List(),List(CallableFunction(CallableAnnotation(i),FUNC(call,List(),LET_BLOCK(LET(asset,FUNCTION_CALL(Native(1091),List(Asset, , 1, 0, true, REF(unit), 0))),LET_BLOCK(LET(assetId,FUNCTION_CALL(Native(1080),List(REF(asset)))),FUNCTION_CALL(Native(1100),List(FUNCTION_CALL(User(BinaryEntry),List(bin, )), FUNCTION_CALL(Native(1100),List(FUNCTION_CALL(User(BooleanEntry),List(bool, true)), FUNCTION_CALL(Native(1100),List(FUNCTION_CALL(User(IntegerEntry),List(int, 1)), FUNCTION_CALL(Native(1100),List(FUNCTION_CALL(User(StringEntry),List(str, )), FUNCTION_CALL(Native(1100),List(FUNCTION_CALL(User(DeleteEntry),List(str)), FUNCTION_CALL(Native(1100),List(REF(asset), FUNCTION_CALL(Native(1100),List(FUNCTION_CALL(User(Reissue),List(REF(assetId), 1, false)), FUNCTION_CALL(Native(1100),List(FUNCTION_CALL(User(Burn),List(REF(assetId), 1)), FUNCTION_CALL(Native(1100),List(FUNCTION_CALL(User(ScriptTransfer),List(GETTER(REF(i),caller), 1, REF(assetId))), REF(nil)))))))))))))))))))))))),Some(VerifierFunction(VerifierAnnotation(tx),FUNC(verify,List(),FUNCTION_CALL(Native(500),List(GETTER(REF(tx),bodyBytes), FUNCTION_CALL(Native(401),List(GETTER(REF(tx),proofs), 0)), GETTER(REF(tx),senderPublicKey)))))))",
  "complexity": 202,
  "verifierComplexity": 202,
  "callableComplexities": {
    "call": 37
  },
  "extraFee": 400000
}`

func TestUtils_ScriptEstimate(t *testing.T) {
	client, err := NewClient(Options{
		Client:  NewMockHttpRequestFromString(utilsScriptEstimateJSON, 200),
		BaseUrl: "https://testnode1.wavesnodes.com",
	})
	require.NoError(t, err)
	body, resp, err :=
		client.Utils.ScriptEstimate(context.Background(), "base64:AQa3b8tH")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, &UtilsScriptEstimate{
		Script:               "base64:AAIFAAAAAAAAAAQIAhIAAAAAAAAAAAEAAAABaQEAAAAEY2FsbAAAAAAEAAAABWFzc2V0CQAEQwAAAAcCAAAABUFzc2V0AgAAAAAAAAAAAAAAAAEAAAAAAAAAAAAGBQAAAAR1bml0AAAAAAAAAAAABAAAAAdhc3NldElkCQAEOAAAAAEFAAAABWFzc2V0CQAETAAAAAIJAQAAAAtCaW5hcnlFbnRyeQAAAAICAAAAA2JpbgEAAAAACQAETAAAAAIJAQAAAAxCb29sZWFuRW50cnkAAAACAgAAAARib29sBgkABEwAAAACCQEAAAAMSW50ZWdlckVudHJ5AAAAAgIAAAADaW50AAAAAAAAAAABCQAETAAAAAIJAQAAAAtTdHJpbmdFbnRyeQAAAAICAAAAA3N0cgIAAAAACQAETAAAAAIJAQAAAAtEZWxldGVFbnRyeQAAAAECAAAAA3N0cgkABEwAAAACBQAAAAVhc3NldAkABEwAAAACCQEAAAAHUmVpc3N1ZQAAAAMFAAAAB2Fzc2V0SWQAAAAAAAAAAAEHCQAETAAAAAIJAQAAAARCdXJuAAAAAgUAAAAHYXNzZXRJZAAAAAAAAAAAAQkABEwAAAACCQEAAAAOU2NyaXB0VHJhbnNmZXIAAAADCAUAAAABaQAAAAZjYWxsZXIAAAAAAAAAAAEFAAAAB2Fzc2V0SWQFAAAAA25pbAAAAAEAAAACdHgBAAAABnZlcmlmeQAAAAAJAAH0AAAAAwgFAAAAAnR4AAAACWJvZHlCeXRlcwkAAZEAAAACCAUAAAACdHgAAAAGcHJvb2ZzAAAAAAAAAAAACAUAAAACdHgAAAAPc2VuZGVyUHVibGljS2V5ZN0QxA==",
		ScriptText:           "DApp(DAppMeta(2,Vector(CallableFuncSignature(<ByteString@5137f21 size=0 contents=\"\">,UnknownFieldSet(Map()))),Vector(),Vector(),UnknownFieldSet(Map())),List(),List(CallableFunction(CallableAnnotation(i),FUNC(call,List(),LET_BLOCK(LET(asset,FUNCTION_CALL(Native(1091),List(Asset, , 1, 0, true, REF(unit), 0))),LET_BLOCK(LET(assetId,FUNCTION_CALL(Native(1080),List(REF(asset)))),FUNCTION_CALL(Native(1100),List(FUNCTION_CALL(User(BinaryEntry),List(bin, )), FUNCTION_CALL(Native(1100),List(FUNCTION_CALL(User(BooleanEntry),List(bool, true)), FUNCTION_CALL(Native(1100),List(FUNCTION_CALL(User(IntegerEntry),List(int, 1)), FUNCTION_CALL(Native(1100),List(FUNCTION_CALL(User(StringEntry),List(str, )), FUNCTION_CALL(Native(1100),List(FUNCTION_CALL(User(DeleteEntry),List(str)), FUNCTION_CALL(Native(1100),List(REF(asset), FUNCTION_CALL(Native(1100),List(FUNCTION_CALL(User(Reissue),List(REF(assetId), 1, false)), FUNCTION_CALL(Native(1100),List(FUNCTION_CALL(User(Burn),List(REF(assetId), 1)), FUNCTION_CALL(Native(1100),List(FUNCTION_CALL(User(ScriptTransfer),List(GETTER(REF(i),caller), 1, REF(assetId))), REF(nil)))))))))))))))))))))))),Some(VerifierFunction(VerifierAnnotation(tx),FUNC(verify,List(),FUNCTION_CALL(Native(500),List(GETTER(REF(tx),bodyBytes), FUNCTION_CALL(Native(401),List(GETTER(REF(tx),proofs), 0)), GETTER(REF(tx),senderPublicKey)))))))",
		Complexity:           202,
		VerifierComplexity:   202,
		ExtraFee:             400000,
		CallableComplexities: map[string]uint64{"call": 37},
	}, body)
	assert.Equal(t, "https://testnode1.wavesnodes.com/utils/script/estimate", resp.Request.URL.String())
}
