package fride

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/ast"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/reader"
)

func TestEstimatorV1(t *testing.T) {
	for _, test := range []struct {
		code      string
		script    string
		catalogue *Catalogue
		cost      uint64
	}{
		{`false`, "AweHXCN1", NewCatalogueV3(), 1},
		{`unit == Unit()`, "AwkAAAAAAAACBQAAAAR1bml0CQEAAAAEVW5pdAAAAACd7sMa", NewCatalogueV3(), 3},
		{`12345 == 12345`, "AwkAAAAAAAACAAAAAAAAADA5AAAAAAAAADA5+DindQ==", NewCatalogueV3(), 3},
		{`let x = 2 * 2; x == 4`, "AwQAAAABeAkAAGgAAAACAAAAAAAAAAACAAAAAAAAAAACCQAAAAAAAAIFAAAAAXgAAAAAAAAAAARdrwMC", NewCatalogueV3(), 12},
		{`let a = "A"; let b = "B"; a + b == "AB"`, "AwQAAAABYQIAAAABQQQAAAABYgIAAAABQgkAAAAAAAACCQABLAAAAAIFAAAAAWEFAAAAAWICAAAAAkFC8C4jQA==", NewCatalogueV3(), 28},
		{`fromBase58String("") == base16'cafebebe'`, "AwkAAAAAAAACCQACWQAAAAECAAAAAAEAAAAEyv6+vpLxJHA=", NewCatalogueV3(), 13},
		{`Address(base58'11111111111111111') == Address(base58'11111111111111111')`, "AwkAAAAAAAACCQEAAAAHQWRkcmVzcwAAAAEBAAAAEQAAAAAAAAAAAAAAAAAAAAAACQEAAAAHQWRkcmVzcwAAAAEBAAAAEQAAAAAAAAAAAAAAAAAAAAAA2+A0og==", NewCatalogueV3(), 5},
		{`toString(Address(base58'3P3336rNSSU8bDAqDb6S5jNs8DJb2bfNmpg')) == "3P3336rNSSU8bDAqDb6S5jNs8DJb2bfNmpf"`, "AwkAAAAAAAACCQAEJQAAAAEJAQAAAAdBZGRyZXNzAAAAAQEAAAAaAVcMIZxOsk2Gw5Avd0ztqi+phtb1Bb83MiUCAAAAIzNQMzMzNnJOU1NVOGJEQXFEYjZTNWpOczhESmIyYmZObXBmb/6mcg==", NewCatalogueV3(), 14},
		{`tx.sender == Address(base58'11111111111111111')`, "AwkAAAAAAAACCAUAAAACdHgAAAAGc2VuZGVyCQEAAAAHQWRkcmVzcwAAAAEBAAAAEQAAAAAAAAAAAAAAAAAAAAAAWc7d/w==", NewCatalogueV3(), 7},
		{`parseIntValue("012345") == 12345`, "AwkAAAAAAAACCQEAAAANcGFyc2VJbnRWYWx1ZQAAAAECAAAABjAxMjM0NQAAAAAAAAAwOXCRV0U=", NewCatalogueV3(), 23},
		{`let x = parseIntValue("12345"); x + x == 0`, "AwQAAAABeAkBAAAADXBhcnNlSW50VmFsdWUAAAABAgAAAAUxMjM0NQkAAAAAAAACCQAAZAAAAAIFAAAAAXgFAAAAAXgAAAAAAAAAAADVoBKt", NewCatalogueV3(), 33},
		{`let x = parseIntValue("12345"); 0 == 0`, "AwQAAAABeAkBAAAADXBhcnNlSW50VmFsdWUAAAABAgAAAAUxMjM0NQkAAAAAAAACAAAAAAAAAAAAAAAAAAAAAAAAk6EsIQ==", NewCatalogueV3(), 8},
		{`let x = parseIntValue("123"); let y = parseIntValue("456");  x + y == y + x`, "AwQAAAABeAkBAAAADXBhcnNlSW50VmFsdWUAAAABAgAAAAMxMjMEAAAAAXkJAQAAAA1wYXJzZUludFZhbHVlAAAAAQIAAAADNDU2CQAAAAAAAAIJAABkAAAAAgUAAAABeAUAAAABeQkAAGQAAAACBQAAAAF5BQAAAAF4sUY0sQ==", NewCatalogueV3(), 63},
		{`let d = [DataEntry("integer", 100500), DataEntry("boolean", true), DataEntry("binary", base16'68656c6c6f'), DataEntry("string", "world")]; getInteger(d, "integer") == 100500`, "AwQAAAABZAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHaW50ZWdlcgAAAAAAAAGIlAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHYm9vbGVhbgYJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABmJpbmFyeQEAAAAFaGVsbG8JAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABnN0cmluZwIAAAAFd29ybGQFAAAAA25pbAkAAAAAAAACCQAEEAAAAAIFAAAAAWQCAAAAB2ludGVnZXIAAAAAAAABiJSeStXa", NewCatalogueV3(), 46},
		{`let d = [DataEntry("integer",100500), DataEntry("boolean", true), DataEntry("binary", base16'68656c6c6f'), DataEntry("string", "world")]; getString(d, "string") == "world"`, "AwQAAAABZAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHaW50ZWdlcgAAAAAAAAGIlAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHYm9vbGVhbgYJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABmJpbmFyeQEAAAAFaGVsbG8JAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABnN0cmluZwIAAAAFd29ybGQFAAAAA25pbAkAAAAAAAACCQAEEwAAAAIFAAAAAWQCAAAABnN0cmluZwIAAAAFd29ybGRFTMLs", NewCatalogueV3(), 46},
		{`let x = 1 + 2; x == 0`, "AwQAAAABeAkAAGQAAAACAAAAAAAAAAABAAAAAAAAAAACCQAAAAAAAAIFAAAAAXgAAAAAAAAAAABuZPgv", NewCatalogueV3(), 12},
		{`let x = 2 + 2; let y = x - x; x - y == x`, "AwQAAAABeAkAAGQAAAACAAAAAAAAAAACAAAAAAAAAAACBAAAAAF5CQAAZQAAAAIFAAAAAXgFAAAAAXgJAAAAAAAAAgkAAGUAAAACBQAAAAF4BQAAAAF5BQAAAAF4G74APQ==", NewCatalogueV3(), 26},
		{`let a = 1 + 2; let b = 2; let c = a + b; b == 0`, "AwQAAAABYQkAAGQAAAACAAAAAAAAAAABAAAAAAAAAAACBAAAAAFiAAAAAAAAAAACBAAAAAFjCQAAZAAAAAIFAAAAAWEFAAAAAWIJAAAAAAAAAgUAAAABYgAAAAAAAAAAAGbVbuk=", NewCatalogueV3(), 20},
		{`let a = 1 + 2 + 3; let b = 4 + 5; let c = if false then a else b; c == 0`, "AwQAAAABYQkAAGQAAAACCQAAZAAAAAIAAAAAAAAAAAEAAAAAAAAAAAIAAAAAAAAAAAMEAAAAAWIJAABkAAAAAgAAAAAAAAAABAAAAAAAAAAABQQAAAABYwMHBQAAAAFhBQAAAAFiCQAAAAAAAAIFAAAAAWMAAAAAAAAAAABW2XVO", NewCatalogueV3(), 28},
		{`big script`, "AQQAAAAMbWF4VGltZVRvQmV0AAAAAWiZ4tPwBAAAABBtaW5UaW1lVG9UcmFkaW5nAAAAAWiZ5KiwBAAAABBtYXhUaW1lVG9UcmFkaW5nAAAAAWiZ5ZMQBAAAAANmZWUAAAAAAACYloAEAAAACGRlY2ltYWxzAAAAAAAAAAACBAAAAAhtdWx0aXBseQAAAAAAAAAAZAQAAAAKdG90YWxNb25leQMJAQAAAAlpc0RlZmluZWQAAAABCQAEGgAAAAIIBQAAAAJ0eAAAAAZzZW5kZXICAAAACnRvdGFsTW9uZXkJAQAAAAdleHRyYWN0AAAAAQkABBoAAAACCAUAAAACdHgAAAAGc2VuZGVyAgAAAAp0b3RhbE1vbmV5AAAAAAAAAAAABAAAAAp1bmlxdWVCZXRzAwkBAAAACWlzRGVmaW5lZAAAAAEJAAQaAAAAAggFAAAAAnR4AAAABnNlbmRlcgIAAAAKdW5pcXVlQmV0cwkBAAAAB2V4dHJhY3QAAAABCQAEGgAAAAIIBQAAAAJ0eAAAAAZzZW5kZXICAAAACnVuaXF1ZUJldHMAAAAAAAAAAAAEAAAAByRtYXRjaDAFAAAAAnR4AwkAAAEAAAACBQAAAAckbWF0Y2gwAgAAAA9EYXRhVHJhbnNhY3Rpb24EAAAAAmR0BQAAAAckbWF0Y2gwAwMJAABnAAAAAgUAAAAMbWF4VGltZVRvQmV0CAUAAAACdHgAAAAJdGltZXN0YW1wCQEAAAAJaXNEZWZpbmVkAAAAAQkABBMAAAACCAUAAAACZHQAAAAEZGF0YQIAAAAFYmV0X3MHBAAAAAtwYXltZW50VHhJZAkBAAAAB2V4dHJhY3QAAAABCQAEEwAAAAIIBQAAAAJkdAAAAARkYXRhAgAAAAtwYXltZW50VHhJZAQAAAAJcGF5bWVudFR4CQAD6AAAAAEJAAJZAAAAAQUAAAALcGF5bWVudFR4SWQEAAAACGJldEdyb3VwCQEAAAAHZXh0cmFjdAAAAAEJAAQTAAAAAggFAAAAAmR0AAAABGRhdGECAAAABWJldF9zBAAAAAxkdEJldFN1bW1hcnkJAQAAAAdleHRyYWN0AAAAAQkABBAAAAACCAUAAAACZHQAAAAEZGF0YQUAAAAIYmV0R3JvdXAEAAAACmJldFN1bW1hcnkDCQEAAAAJaXNEZWZpbmVkAAAAAQkABBoAAAACCAUAAAACdHgAAAAGc2VuZGVyBQAAAAhiZXRHcm91cAkBAAAAB2V4dHJhY3QAAAABCQAEGgAAAAIIBQAAAAJ0eAAAAAZzZW5kZXIFAAAACGJldEdyb3VwAAAAAAAAAAAABAAAAAR2QmV0CQEAAAAHZXh0cmFjdAAAAAEJAAQQAAAAAggFAAAAAmR0AAAABGRhdGECAAAABWJldF92BAAAAAZrdnBCZXQJAQAAAAdleHRyYWN0AAAAAQkABBMAAAACCAUAAAACZHQAAAAEZGF0YQkAAaQAAAABBQAAAAR2QmV0BAAAAAd2S3ZwQmV0CQEAAAAHZXh0cmFjdAAAAAEJAAQQAAAAAggFAAAAAmR0AAAABGRhdGEJAAEsAAAAAgIAAAACdl8JAAGkAAAAAQUAAAAEdkJldAQAAAAEaUJldAkBAAAAB2V4dHJhY3QAAAABCQAEEAAAAAIIBQAAAAJkdAAAAARkYXRhAgAAAAViZXRfaQQAAAAEZEJldAkBAAAAB2V4dHJhY3QAAAABCQAEEAAAAAIIBQAAAAJkdAAAAARkYXRhAgAAAAViZXRfZAQAAAABYwkAAGUAAAACBQAAAAhkZWNpbWFscwkAATEAAAABCQABpAAAAAEFAAAABGRCZXQEAAAABHRCZXQJAAEsAAAAAgkAASwAAAACCQABLAAAAAIJAAGkAAAAAQUAAAAEaUJldAIAAAABLgMJAAAAAAAAAgUAAAABYwAAAAAAAAAAAQIAAAABMAMJAAAAAAAAAgUAAAABYwAAAAAAAAAAAgIAAAACMDADCQAAAAAAAAIFAAAAAWMAAAAAAAAAAAMCAAAAAzAwMAMJAAAAAAAAAgUAAAABYwAAAAAAAAAABAIAAAAEMDAwMAMJAAAAAAAAAgUAAAABYwAAAAAAAAAABQIAAAAFMDAwMDADCQAAAAAAAAIFAAAAAWMAAAAAAAAAAAYCAAAABjAwMDAwMAMJAAAAAAAAAgUAAAABYwAAAAAAAAAABwIAAAAHMDAwMDAwMAIAAAAACQABpAAAAAEFAAAABGRCZXQEAAAACGJldElzTmV3AwkBAAAAASEAAAABCQEAAAAJaXNEZWZpbmVkAAAAAQkABBoAAAACCAUAAAACdHgAAAAGc2VuZGVyBQAAAAhiZXRHcm91cAAAAAAAAAAAAQAAAAAAAAAAAAQAAAAMZHRVbmlxdWVCZXRzCQEAAAAHZXh0cmFjdAAAAAEJAAQQAAAAAggFAAAAAmR0AAAABGRhdGECAAAACnVuaXF1ZUJldHMEAAAAByRtYXRjaDEFAAAACXBheW1lbnRUeAMJAAABAAAAAgUAAAAHJG1hdGNoMQIAAAATVHJhbnNmZXJUcmFuc2FjdGlvbgQAAAAHcGF5bWVudAUAAAAHJG1hdGNoMQMDAwMDAwMDCQEAAAABIQAAAAEJAQAAAAlpc0RlZmluZWQAAAABCQAEHQAAAAIIBQAAAAJ0eAAAAAZzZW5kZXIFAAAAC3BheW1lbnRUeElkCQAAAAAAAAIIBQAAAAdwYXltZW50AAAACXJlY2lwaWVudAgFAAAAAnR4AAAABnNlbmRlcgcJAABmAAAAAggFAAAAB3BheW1lbnQAAAAGYW1vdW50BQAAAANmZWUHCQAAAAAAAAIJAQAAAAdleHRyYWN0AAAAAQkABBAAAAACCAUAAAACZHQAAAAEZGF0YQIAAAAKdG90YWxNb25leQkAAGQAAAACBQAAAAp0b3RhbE1vbmV5CQAAZQAAAAIIBQAAAAdwYXltZW50AAAABmFtb3VudAUAAAADZmVlBwkAAAAAAAACBQAAAAxkdEJldFN1bW1hcnkJAABkAAAAAgUAAAAKYmV0U3VtbWFyeQkAAGUAAAACCAUAAAAHcGF5bWVudAAAAAZhbW91bnQFAAAAA2ZlZQcJAAAAAAAAAgUAAAAEdkJldAkAAGQAAAACCQAAaAAAAAIFAAAABGlCZXQFAAAACG11bHRpcGx5BQAAAARkQmV0BwkAAAAAAAACBQAAAAZrdnBCZXQFAAAACGJldEdyb3VwBwkAAAAAAAACBQAAAAxkdFVuaXF1ZUJldHMJAABkAAAAAgUAAAAKdW5pcXVlQmV0cwUAAAAIYmV0SXNOZXcHCQAAAAAAAAIFAAAAB3ZLdnBCZXQFAAAABHZCZXQHBwMDCQAAZgAAAAIIBQAAAAJ0eAAAAAl0aW1lc3RhbXAFAAAAEG1pblRpbWVUb1RyYWRpbmcJAQAAAAEhAAAAAQkBAAAACWlzRGVmaW5lZAAAAAEJAAQdAAAAAggFAAAAAnR4AAAABnNlbmRlcgIAAAALdHJhZGluZ1R4SWQHBAAAAAt0cmFkaW5nVHhJZAkBAAAAB2V4dHJhY3QAAAABCQAEEwAAAAIIBQAAAAJkdAAAAARkYXRhAgAAAAt0cmFkaW5nVHhJZAQAAAAJdHJhZGluZ1R4CQAD6AAAAAEJAAJZAAAAAQUAAAALdHJhZGluZ1R4SWQEAAAACHByaWNlV2luCQEAAAAHZXh0cmFjdAAAAAEJAAQQAAAAAggFAAAAAmR0AAAABGRhdGECAAAACHByaWNlV2luBAAAAAdkdERlbHRhCQEAAAAHZXh0cmFjdAAAAAEJAAQQAAAAAggFAAAAAmR0AAAABGRhdGECAAAABWRlbHRhBAAAAAlkdFNvcnROdW0JAQAAAAdleHRyYWN0AAAAAQkABBAAAAACCAUAAAACZHQAAAAEZGF0YQIAAAAHc29ydE51bQQAAAAHJG1hdGNoMQUAAAAJdHJhZGluZ1R4AwkAAAEAAAACBQAAAAckbWF0Y2gxAgAAABNFeGNoYW5nZVRyYW5zYWN0aW9uBAAAAAhleGNoYW5nZQUAAAAHJG1hdGNoMQMDAwMJAAAAAAAAAgUAAAAIcHJpY2VXaW4IBQAAAAhleGNoYW5nZQAAAAVwcmljZQkAAGcAAAACCAUAAAAIZXhjaGFuZ2UAAAAJdGltZXN0YW1wBQAAABBtaW5UaW1lVG9UcmFkaW5nBwkAAGcAAAACBQAAABBtYXhUaW1lVG9UcmFkaW5nCAUAAAAIZXhjaGFuZ2UAAAAJdGltZXN0YW1wBwkAAAAAAAACBQAAAAdkdERlbHRhAAAAABdIdugABwkAAAAAAAACBQAAAAlkdFNvcnROdW0AAAAAAAAAAAAHBwMJAQAAAAlpc0RlZmluZWQAAAABCQAEHQAAAAIIBQAAAAJ0eAAAAAZzZW5kZXICAAAAC3RyYWRpbmdUeElkBAAAAAZ3aW5CZXQDCQEAAAAJaXNEZWZpbmVkAAAAAQkABBoAAAACCAUAAAACdHgAAAAGc2VuZGVyAgAAAAZ3aW5CZXQJAQAAAAdleHRyYWN0AAAAAQkABBoAAAACCAUAAAACdHgAAAAGc2VuZGVyAgAAAAVkZWx0YQAAAAAXSHboAAQAAAAIcHJpY2VXaW4JAQAAAAdleHRyYWN0AAAAAQkABBAAAAACCAUAAAACZHQAAAAEZGF0YQIAAAAIcHJpY2VXaW4EAAAACWR0U29ydE51bQkBAAAAB2V4dHJhY3QAAAABCQAEEAAAAAIIBQAAAAJkdAAAAARkYXRhAgAAAAdzb3J0TnVtBAAAAAdzb3J0TnVtCQEAAAAHZXh0cmFjdAAAAAEJAAQaAAAAAggFAAAAAnR4AAAABnNlbmRlcgIAAAAHc29ydE51bQQAAAAJc29ydFZhbHVlCQEAAAAHZXh0cmFjdAAAAAEJAAQaAAAAAggFAAAAAnR4AAAABnNlbmRlcgIAAAAJc29ydFZhbHVlBAAAAA1zb3J0VmFsdWVUZXh0CQEAAAAHZXh0cmFjdAAAAAEJAAQdAAAAAggFAAAAAnR4AAAABnNlbmRlcgIAAAANc29ydFZhbHVlVGV4dAQAAAAIZHRXaW5CZXQJAQAAAAdleHRyYWN0AAAAAQkABBoAAAACCAUAAAACdHgAAAAGc2VuZGVyAgAAAAZ3aW5CZXQEAAAADXNvcnRpbmdFeGlzdHMDCQAAZgAAAAIAAAAAAAAAAAAJAABlAAAAAgUAAAAIcHJpY2VXaW4FAAAABndpbkJldAkAAGUAAAACBQAAAAZ3aW5CZXQFAAAACHByaWNlV2luCQAAZQAAAAIFAAAACHByaWNlV2luBQAAAAZ3aW5CZXQEAAAACnNvcnRpbmdOZXcDCQAAZgAAAAIAAAAAAAAAAAAJAABlAAAAAgUAAAAIcHJpY2VXaW4FAAAACXNvcnRWYWx1ZQkAAGUAAAACBQAAAAlzb3J0VmFsdWUFAAAACHByaWNlV2luCQAAZQAAAAIFAAAACHByaWNlV2luBQAAAAlzb3J0VmFsdWUEAAAAB3NvcnRpbmcDCQAAZgAAAAIFAAAADXNvcnRpbmdFeGlzdHMFAAAACnNvcnRpbmdOZXcFAAAACXNvcnRWYWx1ZQUAAAAGd2luQmV0BAAAAAxkdFVuaXF1ZUJldHMJAQAAAAdleHRyYWN0AAAAAQkABBAAAAACCAUAAAACZHQAAAAEZGF0YQIAAAAKdW5pcXVlQmV0cwMDAwMDAwMJAABmAAAAAgUAAAAMZHRVbmlxdWVCZXRzBQAAAAlkdFNvcnROdW0JAAAAAAAAAgUAAAAJZHRTb3J0TnVtCQAAZAAAAAIFAAAAB3NvcnROdW0AAAAAAAAAAAEHCQEAAAAJaXNEZWZpbmVkAAAAAQkABBoAAAACCAUAAAACdHgAAAAGc2VuZGVyCQABLAAAAAICAAAAAnZfCQABpAAAAAEFAAAACXNvcnRWYWx1ZQcJAAAAAAAAAgUAAAAJc29ydFZhbHVlCQEAAAAHZXh0cmFjdAAAAAEJAAQaAAAAAggFAAAAAnR4AAAABnNlbmRlcgkAASwAAAACAgAAAAJ2XwkAAaQAAAABBQAAAAlzb3J0VmFsdWUHCQEAAAABIQAAAAEJAQAAAAlpc0RlZmluZWQAAAABCQAEHQAAAAIIBQAAAAJ0eAAAAAZzZW5kZXIJAAEsAAAAAgIAAAAFc29ydF8JAAGkAAAAAQUAAAAJc29ydFZhbHVlBwkAAAAAAAACBQAAAA1zb3J0VmFsdWVUZXh0CQABLAAAAAICAAAABXNvcnRfCQABpAAAAAEFAAAACXNvcnRWYWx1ZQcJAQAAAAlpc0RlZmluZWQAAAABCQAEGgAAAAIIBQAAAAJ0eAAAAAZzZW5kZXIJAAEsAAAAAgIAAAACdl8JAAGkAAAAAQUAAAAIZHRXaW5CZXQHCQAAAAAAAAIFAAAACGR0V2luQmV0BQAAAAdzb3J0aW5nBwcGRZ0fDg==", NewCatalogueV2(), 1970},
		{`XmasTree.ride`, "AgQAAAAQc2VydmVyc1B1YmxpY0tleQEAAAAgPf4rQckjjaheseG0cWkeKNTk8LiefV8gVWwmnlRatgAEAAAAFmVuY3R5cHRlZFNlcnZlcnNDaG9pY2UBAAAAID3+K0HJI42oXrHhtHFpHijU5PC4nn1fIFVsJp5UWrYABAAAAAhkb25hdGlvbgAAAAAABfXhAAQAAAAMcGxheWVyc1ByaXplAAAAAAAF9eEABAAAAApib3hlc0NvdW50AAAAAAAAAAAFBAAAABltYXliZURhdGFQbGF5ZXJzUHVibGljS2V5CQAEHQAAAAIIBQAAAAJ0eAAAAAZzZW5kZXICAAAAEHBsYXllcnNQdWJsaWNLZXkEAAAAFGRhdGFQbGF5ZXJzUHVibGljS2V5CQACWQAAAAEJAQAAAAdleHRyYWN0AAAAAQUAAAAZbWF5YmVEYXRhUGxheWVyc1B1YmxpY0tleQQAAAAMZGF0YUxvY2tlZEF0CQEAAAAHZXh0cmFjdAAAAAEJAAQaAAAAAggFAAAAAnR4AAAABnNlbmRlcgIAAAAIbG9ja2VkQXQEAAAAFm1heWJlRGF0YVBsYXllcnNDaG9pY2UJAAQaAAAAAggFAAAAAnR4AAAABnNlbmRlcgIAAAANcGxheWVyc0Nob2ljZQQAAAARZGF0YVBsYXllcnNDaG9pY2UJAQAAAAdleHRyYWN0AAAAAQUAAAAWbWF5YmVEYXRhUGxheWVyc0Nob2ljZQQAAAAWbWF5YmVEYXRhU2VydmVyc0Nob2ljZQkABBoAAAACCAUAAAACdHgAAAAGc2VuZGVyAgAAAA1zZXJ2ZXJzQ2hvaWNlBAAAABFkYXRhU2VydmVyc0Nob2ljZQkBAAAAB2V4dHJhY3QAAAABBQAAABZtYXliZURhdGFTZXJ2ZXJzQ2hvaWNlBAAAAA90aW1lb3V0SW5CbG9ja3MAAAAAAAAAAHgEAAAAEmRhdGFUcmFuc2FjdGlvbkZlZQAAAAAAAAehIAQAAAAWdHJhbnNmZXJUcmFuc2FjdGlvbkZlZQAAAAAAAAehIAQAAAAOc2VydmVyc0FkZHJlc3MJAQAAABRhZGRyZXNzRnJvbVB1YmxpY0tleQAAAAEFAAAAEHNlcnZlcnNQdWJsaWNLZXkEAAAADnBsYXllcnNBZGRyZXNzCQEAAAAUYWRkcmVzc0Zyb21QdWJsaWNLZXkAAAABBQAAABRkYXRhUGxheWVyc1B1YmxpY0tleQQAAAAOYWNjb3VudEJhbGFuY2UJAQAAAAx3YXZlc0JhbGFuY2UAAAABCAUAAAACdHgAAAAGc2VuZGVyBAAAAA5zZW5kZXJJc1NlcnZlcgkAAfQAAAADCAUAAAACdHgAAAAJYm9keUJ5dGVzCQABkQAAAAIIBQAAAAJ0eAAAAAZwcm9vZnMAAAAAAAAAAAAFAAAAEHNlcnZlcnNQdWJsaWNLZXkEAAAADnNlbmRlcklzUGxheWVyCQAB9AAAAAMIBQAAAAJ0eAAAAAlib2R5Qnl0ZXMJAAGRAAAAAggFAAAAAnR4AAAABnByb29mcwAAAAAAAAAAAAUAAAAUZGF0YVBsYXllcnNQdWJsaWNLZXkEAAAADmlzSW5pdGlhbFN0YXRlAwkBAAAAASEAAAABCQEAAAAJaXNEZWZpbmVkAAAAAQUAAAAZbWF5YmVEYXRhUGxheWVyc1B1YmxpY0tleQkAAGcAAAACBQAAAA5hY2NvdW50QmFsYW5jZQkAAGQAAAACCQAAZAAAAAIFAAAADHBsYXllcnNQcml6ZQkAAGgAAAACBQAAABJkYXRhVHJhbnNhY3Rpb25GZWUAAAAAAAAAAAMFAAAAFnRyYW5zZmVyVHJhbnNhY3Rpb25GZWUHBAAAABlkYXRhSXNWYWxpZEZvckxvY2tlZFN0YXRlAwkBAAAACWlzRGVmaW5lZAAAAAEFAAAAGW1heWJlRGF0YVBsYXllcnNQdWJsaWNLZXkJAQAAAAEhAAAAAQkBAAAACWlzRGVmaW5lZAAAAAEFAAAAFm1heWJlRGF0YVBsYXllcnNDaG9pY2UHBAAAABJsb2NrZWRTdGF0ZVRpbWVvdXQJAABmAAAAAgUAAAAGaGVpZ2h0CQAAZAAAAAIFAAAADGRhdGFMb2NrZWRBdAUAAAAPdGltZW91dEluQmxvY2tzBAAAAA1pc0xvY2tlZFN0YXRlAwUAAAAZZGF0YUlzVmFsaWRGb3JMb2NrZWRTdGF0ZQkBAAAAASEAAAABBQAAABJsb2NrZWRTdGF0ZVRpbWVvdXQHBAAAABtpc1VzZXJEZWNpZGVkTm90VG9QbGF5U3RhdGUDBQAAABlkYXRhSXNWYWxpZEZvckxvY2tlZFN0YXRlBQAAABJsb2NrZWRTdGF0ZVRpbWVvdXQHBAAAACFkYXRhSXNWYWxpZEZvclBsYXllclJldmVhbGVkU3RhdGUDCQEAAAAJaXNEZWZpbmVkAAAAAQUAAAAWbWF5YmVEYXRhUGxheWVyc0Nob2ljZQkBAAAAASEAAAABCQEAAAAJaXNEZWZpbmVkAAAAAQUAAAAWbWF5YmVEYXRhU2VydmVyc0Nob2ljZQcEAAAAGnBsYXllclJldmVhbGVkU3RhdGVUaW1lb3V0AwkAAGYAAAACBQAAAAZoZWlnaHQJAABkAAAAAgUAAAAMZGF0YUxvY2tlZEF0CQAAaAAAAAIFAAAAD3RpbWVvdXRJbkJsb2NrcwAAAAAAAAAAAgkAAGcAAAACBQAAAA5hY2NvdW50QmFsYW5jZQkAAGQAAAACCQAAZAAAAAIJAABkAAAAAgUAAAAMcGxheWVyc1ByaXplBQAAAAhkb25hdGlvbgUAAAASZGF0YVRyYW5zYWN0aW9uRmVlBQAAABZ0cmFuc2ZlclRyYW5zYWN0aW9uRmVlBwQAAAAeaXNQbGF5ZXJSZXZlYWxlZEhpc0Nob2ljZVN0YXRlAwUAAAAhZGF0YUlzVmFsaWRGb3JQbGF5ZXJSZXZlYWxlZFN0YXRlCQEAAAABIQAAAAEFAAAAGnBsYXllclJldmVhbGVkU3RhdGVUaW1lb3V0BwQAAAAdaXNTZXJ2ZXJEZWNpZGVkTm90VG9QbGF5U3RhdGUDBQAAACFkYXRhSXNWYWxpZEZvclBsYXllclJldmVhbGVkU3RhdGUFAAAAGnBsYXllclJldmVhbGVkU3RhdGVUaW1lb3V0BwQAAAAlaXNEYXRhVmFsaWRGb3JXaW5uZXJJc0RldGVybWluZWRTdGF0ZQkBAAAACWlzRGVmaW5lZAAAAAEFAAAAFm1heWJlRGF0YVNlcnZlcnNDaG9pY2UEAAAAKGlzQmFsYW5jZVZhbGlkRm9yV2lubmVySXNEZXRlcm1pbmVkU3RhdGUJAABnAAAAAgUAAAAOYWNjb3VudEJhbGFuY2UJAABkAAAAAgkAAGQAAAACBQAAAAxwbGF5ZXJzUHJpemUFAAAACGRvbmF0aW9uBQAAABZ0cmFuc2ZlclRyYW5zYWN0aW9uRmVlBAAAABlpc1dpbm5lcklzRGV0ZXJtaW5lZFN0YXRlAwUAAAAlaXNEYXRhVmFsaWRGb3JXaW5uZXJJc0RldGVybWluZWRTdGF0ZQUAAAAoaXNCYWxhbmNlVmFsaWRGb3JXaW5uZXJJc0RldGVybWluZWRTdGF0ZQcEAAAAJWlzUGxheWVyRGVjaWRlZE5vdFRvU2VuZERvbmF0aW9uU3RhdGUDBQAAACVpc0RhdGFWYWxpZEZvcldpbm5lcklzRGV0ZXJtaW5lZFN0YXRlCQEAAAABIQAAAAEFAAAAKGlzQmFsYW5jZVZhbGlkRm9yV2lubmVySXNEZXRlcm1pbmVkU3RhdGUHBAAAAAckbWF0Y2gwBQAAAAJ0eAMJAAABAAAAAgUAAAAHJG1hdGNoMAIAAAAPRGF0YVRyYW5zYWN0aW9uBAAAAANkdHgFAAAAByRtYXRjaDAEAAAAC3BheWxvYWRTaXplCQABkAAAAAEIBQAAAANkdHgAAAAEZGF0YQQAAAAQZmlyc3RQYXlsb2FkTmFtZQgJAAGRAAAAAggFAAAAA2R0eAAAAARkYXRhAAAAAAAAAAAAAAAAA2tleQQAAAARc2Vjb25kUGF5bG9hZE5hbWUICQABkQAAAAIIBQAAAANkdHgAAAAEZGF0YQAAAAAAAAAAAQAAAANrZXkEAAAAFWZpcnN0UGF5bG9hZEFzSW50ZWdlcgkBAAAAB2V4dHJhY3QAAAABCQAEEAAAAAIIBQAAAANkdHgAAAAEZGF0YQUAAAAQZmlyc3RQYXlsb2FkTmFtZQQAAAAVc2Vjb25kUGF5bG9hZEFzU3RyaW5nCQEAAAAHZXh0cmFjdAAAAAEJAAQTAAAAAggFAAAAA2R0eAAAAARkYXRhBQAAABFzZWNvbmRQYXlsb2FkTmFtZQQAAAALZGF0YUZlZUlzT2sJAAAAAAAAAggFAAAAA2R0eAAAAANmZWUFAAAAEmRhdGFUcmFuc2FjdGlvbkZlZQMDBQAAAA5pc0luaXRpYWxTdGF0ZQYFAAAAG2lzVXNlckRlY2lkZWROb3RUb1BsYXlTdGF0ZQQAAAAXdmFsaWRMb2NrZWRBdElzUHJvdmlkZWQDCQAAAAAAAAIFAAAAEGZpcnN0UGF5bG9hZE5hbWUCAAAACGxvY2tlZEF0AwkAAGcAAAACBQAAAAZoZWlnaHQFAAAAFWZpcnN0UGF5bG9hZEFzSW50ZWdlcgkAAGcAAAACAAAAAAAAAAAFCQAAZQAAAAIFAAAABmhlaWdodAUAAAAVZmlyc3RQYXlsb2FkQXNJbnRlZ2VyBwcEAAAAGnBsYXllcnNQdWJsaWNLZXlJc1Byb3ZpZGVkCQAAAAAAAAIFAAAAEXNlY29uZFBheWxvYWROYW1lAgAAABBwbGF5ZXJzUHVibGljS2V5AwMDAwUAAAAOc2VuZGVySXNTZXJ2ZXIFAAAAC2RhdGFGZWVJc09rBwUAAAAXdmFsaWRMb2NrZWRBdElzUHJvdmlkZWQHBQAAABpwbGF5ZXJzUHVibGljS2V5SXNQcm92aWRlZAcJAAAAAAAAAgUAAAALcGF5bG9hZFNpemUAAAAAAAAAAAIHAwUAAAANaXNMb2NrZWRTdGF0ZQQAAAAcdmFsaWRQbGF5ZXJzQ2hvaWNlSXNQcm92aWRlZAMJAAAAAAAAAgUAAAAQZmlyc3RQYXlsb2FkTmFtZQIAAAANcGxheWVyc0Nob2ljZQMJAABnAAAAAgUAAAAVZmlyc3RQYXlsb2FkQXNJbnRlZ2VyAAAAAAAAAAAACQAAZgAAAAIFAAAACmJveGVzQ291bnQFAAAAFWZpcnN0UGF5bG9hZEFzSW50ZWdlcgcHAwMDBQAAAA5zZW5kZXJJc1BsYXllcgUAAAALZGF0YUZlZUlzT2sHBQAAABx2YWxpZFBsYXllcnNDaG9pY2VJc1Byb3ZpZGVkBwkAAAAAAAACBQAAAAtwYXlsb2FkU2l6ZQAAAAAAAAAAAQcDBQAAAB5pc1BsYXllclJldmVhbGVkSGlzQ2hvaWNlU3RhdGUEAAAAHHZhbGlkU2VydmVyQ2hvaWNlV2FzUHJvdmlkZWQDCQAAAAAAAAIFAAAAEGZpcnN0UGF5bG9hZE5hbWUCAAAADXNlcnZlcnNDaG9pY2UDCQAAZwAAAAIFAAAAFWZpcnN0UGF5bG9hZEFzSW50ZWdlcgAAAAAAAAAAAAkAAGYAAAACBQAAAApib3hlc0NvdW50BQAAABVmaXJzdFBheWxvYWRBc0ludGVnZXIHBwQAAAAWc2VydmVyc1NhbHRXYXNQcm92aWRlZAkAAAAAAAACBQAAABFzZWNvbmRQYXlsb2FkTmFtZQIAAAALc2VydmVyc1NhbHQEAAAAImNob2ljZUFuZFNhbHRNYXRjaGVzSGFyZGNvZGVkVmFsdWUJAAAAAAAAAgkAAfUAAAABCQABmwAAAAEJAAEsAAAAAgkAAaQAAAABBQAAABVmaXJzdFBheWxvYWRBc0ludGVnZXIFAAAAFXNlY29uZFBheWxvYWRBc1N0cmluZwUAAAAWZW5jdHlwdGVkU2VydmVyc0Nob2ljZQMDAwMDBQAAAA5zZW5kZXJJc1NlcnZlcgUAAAALZGF0YUZlZUlzT2sHBQAAABx2YWxpZFNlcnZlckNob2ljZVdhc1Byb3ZpZGVkBwUAAAAWc2VydmVyc1NhbHRXYXNQcm92aWRlZAcFAAAAImNob2ljZUFuZFNhbHRNYXRjaGVzSGFyZGNvZGVkVmFsdWUHCQAAAAAAAAIFAAAAC3BheWxvYWRTaXplAAAAAAAAAAACBwcDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAE1RyYW5zZmVyVHJhbnNhY3Rpb24EAAAAA3R0eAUAAAAHJG1hdGNoMAQAAAAPdHJhbnNmZXJGZWVJc09rCQAAAAAAAAIIBQAAAAN0dHgAAAADZmVlBQAAABZ0cmFuc2ZlclRyYW5zYWN0aW9uRmVlAwUAAAAZaXNXaW5uZXJJc0RldGVybWluZWRTdGF0ZQQAAAANd2lubmVyQWRkcmVzcwMJAAAAAAAAAgUAAAARZGF0YVNlcnZlcnNDaG9pY2UFAAAAEWRhdGFQbGF5ZXJzQ2hvaWNlBQAAAA5wbGF5ZXJzQWRkcmVzcwUAAAAOc2VydmVyc0FkZHJlc3MEAAAAEXByaXplR29lc1RvV2lubmVyCQAAAAAAAAIIBQAAAAN0dHgAAAAJcmVjaXBpZW50BQAAAA13aW5uZXJBZGRyZXNzAwMFAAAAD3RyYW5zZmVyRmVlSXNPawUAAAARcHJpemVHb2VzVG9XaW5uZXIHCQAAAAAAAAIIBQAAAAN0dHgAAAAGYW1vdW50CQAAZAAAAAIFAAAADHBsYXllcnNQcml6ZQUAAAAIZG9uYXRpb24HAwUAAAAdaXNTZXJ2ZXJEZWNpZGVkTm90VG9QbGF5U3RhdGUEAAAAEXJlY2lwaWVudElzUGxheWVyCQAAAAAAAAIIBQAAAAN0dHgAAAAJcmVjaXBpZW50BQAAAA5wbGF5ZXJzQWRkcmVzcwMDBQAAAA90cmFuc2ZlckZlZUlzT2sFAAAAEXJlY2lwaWVudElzUGxheWVyBwkAAAAAAAACCAUAAAADdHR4AAAABmFtb3VudAkAAGQAAAACBQAAAAxwbGF5ZXJzUHJpemUFAAAACGRvbmF0aW9uBwMFAAAAJWlzUGxheWVyRGVjaWRlZE5vdFRvU2VuZERvbmF0aW9uU3RhdGUEAAAAEXJlY2lwaWVudElzU2VydmVyCQAAAAAAAAIIBQAAAAN0dHgAAAAJcmVjaXBpZW50BQAAAA5zZXJ2ZXJzQWRkcmVzcwMDBQAAAA90cmFuc2ZlckZlZUlzT2sFAAAAEXJlY2lwaWVudElzU2VydmVyBwkAAAAAAAACCAUAAAADdHR4AAAABmFtb3VudAUAAAAMcGxheWVyc1ByaXplBwcHOPut3Q==", NewCatalogueV2(), 1368},
		{`guess.ride`, "AgQAAAAEdGhpcwkBAAAAB2V4dHJhY3QAAAABCAUAAAACdHgAAAAGc2VuZGVyBAAAAAckbWF0Y2gwBQAAAAJ0eAMJAAABAAAAAgUAAAAHJG1hdGNoMAIAAAATVHJhbnNmZXJUcmFuc2FjdGlvbgQAAAABdAUAAAAHJG1hdGNoMAQAAAANY29ycmVjdEFuc3dlcgkBAAAAB2V4dHJhY3QAAAABCQAEHAAAAAIFAAAABHRoaXMCAAAADWhhc2hlZCBhbnN3ZXIEAAAABmFuc3dlcgkAAfUAAAABCAUAAAABdAAAAAphdHRhY2htZW50AwkAAAAAAAACBQAAAA1jb3JyZWN0QW5zd2VyBQAAAAZhbnN3ZXIJAQAAAAEhAAAAAQkBAAAACWlzRGVmaW5lZAAAAAEIBQAAAAF0AAAAB2Fzc2V0SWQHAwMJAAABAAAAAgUAAAAHJG1hdGNoMAIAAAAPRGF0YVRyYW5zYWN0aW9uBgkAAAEAAAACBQAAAAckbWF0Y2gwAgAAABRTZXRTY3JpcHRUcmFuc2FjdGlvbgQAAAABcwUAAAAHJG1hdGNoMAkAAfQAAAADCAUAAAABcwAAAAlib2R5Qnl0ZXMJAAGRAAAAAggFAAAAAXMAAAAGcHJvb2ZzAAAAAAAAAAAACAUAAAABcwAAAA9zZW5kZXJQdWJsaWNLZXkHnYrj7g==", NewCatalogueV2(), 237},
		{`Multisig.ride`, "AgQAAAALYWxpY2VQdWJLZXkBAAAAID3+K0HJI42oXrHhtHFpHijU5PC4nn1fIFVsJp5UWrYABAAAAAlib2JQdWJLZXkBAAAAIBO1uieokBahePoeVqt4/usbhaXRq+i5EvtfsdBILNtuBAAAAAxjb29wZXJQdWJLZXkBAAAAIOfM/qkwkfi4pdngdn18n5yxNwCrBOBC3ihWaFg4gV4yBAAAAAthbGljZVNpZ25lZAMJAAH0AAAAAwgFAAAAAnR4AAAACWJvZHlCeXRlcwkAAZEAAAACCAUAAAACdHgAAAAGcHJvb2ZzAAAAAAAAAAAABQAAAAthbGljZVB1YktleQAAAAAAAAAAAQAAAAAAAAAAAAQAAAAJYm9iU2lnbmVkAwkAAfQAAAADCAUAAAACdHgAAAAJYm9keUJ5dGVzCQABkQAAAAIIBQAAAAJ0eAAAAAZwcm9vZnMAAAAAAAAAAAEFAAAACWJvYlB1YktleQAAAAAAAAAAAQAAAAAAAAAAAAQAAAAMY29vcGVyU2lnbmVkAwkAAfQAAAADCAUAAAACdHgAAAAJYm9keUJ5dGVzCQABkQAAAAIIBQAAAAJ0eAAAAAZwcm9vZnMAAAAAAAAAAAIFAAAADGNvb3BlclB1YktleQAAAAAAAAAAAQAAAAAAAAAAAAkAAGcAAAACCQAAZAAAAAIJAABkAAAAAgUAAAALYWxpY2VTaWduZWQFAAAACWJvYlNpZ25lZAUAAAAMY29vcGVyU2lnbmVkAAAAAAAAAAACqFBMLg==", NewCatalogueV2(), 388},
		{`AuthorizedTrader.ride`, "AgQAAAAPdHJhZGVyUHVibGljS2V5AQAAACAF+j8WBUppk2Gd7LGAEtbrHG3NeWfWUsxIsUc0+q0zfwQAAAAOb3duZXJQdWJsaWNLZXkBAAAAIDahakAL6O7oXCsJB8m9Hji5oezJYYtaVEq8FwLm00hdBAAAAAthbW91bnRBc3NldAEAAAAgbPZB9HxAkx+yFVJT8t7D7SNvLfqtrBRQ6gKzMgtYuuIEAAAAEG1hdGNoZXJQdWJsaWNLZXkBAAAAIGRDQuXcp/doeH4AWAJKWAvM8pffT3ZEKdvTPqW98d8CBAAAAAckbWF0Y2gwBQAAAAJ0eAMJAAABAAAAAgUAAAAHJG1hdGNoMAIAAAAFT3JkZXIEAAAAAW8FAAAAByRtYXRjaDAEAAAAEWlzV2F2ZXNQcmljZUFzc2V0CQEAAAABIQAAAAEJAQAAAAlpc0RlZmluZWQAAAABCAgFAAAAAW8AAAAJYXNzZXRQYWlyAAAACnByaWNlQXNzZXQEAAAACXJpZ2h0UGFpcgMJAAAAAAAAAggIBQAAAAFvAAAACWFzc2V0UGFpcgAAAAthbW91bnRBc3NldAUAAAALYW1vdW50QXNzZXQFAAAAEWlzV2F2ZXNQcmljZUFzc2V0BwMDAwkAAfQAAAADCAUAAAABbwAAAAlib2R5Qnl0ZXMJAAGRAAAAAggFAAAAAW8AAAAGcHJvb2ZzAAAAAAAAAAAABQAAAA90cmFkZXJQdWJsaWNLZXkFAAAACXJpZ2h0UGFpcgcJAABmAAAAAgkAAGgAAAACAAAAAAAAAAB4AAAAAAAAAAPoCQAAZQAAAAIIBQAAAAFvAAAACmV4cGlyYXRpb24IBQAAAAFvAAAACXRpbWVzdGFtcAcJAAAAAAAAAggFAAAAAW8AAAAQbWF0Y2hlclB1YmxpY0tleQUAAAAQbWF0Y2hlclB1YmxpY0tleQcDAwkAAAEAAAACBQAAAAckbWF0Y2gwAgAAABNUcmFuc2ZlclRyYW5zYWN0aW9uBgkAAAEAAAACBQAAAAckbWF0Y2gwAgAAABRTZXRTY3JpcHRUcmFuc2FjdGlvbgQAAAABdAUAAAAHJG1hdGNoMAkAAfQAAAADCAUAAAABdAAAAAlib2R5Qnl0ZXMJAAGRAAAAAggFAAAAAXQAAAAGcHJvb2ZzAAAAAAAAAAAABQAAAA5vd25lclB1YmxpY0tleQfWPv6I", NewCatalogueV2(), 254},
		{`let a = unit; let b = unit; let c = unit; let d = unit; let x = if true then a else b; let y = if false then c else d; x == y`, "AwQAAAABYQUAAAAEdW5pdAQAAAABYgUAAAAEdW5pdAQAAAABYwUAAAAEdW5pdAQAAAABZAUAAAAEdW5pdAQAAAABeAMGBQAAAAFhBQAAAAFiBAAAAAF5AwcFAAAAAWMFAAAAAWQJAAAAAAAAAgUAAAABeAUAAAABeei/I5Y=", NewCatalogueV3(), 47},
		{`match tx {case dt: DataTransaction => !isDefined(getInteger(dt.data, "xxx")) case _ => false }`, "AgQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAD0RhdGFUcmFuc2FjdGlvbgQAAAACZHQFAAAAByRtYXRjaDAJAQAAAAEhAAAAAQkBAAAACWlzRGVmaW5lZAAAAAEJAAQQAAAAAggFAAAAAmR0AAAABGRhdGECAAAAA3h4eAeneNyG", NewCatalogueV2(), 80},
		{`match tx {case dt: DataTransaction => !isDefined(getInteger(dt.data, "xxx")) case _ => false }`, "AgQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAD0RhdGFUcmFuc2FjdGlvbgQAAAACZHQFAAAAByRtYXRjaDAJAQAAAAEhAAAAAQkBAAAACWlzRGVmaW5lZAAAAAEJAAQQAAAAAggFAAAAAmR0AAAABGRhdGECAAAAA3h4eAeneNyG", NewCatalogueV3(), 36},
		{`let totalMoney = if (isDefined(getInteger(tx.sender, "totalMoney"))) then extract(getInteger(tx.sender, "totalMoney")) else 0; totalMoney != 0`, "AwQAAAAKdG90YWxNb25leQMJAQAAAAlpc0RlZmluZWQAAAABCQAEGgAAAAIIBQAAAAJ0eAAAAAZzZW5kZXICAAAACnRvdGFsTW9uZXkJAQAAAAdleHRyYWN0AAAAAQkABBoAAAACCAUAAAACdHgAAAAGc2VuZGVyAgAAAAp0b3RhbE1vbmV5AAAAAAAAAAAACQEAAAACIT0AAAACBQAAAAp0b3RhbE1vbmV5AAAAAAAAAAAAdjfmag==", NewCatalogueV3(), 234},
		{`let s = size(toString(1000)); s != 0`, "AwQAAAABcwkAATEAAAABCQABpAAAAAEAAAAAAAAAA+gJAQAAAAIhPQAAAAIFAAAAAXMAAAAAAAAAAACmTwkf", NewCatalogueV3(), 12},
		{`let a = "A"; let x = a + if true then {let c = "C"; c} else {let b = "B"; b}; x == "ABC"`, "AwQAAAABYQIAAAABQQQAAAABeAkAASwAAAACBQAAAAFhAwYEAAAAAWMCAAAAAUMFAAAAAWMEAAAAAWICAAAAAUIFAAAAAWIJAAAAAAAAAgUAAAABeAIAAAADQUJDncKWCg==", NewCatalogueV3(), 37},
		{`let a = addressFromString("cafebebedeadbeef"); a == Address(base16'cafebebedeadbeef')`, "AwQAAAABYQkBAAAAEWFkZHJlc3NGcm9tU3RyaW5nAAAAAQIAAAAQY2FmZWJlYmVkZWFkYmVlZgkAAAAAAAACBQAAAAFhCQEAAAAHQWRkcmVzcwAAAAEBAAAACMr+vr7erb7v7Rvb0w==", NewCatalogueV3(), 135},
		{`match tx {case transfer: TransferTransaction => sigVerify(tx.bodyBytes, tx.proofs[0], tx.senderPublicKey)case _ => false}`, "AwQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAE1RyYW5zZmVyVHJhbnNhY3Rpb24EAAAACHRyYW5zZmVyBQAAAAckbWF0Y2gwCQAB9AAAAAMIBQAAAAJ0eAAAAAlib2R5Qnl0ZXMJAAGRAAAAAggFAAAAAnR4AAAABnByb29mcwAAAAAAAAAAAAgFAAAAAnR4AAAAD3NlbmRlclB1YmxpY0tleQeNAjRw", NewCatalogueV3(), 132},
		{`match tx {case IssueTransaction => true case TransferTransaction => false case ReissueTransaction => true case BurnTransaction => false case ExchangeTransaction => true case SetScriptTransaction => false case SetAssetScriptTransaction => true case SponsorFeeTransaction => false case PaymentTransaction => true case GenesisTransaction => false case _ => false}`, "AwQAAAAHJG1hdGNoMAUAAAACdHgEAAAAEElzc3VlVHJhbnNhY3Rpb24FAAAAByRtYXRjaDAGeIskSg==", NewCatalogueV3(), 11},
		{`match (tx) {case t: TransferTransaction => (t.amount - 1) * 2 - 3 - t.fee case _ => 0} == 0`, "AgkAAAAAAAACBAAAAAckbWF0Y2gwBQAAAAJ0eAMJAAABAAAAAgUAAAAHJG1hdGNoMAIAAAATVHJhbnNmZXJUcmFuc2FjdGlvbgQAAAABdAUAAAAHJG1hdGNoMAkAAGUAAAACCQAAZQAAAAIJAABoAAAAAgkAAGUAAAACCAUAAAABdAAAAAZhbW91bnQAAAAAAAAAAAEAAAAAAAAAAAIAAAAAAAAAAAMIBQAAAAF0AAAAA2ZlZQAAAAAAAAAAAAAAAAAAAAAAADdxFIQ=", NewCatalogueV2(), 36},
		{`match tx {case tx: TransferTransaction => isDefined(tx.feeAssetId) case _ => false}`, "AgQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAE1RyYW5zZmVyVHJhbnNhY3Rpb24EAAAAAnR4BQAAAAckbWF0Y2gwCQEAAAAJaXNEZWZpbmVkAAAAAQgFAAAAAnR4AAAACmZlZUFzc2V0SWQHXC5tqw==", NewCatalogueV2(), 58},
		{`match tx {case t: TransferTransaction => isDefined(t.feeAssetId) case _ => false}`, "AgQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAE1RyYW5zZmVyVHJhbnNhY3Rpb24EAAAAAXQFAAAAByRtYXRjaDAJAQAAAAlpc0RlZmluZWQAAAABCAUAAAABdAAAAApmZWVBc3NldElkB9Agf0U=", NewCatalogueV2(), 58},
		{`match tx {case t: TransferTransaction => isDefined(t.feeAssetId) case _ => false}`, "AgQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAE1RyYW5zZmVyVHJhbnNhY3Rpb24EAAAAAXQFAAAAByRtYXRjaDAJAQAAAAlpc0RlZmluZWQAAAABCAUAAAABdAAAAApmZWVBc3NldElkB9Agf0U=", NewCatalogueV2(), 58},
		{`let a = addressFromStringValue("3P2USE3iYK5w7jNahAUHTytNbVRccGZwQH3"); let i = getInteger(a, "integer"); let x = match i {case i: Int => i case _ => 0}; x == 100500`, "AwQAAAABYQkBAAAAHEBleHRyVXNlcihhZGRyZXNzRnJvbVN0cmluZykAAAABAgAAACMzUDJVU0UzaVlLNXc3ak5haEFVSFR5dE5iVlJjY0dad1FIMwQAAAABaQkABBoAAAACBQAAAAFhAgAAAAdpbnRlZ2VyBAAAAAF4BAAAAAckbWF0Y2gwBQAAAAFpAwkAAAEAAAACBQAAAAckbWF0Y2gwAgAAAANJbnQEAAAAAWkFAAAAByRtYXRjaDAFAAAAAWkAAAAAAAAAAAAJAAAAAAAAAgUAAAABeAAAAAAAAAGIlKWtlDk=", NewCatalogueV3(), 268},
		{`func first(a: Int, b: Int) = {let x = a + b; x}; first(1, 2) == 0`, "AwoBAAAABWZpcnN0AAAAAgAAAAFhAAAAAWIEAAAAAXgJAABkAAAAAgUAAAABYQUAAAABYgUAAAABeAkAAAAAAAACCQEAAAAFZmlyc3QAAAACAAAAAAAAAAABAAAAAAAAAAACAAAAAAAAAAAAm+QHtw==", NewCatalogueV3(), 33},
		{`func f(a: Int) = 1; func g(a: Int) = 2; f(g(1)) == 0`, "AwoBAAAAAWYAAAABAAAAAWEAAAAAAAAAAAEKAQAAAAFnAAAAAQAAAAFhAAAAAAAAAAACCQAAAAAAAAIJAQAAAAFmAAAAAQkBAAAAAWcAAAABAAAAAAAAAAABAAAAAAAAAAAAT0GP5g==", NewCatalogueV3(), 25},
		{`func inc(y: Int) = y + 1; let xxx = 5; inc(xxx) == 1`, "AwoBAAAAA2luYwAAAAEAAAABeQkAAGQAAAACBQAAAAF5AAAAAAAAAAABBAAAAAN4eHgAAAAAAAAAAAUJAAAAAAAAAgkBAAAAA2luYwAAAAEFAAAAA3h4eAAAAAAAAAAAAbumbXA=", NewCatalogueV3(), 25},
		{`func f() = {func f() = {func f() = {1}; f()}; f()}; f() == 0`, "AwoBAAAAAWYAAAAACgEAAAABZgAAAAAKAQAAAAFmAAAAAAAAAAAAAAAAAQkBAAAAAWYAAAAACQEAAAABZgAAAAAJAAAAAAAAAgkBAAAAAWYAAAAAAAAAAAAAAAAAYYLPvQ==", NewCatalogueV3(), 18},
		{`let me = addressFromStringValue(""); func get() = getStringValue(this, ""); get() + get() + get() == ""`, "AwQAAAACbWUJAQAAABxAZXh0clVzZXIoYWRkcmVzc0Zyb21TdHJpbmcpAAAAAQIAAAAACgEAAAADZ2V0AAAAAAkBAAAAEUBleHRyTmF0aXZlKDEwNTMpAAAAAgUAAAAEdGhpcwIAAAAACQAAAAAAAAIJAAEsAAAAAgkAASwAAAACCQEAAAADZ2V0AAAAAAkBAAAAA2dldAAAAAAJAQAAAANnZXQAAAAAAgAAAACvuyuT", NewCatalogueV3(), 341},
	} {
		r, err := reader.NewReaderFromBase64(test.script)
		require.NoError(t, err, test.code)
		script, err := ast.BuildScript(r)
		require.NoError(t, err, test.code)
		e1 := NewEstimator(1, test.catalogue, ast.VariablesV3())
		e2 := NewEstimator(2, test.catalogue, ast.VariablesV3())
		cost1, err := e1.EstimateVerifier(script)
		require.NoError(t, err, test.code)
		cost2, err := e2.EstimateVerifier(script)
		require.NoError(t, err, test.code)
		assert.Equal(t, int(test.cost), int(cost1.Verifier), test.code)
		assert.Equal(t, int(test.cost), int(cost2.Verifier), test.code)
	}
}

func TestEstimatorV2(t *testing.T) {

}

func TestEstimatorV3(t *testing.T) {

}
