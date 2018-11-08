package client

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
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
		ApiKey:  "ApiKey",
		BaseUrl: "https://testnode1.wavesnodes.com",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Utils.Seed(context.Background())
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "33sJ3mEWyeZ3w004CPChfJvgapbPr88e6XV01Wd2cjyy", body)
	assert.Equal(t, "https://testnode1.wavesnodes.com/utils/seed", resp.Request.URL.String())
	assert.Equal(t, "ApiKey", resp.Request.Header.Get("X-Api-Key"))
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
		ApiKey:  "ApiKey",
		BaseUrl: "https://testnode1.wavesnodes.com",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Utils.HashSecure(context.Background(), "xxx")
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, &UtilsHashSecure{
		Message: "xxx",
		Hash:    "FhRKMmvP4qq3ZSQVSpu7QRY9xruYUc9adsxTg56SZhFE",
	}, body)
	assert.Equal(t, "https://testnode1.wavesnodes.com/utils/hash/secure", resp.Request.URL.String())
	assert.Equal(t, "ApiKey", resp.Request.Header.Get("X-Api-Key"))
}

func TestUtils_HashFast(t *testing.T) {
	client, err := NewClient(Options{
		Client:  NewMockHttpRequestFromString(utilsHashSecureJson, 200),
		ApiKey:  "ApiKey",
		BaseUrl: "https://testnode1.wavesnodes.com",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Utils.HashFast(context.Background(), "xxx")
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, &UtilsHashFast{
		Message: "xxx",
		Hash:    "FhRKMmvP4qq3ZSQVSpu7QRY9xruYUc9adsxTg56SZhFE",
	}, body)
	assert.Equal(t, "https://testnode1.wavesnodes.com/utils/hash/fast", resp.Request.URL.String())
	assert.Equal(t, "ApiKey", resp.Request.Header.Get("X-Api-Key"))
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
		ApiKey:  "ApiKey",
		BaseUrl: "https://testnode1.wavesnodes.com",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Utils.Time(context.Background())
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, &UtilsTime{
		System: 1540980020056,
		NTP:    1540980020055,
	}, body)
	assert.Equal(t, "https://testnode1.wavesnodes.com/utils/time", resp.Request.URL.String())
	assert.Equal(t, "ApiKey", resp.Request.Header.Get("X-Api-Key"))
}

var utilsSignJson = `
{
  "message": "123124122421",
  "signature": "4yGyW7bQzAHnwxCCv5jLmJakJ9c2ypeU1vvwTEDib5XXCfsU5dmbzEihf2KmAHEC3ULfJzwji7f9vmDPbESDgfzM"
}`

func TestUtils_Sign(t *testing.T) {
	secretKey, err := crypto.NewSecretKeyFromBase58("YoLY4iripseWvtMt29sc89oJnjxzodDgQ9REmEPFHkK")
	require.Nil(t, err)

	client, err := NewClient(Options{
		Client:  NewMockHttpRequestFromString(utilsSignJson, 200),
		ApiKey:  "ApiKey",
		BaseUrl: "https://testnode1.wavesnodes.com",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Utils.Sign(context.Background(), secretKey, "123124122421")
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, &UtilsSign{
		Message:   "123124122421",
		Signature: "4yGyW7bQzAHnwxCCv5jLmJakJ9c2ypeU1vvwTEDib5XXCfsU5dmbzEihf2KmAHEC3ULfJzwji7f9vmDPbESDgfzM",
	}, body)
	assert.Equal(t, "https://testnode1.wavesnodes.com/utils/sign/YoLY4iripseWvtMt29sc89oJnjxzodDgQ9REmEPFHkK", resp.Request.URL.String())
	assert.Equal(t, "ApiKey", resp.Request.Header.Get("X-Api-Key"))
}

func TestUtils_SeedByLength(t *testing.T) {
	client, err := NewClient(Options{
		Client:  NewMockHttpRequestFromString(utilsSeedJson, 200),
		ApiKey:  "ApiKey",
		BaseUrl: "https://testnode1.wavesnodes.com",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Utils.SeedByLength(context.Background(), 44)
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "33sJ3mEWyeZ3w004CPChfJvgapbPr88e6XV01Wd2cjyy", body)
	assert.Equal(t, "https://testnode1.wavesnodes.com/utils/seed/44", resp.Request.URL.String())
	assert.Equal(t, "ApiKey", resp.Request.Header.Get("X-Api-Key"))
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
	require.Nil(t, err)
	body, resp, err :=
		client.Utils.ScriptCompile(context.Background(), "true")
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, &UtilsScriptCompile{
		Script:     "base64:AQa3b8tH",
		Complexity: 1,
		ExtraFee:   400000,
	}, body)
	assert.Equal(t, "https://testnode1.wavesnodes.com/utils/script/compile", resp.Request.URL.String())
}

var utilsScriptEstimateJson = `
{
  "script": "base64:AQa3b8tH",
  "scriptText": "TRUE",
  "complexity": 1,
  "extraFee": 400000
}`

func TestUtils_ScriptEstimate(t *testing.T) {
	client, err := NewClient(Options{
		Client:  NewMockHttpRequestFromString(utilsScriptEstimateJson, 200),
		BaseUrl: "https://testnode1.wavesnodes.com",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Utils.ScriptEstimate(context.Background(), "base64:AQa3b8tH")
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, &UtilsScriptEstimate{
		Script:     "base64:AQa3b8tH",
		ScriptText: "TRUE",
		Complexity: 1,
		ExtraFee:   400000,
	}, body)
	assert.Equal(t, "https://testnode1.wavesnodes.com/utils/script/estimate", resp.Request.URL.String())
}
