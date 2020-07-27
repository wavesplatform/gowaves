package fride

func catalogueV12() map[string]int {
	c := make(map[string]int)
	c["0"] = 1
	c["1"] = 1
	c["2"] = 1
	c["100"] = 1
	c["101"] = 1
	c["102"] = 1
	c["103"] = 1
	c["104"] = 1
	c["105"] = 1
	c["106"] = 1
	c["107"] = 1
	c["200"] = 1
	c["201"] = 1
	c["202"] = 1
	c["203"] = 10
	c["300"] = 10
	c["303"] = 1
	c["304"] = 1
	c["305"] = 1
	c["400"] = 2
	c["401"] = 2
	c["410"] = 1
	c["411"] = 1
	c["412"] = 1
	c["420"] = 1
	c["421"] = 1
	c["500"] = 100
	c["501"] = 10
	c["502"] = 10
	c["503"] = 10
	c["600"] = 10
	c["601"] = 10
	c["602"] = 10
	c["603"] = 10
	c["1000"] = 100
	c["1001"] = 100
	c["1003"] = 100
	c["1040"] = 10
	c["1041"] = 10
	c["1042"] = 10
	c["1043"] = 10
	c["1050"] = 100
	c["1051"] = 100
	c["1052"] = 100
	c["1053"] = 100
	c["1060"] = 100

	c["throw"] = 2
	c["addressFromString"] = 124
	c["!="] = 26
	c["isDefined"] = 35
	c["extract"] = 13
	c["dropRightBytes"] = 19
	c["takeRightBytes"] = 19
	c["takeRight"] = 19
	c["dropRight"] = 19
	c["!"] = 11
	c["-"] = 9
	c["getInteger"] = 10
	c["getBoolean"] = 10
	c["getBinary"] = 10
	c["getString"] = 10
	c["addressFromPublicKey"] = 82
	c["wavesBalance"] = 109

	// Type constructors, type constructor cost equals to the number of it's parameters
	c["Address"] = 1
	c["Alias"] = 1
	c["DataEntry"] = 2
	c["DataTransaction"] = 9
	c["AssetPair"] = 2
	return c
}

func catalogueV3() map[string]int {
	c := catalogueV12()
	// New native functions
	c["108"] = 100
	c["109"] = 100
	c["504"] = 300
	c["604"] = 10
	c["605"] = 10
	c["1004"] = 100
	c["1005"] = 100
	c["1006"] = 100
	c["700"] = 30
	c["1061"] = 10
	c["1070"] = 100
	c["1100"] = 2
	c["1200"] = 20
	c["1201"] = 10
	c["1202"] = 10
	c["1203"] = 20
	c["1204"] = 20
	c["1205"] = 100
	c["1206"] = 20
	c["1207"] = 20
	c["1208"] = 20

	// Cost updates for existing user functions
	c["throw"] = 1
	c["isDefined"] = 1
	c["!="] = 1
	c["!"] = 1
	c["-"] = 1

	// Constructors for simple types
	c["Ceiling"] = 0
	c["Floor"] = 0
	c["HalfEven"] = 0
	c["Down"] = 0
	c["Up"] = 0
	c["HalfUp"] = 0
	c["HalfDown"] = 0
	c["NoAlg"] = 0
	c["Md5"] = 0
	c["Sha1"] = 0
	c["Sha224"] = 0
	c["Sha256"] = 0
	c["Sha384"] = 0
	c["Sha512"] = 0
	c["Sha3224"] = 0
	c["Sha3256"] = 0
	c["Sha3384"] = 0
	c["Sha3512"] = 0
	c["Unit"] = 0

	// New user functions
	c["@extrNative(1040)"] = 10
	c["@extrNative(1041)"] = 10
	c["@extrNative(1042)"] = 10
	c["@extrNative(1043)"] = 10
	c["@extrNative(1050)"] = 100
	c["@extrNative(1051)"] = 100
	c["@extrNative(1052)"] = 100
	c["@extrNative(1053)"] = 100
	c["@extrUser(getInteger)"] = 10
	c["@extrUser(getBoolean)"] = 10
	c["@extrUser(getBinary)"] = 10
	c["@extrUser(getString)"] = 10
	c["@extrUser(addressFromString)"] = 124
	c["parseIntValue"] = 20
	c["value"] = 13
	c["valueOrErrorMessage"] = 13

	c["WriteSet"] = 1
	c["TransferSet"] = 1
	c["ScriptTransfer"] = 3
	c["ScriptResult"] = 2
	return c
}

func catalogueV4() map[string]int {
	c := catalogueV3()
	c["IntegerEntry"] = 2
	c["BooleanEntry"] = 2
	c["BinaryEntry"] = 2
	c["StringEntry"] = 2
	c["DeleteEntry"] = 1
	c["Issue"] = 7
	c["Reissue"] = 3
	c["Burn"] = 2
	c["contains"] = 20
	c["valueOrElse"] = 13
	c["405"] = 10
	c["406"] = 3
	c["407"] = 3
	c["701"] = 30
	c["800"] = 3900
	c["900"] = 70
	c["1070"] = 5
	c["1080"] = 10
	c["1100"] = 2
	c["1101"] = 3
	c["1102"] = 10
	c["1103"] = 5
	c["1104"] = 5
	c["2400"] = 1900
	c["2401"] = 2000
	c["2402"] = 2150
	c["2403"] = 2300
	c["2404"] = 2450
	c["2405"] = 2550
	c["2406"] = 2700
	c["2407"] = 2900
	c["2408"] = 3000
	c["2409"] = 3150
	c["2410"] = 3250
	c["2411"] = 3400
	c["2412"] = 3500
	c["2413"] = 3650
	c["2414"] = 3750
	c["2500"] = 100
	c["2501"] = 110
	c["2502"] = 125
	c["2503"] = 150
	c["2600"] = 100
	c["2601"] = 500
	c["2602"] = 550
	c["2603"] = 625
	c["2700"] = 10
	c["2701"] = 25
	c["2702"] = 50
	c["2703"] = 100
	c["2800"] = 10
	c["2801"] = 25
	c["2802"] = 50
	c["2803"] = 100
	c["2900"] = 10
	c["2901"] = 25
	c["2902"] = 50
	c["2903"] = 100
	return c
}
