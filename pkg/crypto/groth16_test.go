package crypto

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroth16VerifyBLS(t *testing.T) {
	for i, test := range []struct {
		vk     string
		proof  string
		inputs string
		ok     bool
	}{
		{
			vk: "hwk883gUlTKCyXYA6XWZa8H9/xKIYZaJ0xEs0M5hQOMxiGpxocuX/8maSDmeCk3bo5ViaDBdO7ZBxAhLSe5k/5TFQyF5Lv7KN2tL" +
				"KnwgoWMqB16OL8WdbePIwTCuPtJNAFKoTZylLDbSf02kckMcZQDPF9iGh+JC99Pio74vDpwTEjUx5tQ99gNQwxULtztsqDRsPnEv" +
				"KvLmsxHt8LQVBkEBm2PBJFY+OXf1MNW021viDBpR10mX4WQ6zrsGL5L0GY4cwf4tlbh+Obit+LnN/SQTnREf8fPpdKZ1sa/ui3pG" +
				"i8lMT6io4D7Ujlwx2RdCkBF+isfMf77HCEGsZANw0hSrO2FGg14Sl26xLAIohdaW8O7gEaag8JdVAZ3OVLd5Df1NkZBEr753Xb8W" +
				"waXsJjE7qxwINL1KdqA4+EiYW4edb7+a9bbBeOPtb67ZxmFqgyTNS/4obxahezNkjk00ytswsENg//Ee6dWBJZyLH+QGsaU2jO/W" +
				"4WvRyZhmKKPdipOhiz4Rlrd2XYgsfHsfWf5v4GOTL+13ZB24dW1/m39n2woJ+v686fXbNW85XP/r",
			proof: "lvQLU/KqgFhsLkt/5C/scqs7nWR+eYtyPdWiLVBux9GblT4AhHYMdCgwQfSJcudvsgV6fXoK+DUSRgJ++Nqt+Wvb7GlYlHpxC" +
				"ysQhz26TTu8Nyo7zpmVPH92+UYmbvbQCSvX2BhWtvkfHmqDVjmSIQ4RUMfeveA1KZbSf999NE4qKK8Do+8oXcmTM4LZVmh1rlyqz" +
				"nIdFXPN7x3pD4E0gb6/y69xtWMChv9654FMg05bAdueKt9uA4BEcAbpkdHF",
			inputs: "LcMT3OOlkHLzJBKCKjjzzVMg+r+FVgd52LlhZPB4RFg=",
			ok:     true,
		},
		{
			vk: "hwk883gUlTKCyXYA6XWZa8H9/xKIYZaJ0xEs0M5hQOMxiGpxocuX/8maSDmeCk3bo5ViaDBdO7ZBxAhLSe5k/5TFQyF5Lv7KN2tL" +
				"KnwgoWMqB16OL8WdbePIwTCuPtJNAFKoTZylLDbSf02kckMcZQDPF9iGh+JC99Pio74vDpwTEjUx5tQ99gNQwxULtztsqDRsPnEv" +
				"KvLmsxHt8LQVBkEBm2PBJFY+OXf1MNW021viDBpR10mX4WQ6zrsGL5L0GY4cwf4tlbh+Obit+LnN/SQTnREf8fPpdKZ1sa/ui3pG" +
				"i8lMT6io4D7Ujlwx2RdCkBF+isfMf77HCEGsZANw0hSrO2FGg14Sl26xLAIohdaW8O7gEaag8JdVAZ3OVLd5Df1NkZBEr753Xb8W" +
				"waXsJjE7qxwINL1KdqA4+EiYW4edb7+a9bbBeOPtb67ZxmFqgyTNS/4obxahezNkjk00ytswsENg//Ee6dWBJZyLH+QGsaU2jO/W" +
				"4WvRyZhmKKPdipOhiz4Rlrd2XYgsfHsfWf5v4GOTL+13ZB24dW1/m39n2woJ+v686fXbNW85XP/r",
			proof: "lvQLU/KqgFhsLkt/5C/scqs7nWR+eYtyPdWiLVBux9GblT4AhHYMdCgwQfSJcudvsgV6fXoK+DUSRgJ++Nqt+Wvb7GlYlHpxC" +
				"ysQhz26TTu8Nyo7zpmVPH92+UYmbvbQCSvX2BhWtvkfHmqDVjmSIQ4RUMfeveA1KZbSf999NE4qKK8Do+8oXcmTM4LZVmh1rlyqz" +
				"nIdFXPN7x3pD4E0gb6/y69xtWMChv9654FMg05bAdueKt9uA4BEcAbpkdHF",
			inputs: "cmzVCcRVnckw3QUPhmG4Bkppeg4K50oDQwQ9EH+Fq1s=",
			ok:     false,
		},
		{
			vk: "kYYCAS8vM2T99GeCr4toQ+iQzvl5fI89mPrncYqx3C1d75BQbFk8LMtcnLWwntd6knkzSwcsialcheg69eZYPK8EzKRVI5FrRHKi" +
				"8rgB+R5jyPV70ejmYEx1neTmfYKODRmARr/ld6pZTzBWYDfrCkiS1QB+3q3M08OQgYcLzs/vjW4epetDCmk0K1CEGcWdh7yLzdqr" +
				"7HHQNOpZI8mdj/7lR0IBqB9zvRfyTr+guUG22kZo4y2KINDp272xGglKEeTglTxyDUriZJNF/+T6F8w70MR/rV+flvuo6EJ0+HA+" +
				"A2ZnBbTjOIl9wjisBV+0jgld4oAppAOzvQ7eoIx2tbuuKVSdbJm65KDxl/T+boaYnjRm3omdETYnYRk3HAhrAeWpefX+dM/k7Prc" +
				"heInnxHUyjzSzqlN03xYjg28kdda9FZJaVsQKqdEJ/St9ivXlp7+dPDIOfm77haSFnvr33VwYH/KbIalfOJPRvBLzqlHD8BxunNe" +
				"bMr6Gr6S+u+n",
			proof: "sStVLdyxqInmv76iaNnRFB464lGq48iVeqYWSi2linE9DST0fTNhxSnvSXAoPpt8tFsanj5vPafC+ij/Fh98dOUlMbO42bf28" +
				"0pOZ4lm+zr63AWUpOOIugST+S6pq9zeB0OHp2NY8XFmriOEKhxeabhuV89ljqCDjlhXBeNZwM5zti4zg89Hd8TbKcw46jAsjIJe2" +
				"Siw3Th7ELQQKR5ucX50f0GISmnOSceePPdvjbGJ8fSFOnSmSp8dK7uyehrU",
			inputs: "",
			ok:     true,
		},
		{
			vk: "mY//hEITCBCZUJUN/wsOlw1iUSSOESL6PFSbN1abGK80t5jPNICNlPuSorio4mmWpf+4uOyv3gPZe54SYGM4pfhteqJpwFQxdlpw" +
				"XWyYxMTNaSLDj8VtSn/EJaSu+P6nFmWsda3mTYUPYMZzWE4hMqpDgFPcJhw3prArMThDPbR3Hx7E6NRAAR0LqcrdtsbDqu2T0tto" +
				"1rpnFILdvHL4PqEUfTmF2mkM+DKj7lKwvvZUbukqBwLrnnbdfyqZJryzGAMIa2JvMEMYszGsYyiPXZvYx6Luk54oWOlOrwEKrCY4" +
				"NMPwch6DbFq6KpnNSQwOpgRYCz7wpjk57X+NGJmo85tYKc+TNa1rT4/DxG9v6SHkpXmmPeHhzIIW8MOdkFjxB5o6Qn8Fa0c6Tt6b" +
				"r2gzkrGr1eK5/+RiIgEzVhcRrqdY/p7PLmKXqawrEvIv9QZ3ijytPNwinlC8XdRLO/YvP33PjcI9WSMcHV6POP9KPMo1rngaIPMe" +
				"gKgAvTEouNFKp4v3wAXRXX5xEjwXAmM5wyB/SAOaPPCK/emls9kqolHsaj7nuTTbrvSV8bqzUwzQ",
			proof: "g53N8ecorvG2sDgNv8D7quVhKMIIpdP9Bqk/8gmV5cJ5Rhk9gKvb4F0ll8J/ZZJVqa27OyciJwx6lym6QpVK9q1ASrqio7rD5" +
				"POMDGm64Iay/ixXXn+//F+uKgDXADj9AySri2J1j3qEkqqe3kxKthw94DzAfUBPncHfTPazVtE48AfzB1KWZA7Vf/x/3phYs4ckc" +
				"P7ZrdVViJVLbUgFy543dpKfEH2MD30ZLLYRhw8SatRCyIJuTZcMlluEKG+d",
			inputs: "aZ8tqrOeEJKt4AMqiRF/WJhIKTDC0HeDTgiJVLZ8OEs=",
			ok:     true,
		},
		{
			vk: "tRpqHB4HADuHAUvHTcrzxmq1awdwEBA0GOJfebYTODyUqXBQ7FkYrz1oDvPyx5Z3sUmODSJXAQmAFBVnS2t+Xzf5ZCr1gCtMiJVj" +
				"Q48/nob/SkrS4cTHHjbKIVS9cdD/BG/VDrZvBt/dPqXmdUFyFuTTMrViagR57YRrDmm1qm5LQ/A8VwUBdiArwgRQXH9jsYhgVmfc" +
				"RAjJytrbYeR6ck4ZfmGr6x6akKiBLY4B1l9LaHTyz/6KSM5t8atpuR3HBJZfbBm2/K8nnYTl+mAU/EnIN3YQdUd65Hsd4Gtf6VT2" +
				"qfz6hcrSgHutxR1usIL2kyU9X4Kqjx6I6zYwVbn7PWbiy3OtY277z4ggIqW6AuDgzUeIyG9a4stMeQ07mOV/Ef4faj+eh4GJRKjJ" +
				"m7aUTYJCSAGY6klOXNoEzB54XF4EY5pkMPfW73SmxJi9B0aHkZWDy2tzUlwvxZ/BfsDkUZnt6mI+qdDOtTG6JFItSQZotYGDBm6z" +
				"Pczwo3ZAGpr8gibTE6DjT7GGNDEl26jgAJ3aAdBrf7Yb0vWEYizOJK4SO/Ud+4/WxXDby7xbwlFYkgEtYbMO6PXozhRqDiotJ0Cf" +
				"dSExNHA9A37mR/bpNOKyhArfyvSBIJnUQgOw5wMBq+GOP5n78E99a5rY4FXGUmM3LGdp/CvkGITYf04SWHkZAEueYH96Ys5jrHlI" +
				"ZQA2k9j02Ji+SL82DJFH8LDh77fgh9zh0wAjCAqY7/r72434RDA97bfEZJavRmAENsgflsSVb8d9rQMBpWl3Xkb8mNlUOSf+LAXe" +
				"XYQR42Z4yuUjwAUvk//+imuhsWF8ZCMkpb9wQ/6crVH4E5E3f6If/Mt/DcenWlPNtvu2CJFatc8q31aSdnWhMN8U65SX3DBouDc8" +
				"EXDFd5twy4VWMS5lhY6VbU/lS8T8oyhr+NIpstsKUmSh0EM1rGyUh2PNgIYzoeBznHWagp2WO3nIbNYIcXEROBT8QpqA4Dqzxv66" +
				"5jwajGXmAawRvdZqzLqvCkeujekplZYoV0aXEnYEOIvfF7d4xay3qkx2NspooM4HeZpiHknIWkUVhGVJBzBDLjLBjiGBK+TGHfH8" +
				"Oadexhdet7ExyIWibSmamWQvffZkyl3WnMoVbTQ3lOks4Mca3sU5hp1iMepdu0rKoBh0NXcw9F9hkiggDIkRNINq2rlvUypPiSmp" +
				"8U8tDSMeG0YVSovFlA4DsjBwntJH45NgNbY/Rbu/hfe7QskTkBiTo2A+kmYSH75Uvf2UAXwBAT1PoE0sqtYndF2Kbthl6GylV3j9" +
				"NIKtIzHd/GwleExuM7KlI1H22P78br5zmh8D7V1aFcxPpftQhjch4abXuxEP4ahgfNmthdhoSvQykLhjbmG9BrvwmyaDRd/sHCTe" +
				"SXmLqIybrd6tA8ZLJq2DLzKJEOlmfM9aIihLe/FLndfnTSkNK2et4o8vM3YjAmgOnrAo7JIp",
			proof: "lgFU4Jyo9GdHL7w31u3zXc8RQRnHVarZWNfd0lD45GvvQtwrZ1Y1OKB4T29a79UagPHOdk1S0k0hYAYQyyNAfRUzde1HP8R+2" +
				"dms75gGZEnx2tXexEN+BVjRJfC8PR1lFJa6xvsEx5uSrOZzKmoMfCwcA55SMT5jFo4+KyWg2wP5OnFPx7XTdEKvf5YhpY0krQKiq" +
				"3OUu79EwjNF1xV1+iLxx2KEIyK7RSYxO1BHrKOGOEzxSUK00MA+YVHe+DvW",
			inputs: "aZ8tqrOeEJKt4AMqiRF/WJhIKTDC0HeDTgiJVLZ8OEtiLNj7hflFeVnNXPguxyoqkI/V7pGJtXBpH5N+RswQNA0b23aM33aH" +
				"0HKHOWoGY/T/L7TQzYFGJ3vTLiXDFZg1OVqkGOMvqAgonOrHGi6IgcALyUMyCKlL5BQY23SeILJpYKolybJNwJfbjxpg0Oz+D2fr" +
				"7r9XL1GMvgblu52bVQT1fR8uCRJfSsgA2OGw6k/MpKDCfMcjbR8jnZa8ROEvF4cohm7iV1788Vp2/2bdcEZRQSoaGV8pOmA9Ekqz" +
				"JVRABjkDso40fnQcm2IzjBUOsX+uFExVan56/vl9VZVwB0wnee3Uxiredn0kOayiPB16yimxXCDet+M+0UKjmIlmXYpkrCDrH0dn" +
				"53w+U3OHqMQxPDnUpYBxadM1eI8xWFFxzaLkvega0q0DmEquyY02yiTqo+7Q4qaJVTLgu6/8ekzPxGKRi845NL8gRgaTtM3kidDz" +
				"IQpyODZD0yeEZDY1M+3sUKHcVkhoxTQBTMyKJPc+M5DeBL3uaWMrvxuL6q8+X0xeBt+9kguPUNtIYqUgPAaXvM2i041bWHTJ0dZL" +
				"yDJVOyzGaXRaF4mNkAuh4Et6Zw5PuOpMM2mI1oFKEZj7",
			ok: true,
		},
		{
			vk: "kY4NWaOoYItWtLKVQnxDh+XTsa0Yev5Ae3Q9vlQSKp6+IUtwS7GH5ZrZefmBEwWEqvAtYaSs5qW3riOiiRFoLp7MThW4vCEhK0j8" +
				"BZY5ZM/tnjB7mrLB59kGvzpW8PM/AoQRIWzyvO3Dxxfyj/UQcQRw+KakVRvrFca3Vy2K5cFwxYHwl6PFDM+OmGrlgOCoqZtY1SLO" +
				"d+ovmFOODKiHBZzDZhC/lRfjKVy4LzI7AXDuFn4tlWoT7IsJyy6lYNaWFfLjYZPAsrv1gXJ1NYat5B6E0Pnz5C67u2Uigmlol2D9" +
				"1re3oAqIo+r8kiyFKOSBooG0cMN47zQor6qj0owuxJjn5Ymrcd/FCQ1ud4cKoUlNaGWIekSjxJEB87elMy5oEUlUzVI9ObMm+2SE" +
				"3Udgws7pkMM8fgQUQUqUVyc7sNCE9m/hQzlwtbXrNSS5Pb+6ow7aHMOavjVyaXiS0f6b1pwJpS1yT+K85UA1CLqqxCaEw5+8WAjM" +
				"zBOrKmxBUpYApI4FBAIa/SjeU/wYnljUUMTMfnBfCQ8MS01hFSQZSoPx1do8Zxn5Y3NPgpaomXDfpyVK9Q0U0NkqQqPsk+T+Arox" +
				"QGxq9f/HOX5I5ZibF27dZ32tCbTKo22GgspqtAv2iv06PubySY5lRIEYlCjr5j8Ahl9gFvN+22cIh1iGiuwByhPjGDgP5h78xZXC" +
				"BoJekEYPcI2C0LtBch5pZC/JpS1kF9lBLndodhIlutEr3mkKohR+D/czN/FTdxU2b82QqfZOHc+6rv2biEXy8AdoAMykj1dsIw7/" +
				"d5M8XcgPiUzNko4H6p02Rt2R01MOYboTogaQH8lyU6o8c+iORRGEoZDTq4htC+Qa7AXTodvSmG33IrwJVGOKDMtvWI1VYdhWs32S" +
				"B0W1d+BrFb0ObBGsz+Un7P+V8qerCMqu906BkbjdWmsKbKQBFC8/YDTdSi92rIq1ISUQWn88AgW/q+u6KPxybU5EZgbA+EZwCDB6" +
				"MyBNhHcrAvVFeX+kj1RY1Gx1kzCE3ldsT37sCbayFtyMMbL6gDQCoTadJX/jhs9wgp0dZujwOk0Wefhgy1BUHXl/q+2nXAKPvKml" +
				"i6Wo7/pYr/q13Gcsj7Z7WSKVn4Fm4XfkJD62q6paCxO51BlJQEcnpNPKS7+zjhmQlTRiEryD8ve7KQzk20eb4TgIMR1hI5pnQmjG" +
				"eT56xZySp2nDnYDsqsnXB5uQY8lyf6IYC/PHzEb3rSx91k0ZEu5w5IMrVK8otNzZHrUuM0aPdImpLQJ4qEgvmezORpcUCq4SRp9b" +
				"Gl3/yzXE5tWZgn3Q6kXyjFMhu+foTYy1NV+HJbJI1nYMjeTr3f+RxSphIYWyMZ7sD3RgDzRk5iQqD1J+8rdOIZliObfrmWaro/BB" +
				"xNvd1fPAlFEPiDegBcDaVWHS2A1FPIC9d+DU05vizrBfli6su9rCvSBNVnoDSBF2zeU+2NjXj7ycHYxCuZgl8dBu8FZjvjlDUZCq" +
				"fdq3PszQeo2X55trDJEHeVWaRoIcgiG2hfTN",
			proof: "jqPSA/XKqZDJnRSmM0sJxbrFv7GUcA45QMysIx1xTsI3+2iysF5Tr68565ZuO65qjo2lklZpQo+wtyKSA/56EaKOJZCZhSvDd" +
				"BEdvVYJCjmWusuK5qav7xZO0w5W1qRiEgIdcGUz5V7JHqfRf4xI6/uUD846alyzzNjxQtKErqJbRw6yyBO6j6box363pinjiMTzU" +
				"4w/qltzFuOEpKxy/H3vyH8RcsF24Ou/Rb6vfR7cSLtLwCsf/BMtPcsQfdRK",
			inputs: "aZ8tqrOeEJKt4AMqiRF/WJhIKTDC0HeDTgiJVLZ8OEtiLNj7hflFeVnNXPguxyoqkI/V7pGJtXBpH5N+RswQNA0b23aM33aH" +
				"0HKHOWoGY/T/L7TQzYFGJ3vTLiXDFZg1OVqkGOMvqAgonOrHGi6IgcALyUMyCKlL5BQY23SeILJpYKolybJNwJfbjxpg0Oz+D2fr" +
				"7r9XL1GMvgblu52bVQT1fR8uCRJfSsgA2OGw6k/MpKDCfMcjbR8jnZa8ROEvF4cohm7iV1788Vp2/2bdcEZRQSoaGV8pOmA9Ekqz" +
				"JVRABjkDso40fnQcm2IzjBUOsX+uFExVan56/vl9VZVwB0wnee3Uxiredn0kOayiPB16yimxXCDet+M+0UKjmIlmXYpkrCDrH0dn" +
				"53w+U3OHqMQxPDnUpYBxadM1eI8xWFFxzaLkvega0q0DmEquyY02yiTqo+7Q4qaJVTLgu6/8ekzPxGKRi845NL8gRgaTtM3kidDz" +
				"IQpyODZD0yeEZDY1M+3sUKHcVkhoxTQBTMyKJPc+M5DeBL3uaWMrvxuL6q8+X0xeBt+9kguPUNtIYqUgPAaXvM2i041bWHTJ0dZL" +
				"yDJVOyzGaXRaF4mNkAuh4Et6Zw5PuOpMM2mI1oFKEZj7Xqf/yAmy/Le3GfJnMg5vNgE7QxmVsjuKUP28iN8rdi4=",
			ok: true,
		},
		{
			vk: "pQUlLSBu9HmVa9hB0rEu1weeBv2RKQQ8yCHpwXTHeSkcQqmSOuzednF8o0+MdyNuhKgxmPN2c94UBtlYc0kZS6CwyMEEV/nVGSja" +
				"jEZPdnpbK7fEcPd0hWNcOxKWq8qBBPfT69Ore74buf8C26ZTyKnjgMsGCvoDAMOsA07DjjQ1nIkkwIGFFUT3iMO83TdEpWgV/2z7" +
				"WT9axNH/QFPOjXvwQJFnC7hLxHnX6pgKOdAaioKdi6FX3Y2SwWEO3UuxFd3KwsrZ2+mma/W3KP/cPpSzqyHa5VaJwOCw6vSM4wHS" +
				"GKmDF4TSrrnMxzIYiTbTlrwLi5GjMxD6BKzMMN9+7xFuO7txLCEIhGrIMFIvqTw1QFAO4rmAgyG+ljlYTfWHAkzqvImL1o8dMHhG" +
				"OTsMLLMg39KsZVqalZwwL3ckpdAf81OJJeWCpCuaSgSXnWhJmHxQuA9zUhrmlR1wHO9eegHh/p01osP0xU03rY1oGonOZ28acYG6" +
				"MSOfZBkKT+NoqOcEWtL4RCP6t7BWXHgIUmlhCEj/pwNVx92Vc3ZzE8zMh3U196ICHzTSZz0rMwJkmT0l1m7QdvBpqUeqCxyXgY+6" +
				"afqsdAdGjZeuUOPB2RDam3Cm2j2Z5VygvdIBI12qlIoEBhnrhCxx6TN+ywilfI2aBjzTtn0rCe7IA9sYtcYn3XSooU7TBNB39O8c" +
				"bGgnmGYQygxBsQ/Emj2KDCqQ4A1MRnSe3q6tQhjToqDjHRXEKzlWka/4+hWNnJpicq/LmT3jxCH9/yre8qFUXy+Hq2ycitjv3rog" +
				"w+hyXlK3pIoQmDskJnqBk3hxisj3QQrQiv06PubySY5lRIEYlCjr5j8Ahl9gFvN+22cIh1iGiuwByhPjGDgP5h78xZXCBoJekEYP" +
				"cI2C0LtBch5pZC/JpS1kF9lBLndodhIlutEr3mkKohR+D/czN/FTdxU2b82QqfZOHc+6rv2biEXy8AdoAMykj1dsIw7/d5M8XcgP" +
				"iUzNko4H6p02Rt2R01MOYboTogaQH8lyU6o8c+iORRGEoZDTq4htC+Qa7AXTodvSmG33IrwJVGOKDMtvWI1VYdhWs32SB0W1d+Br" +
				"Fb0ObBGsz+Un7P+V8qerCMqu906BkbjdWmsKbKQBFC8/YDTdSi92rIq1ISUQWn88AgW/q+u6KPxybU5EZgbA+EZwCDB6MyBNhHcr" +
				"AvVFeX+kj1RY1Gx1kzCE3ldsT37sCbayFtyMMbL6gDQCoTadJX/jhs9wgp0dZujwOk0Wefhgy1BUHXl/q+2nXAKPvKmli6Wo7/pY" +
				"r/q13Gcsj7Z7WSKVn4Fm4XfkJD62q6paCxO51BlJQEcnpNPKS7+zjhmQlTRiEryD8ve7KQzk20eb4TgIMR1hI5pnQmjGeT56xZyS" +
				"p2nDnYDsqsnXB5uQY8lyf6IYC/PHzEb3rSx91k0ZEu5w5IMrVK8otNzZHrUuM0aPdImpLQJ4qEgvmezORpcUCq4SRp9bGl3/yzXE" +
				"5tWZgn3Q6kXyjFMhu+foTYy1NV+HJbJI1nYMjeTr3f+RxSphIYWyMZ7sD3RgDzRk5iQqD1J+8rdOIZliObfrmWaro/BBxNvd1fPA",
			proof: "qV2FNaBFqWeL6n9q9OUbCSTcIQvwO0vfaA/f/SxEtLSIaOGIOx8r+WVGFdxmC6i3oOaoEkJWvML7PpKBDtqiK7pKDIaMV5PkV" +
				"/kQl6UgxZv9OInTwpVPtYcgeeTokG/eBi1qKzJwDoEHVqKeLqrLXJHXhBVQLdoIUOeKj8YMkagVniO9EtK0fW0/9QnRIxXoilxSj" +
				"5HBEpYwFBitJXRk1ftFGWZFxJXU5PXdRmC+pomyo5Scx+UJQ2NLRWHjKlV0",
			inputs: "aZ8tqrOeEJKt4AMqiRF/WJhIKTDC0HeDTgiJVLZ8OEtiLNj7hflFeVnNXPguxyoqkI/V7pGJtXBpH5N+RswQNA0b23aM33aH" +
				"0HKHOWoGY/T/L7TQzYFGJ3vTLiXDFZg1OVqkGOMvqAgonOrHGi6IgcALyUMyCKlL5BQY23SeILJpYKolybJNwJfbjxpg0Oz+D2fr" +
				"7r9XL1GMvgblu52bVQT1fR8uCRJfSsgA2OGw6k/MpKDCfMcjbR8jnZa8ROEvF4cohm7iV1788Vp2/2bdcEZRQSoaGV8pOmA9Ekqz" +
				"JVRABjkDso40fnQcm2IzjBUOsX+uFExVan56/vl9VZVwB0wnee3Uxiredn0kOayiPB16yimxXCDet+M+0UKjmIlmXYpkrCDrH0dn" +
				"53w+U3OHqMQxPDnUpYBxadM1eI8xWFFxzaLkvega0q0DmEquyY02yiTqo+7Q4qaJVTLgu6/8ekzPxGKRi845NL8gRgaTtM3kidDz" +
				"IQpyODZD0yeEZDY1M+3sUKHcVkhoxTQBTMyKJPc+M5DeBL3uaWMrvxuL6q8+X0xeBt+9kguPUNtIYqUgPAaXvM2i041bWHTJ0dZL" +
				"yDJVOyzGaXRaF4mNkAuh4Et6Zw5PuOpMM2mI1oFKEZj7Xqf/yAmy/Le3GfJnMg5vNgE7QxmVsjuKUP28iN8rdi4bUp7c0KJpqLXE" +
				"6evfRrdZBDRYp+rmOLLDg55ggNuwog==",
			ok: true,
		},
		// grothFail from Scala
		{
			"lp7+dPDIOfm77haSFnvr33VwYH/KbIalfOJPRvBLzqlHD8BxunNebMr6Gr6S+u+nh7yLzdqr7HHQNOpZI8mdj/7lR0IBqB9zvRfyTr+guUG22kZo4y2KINDp272xGglKEeTglTxyDUriZJNF/+T6F8w70MR/rV+flvuo6EJ0+HA+A2ZnBbTjOIl9wjisBV+0iISo2JdNY1vPXlpwhlL2fVpW/WlREkF0bKlBadDIbNJBgM4niJGuEZDru3wqrGueETKHPv7hQ8em+p6vQolp7c0iknjXrGnvlpf4QtUtpg3z/D+snWjRPbVqRgKXWtihuIvPFaM6dt7HZEbkeMnXWwSINeYC/j3lqYnce8Jq+XkuF42stVNiooI+TuXECnFdFi9Ib25b9wtyz3H/oKg48He1ftntj5uIRCOBvzkFHGUF6Ty214v3JYvXJjdS4uS2jekplZYoV0aXEnYEOIvfF7d4xay3qkx2NspooM4HeZpiHknIWkUVhGVJBzBDLjLB",
			"jiGBK+TGHfH8Oadexhdet7ExyIWibSmamWQvffZkyl3WnMoVbTQ3lOks4Mca3sU5qgcaLyQQ1FjFW4g6vtoMapZ43hTGKaWO7bQHsOCvdwHCdwJDulVH16cMTyS9F0BfBJxa88F+JKZc4qMTJjQhspmq755SrKhN9Jf+7uPUhgB4hJTSrmlOkTatgW+/HAf5kZKhv2oRK5p5kS4sU48oqlG1azhMtcHEXDQdcwf9ANel4Z9cb+MQyp2RzI/3hlIx",
			"",
			false},
		{
			"lp7+dPDIOfm77haSFnvr33VwYH/KbIalfOJPRvBLzqlHD8BxunNebMr6Gr6S+u+nh7yLzdqr7HHQNOpZI8mdj/7lR0IBqB9zvRfyTr+guUG22kZo4y2KINDp272xGglKEeTglTxyDUriZJNF/+T6F8w70MR/rV+flvuo6EJ0+HA+A2ZnBbTjOIl9wjisBV+0iISo2JdNY1vPXlpwhlL2fVpW/WlREkF0bKlBadDIbNJBgM4niJGuEZDru3wqrGueETKHPv7hQ8em+p6vQolp7c0iknjXrGnvlpf4QtUtpg3z/D+snWjRPbVqRgKXWtihuIvPFaM6dt7HZEbkeMnXWwSINeYC/j3lqYnce8Jq+XkuF42stVNiooI+TuXECnFdFi9Ib25b9wtyz3H/oKg48He1ftntj5uIRCOBvzkFHGUF6Ty214v3JYvXJjdS4uS2jekplZYoV0aXEnYEOIvfF7d4xay3qkx2NspooM4HeZpiHknIWkUVhGVJBzBDLjLBjiGBK+TGHfH8Oadexhdet7ExyIWibSmamWQvffZkyl3WnMoVbTQ3lOks4Mca3sU5",
			"hp1iMepdu0rKoBh0NXcw9F9hkiggDIkRNINq2rlvUypPiSmp8U8tDSMeG0YVSovFteecr3THhBJj0qNeEe9jA2Ci64fKG9WT1heMYzEAQKebOErYXYCm9d72n97mYn1XBq+g1Y730XEDv4BIDI1hBDntJcgcj/cSvcILB1+60axJvtyMyuizxUr1JUBUq9njtmJ9m8zK6QZLNqMiKh0f2jokQb5mVhu6v5guW3KIjwQc/oFK/l5ehKAOPKUUggNh",
			"c9BSUPtO0xjPxWVNkEMfXe7O4UZKpaH/nLIyQJj7iA4=",
			false},
		{
			"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",
			false},
		{
			"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",
			false},
	} {
		t.Run(nameTest(test.ok, i), func(t *testing.T) {
			vk := deB64(t, test.vk)
			proof := deB64(t, test.proof)
			inputs := deB64(t, test.inputs)
			ok, err := Groth16Verify(vk, proof, inputs, ecc.BLS12_381)
			if test.ok {
				require.NoError(t, err)
				assert.True(t, ok)
			} else {
				assert.NoError(t, err)
				assert.False(t, ok)
			}
		})
	}
}

func BenchmarkGroth16Verify0inputsBLS(b *testing.B) {
	vk := deB64(b, "kYYCAS8vM2T99GeCr4toQ+iQzvl5fI89mPrncYqx3C1d75BQbFk8LMtcnLWwntd6knkzSwcsialcheg69eZYPK8EzKRVI"+
		"5FrRHKi8rgB+R5jyPV70ejmYEx1neTmfYKODRmARr/ld6pZTzBWYDfrCkiS1QB+3q3M08OQgYcLzs/vjW4epetDCmk0K1CEGcWdh7yLzdqr7H"+
		"HQNOpZI8mdj/7lR0IBqB9zvRfyTr+guUG22kZo4y2KINDp272xGglKEeTglTxyDUriZJNF/+T6F8w70MR/rV+flvuo6EJ0+HA+A2ZnBbTjOIl"+
		"9wjisBV+0jgld4oAppAOzvQ7eoIx2tbuuKVSdbJm65KDxl/T+boaYnjRm3omdETYnYRk3HAhrAeWpefX+dM/k7PrcheInnxHUyjzSzqlN03xY"+
		"jg28kdda9FZJaVsQKqdEJ/St9ivXlp7+dPDIOfm77haSFnvr33VwYH/KbIalfOJPRvBLzqlHD8BxunNebMr6Gr6S+u+n")
	proof := deB64(b, "sStVLdyxqInmv76iaNnRFB464lGq48iVeqYWSi2linE9DST0fTNhxSnvSXAoPpt8tFsanj5vPafC+ij/Fh98dOUlMb"+
		"O42bf280pOZ4lm+zr63AWUpOOIugST+S6pq9zeB0OHp2NY8XFmriOEKhxeabhuV89ljqCDjlhXBeNZwM5zti4zg89Hd8TbKcw46jAsjIJe2Si"+
		"w3Th7ELQQKR5ucX50f0GISmnOSceePPdvjbGJ8fSFOnSmSp8dK7uyehrU")
	inputs := deB64(b, "")
	for b.Loop() {
		result, err := Groth16Verify(vk, proof, inputs, ecc.BLS12_381)
		require.True(b, result)
		require.NoError(b, err)
	}
}

func BenchmarkGroth16Verify1inputsBLS(b *testing.B) {
	vk := deB64(b, "mY//hEITCBCZUJUN/wsOlw1iUSSOESL6PFSbN1abGK80t5jPNICNlPuSorio4mmWpf+4uOyv3gPZe54SYGM4pfhteqJp"+
		"wFQxdlpwXWyYxMTNaSLDj8VtSn/EJaSu+P6nFmWsda3mTYUPYMZzWE4hMqpDgFPcJhw3prArMThDPbR3Hx7E6NRAAR0LqcrdtsbDqu2T0tto"+
		"1rpnFILdvHL4PqEUfTmF2mkM+DKj7lKwvvZUbukqBwLrnnbdfyqZJryzGAMIa2JvMEMYszGsYyiPXZvYx6Luk54oWOlOrwEKrCY4NMPwch6D"+
		"bFq6KpnNSQwOpgRYCz7wpjk57X+NGJmo85tYKc+TNa1rT4/DxG9v6SHkpXmmPeHhzIIW8MOdkFjxB5o6Qn8Fa0c6Tt6br2gzkrGr1eK5/+Ri"+
		"IgEzVhcRrqdY/p7PLmKXqawrEvIv9QZ3ijytPNwinlC8XdRLO/YvP33PjcI9WSMcHV6POP9KPMo1rngaIPMegKgAvTEouNFKp4v3wAXRXX5x"+
		"EjwXAmM5wyB/SAOaPPCK/emls9kqolHsaj7nuTTbrvSV8bqzUwzQ")
	proof := deB64(b, "g53N8ecorvG2sDgNv8D7quVhKMIIpdP9Bqk/8gmV5cJ5Rhk9gKvb4F0ll8J/ZZJVqa27OyciJwx6lym6QpVK9q1AS"+
		"rqio7rD5POMDGm64Iay/ixXXn+//F+uKgDXADj9AySri2J1j3qEkqqe3kxKthw94DzAfUBPncHfTPazVtE48AfzB1KWZA7Vf/x/3phYs4ckc"+
		"P7ZrdVViJVLbUgFy543dpKfEH2MD30ZLLYRhw8SatRCyIJuTZcMlluEKG+d")
	inputs := deB64(b, "aZ8tqrOeEJKt4AMqiRF/WJhIKTDC0HeDTgiJVLZ8OEs=")
	for b.Loop() {
		result, err := Groth16Verify(vk, proof, inputs, ecc.BLS12_381)
		require.True(b, result)
		require.NoError(b, err)
	}
}

func BenchmarkGroth16Verify15inputsBLS(b *testing.B) {
	vk := deB64(b, "tRpqHB4HADuHAUvHTcrzxmq1awdwEBA0GOJfebYTODyUqXBQ7FkYrz1oDvPyx5Z3sUmODSJXAQmAFBVnS2t+Xzf5ZCr1"+
		"gCtMiJVjQ48/nob/SkrS4cTHHjbKIVS9cdD/BG/VDrZvBt/dPqXmdUFyFuTTMrViagR57YRrDmm1qm5LQ/A8VwUBdiArwgRQXH9jsYhgVmfc"+
		"RAjJytrbYeR6ck4ZfmGr6x6akKiBLY4B1l9LaHTyz/6KSM5t8atpuR3HBJZfbBm2/K8nnYTl+mAU/EnIN3YQdUd65Hsd4Gtf6VT2qfz6hcrS"+
		"gHutxR1usIL2kyU9X4Kqjx6I6zYwVbn7PWbiy3OtY277z4ggIqW6AuDgzUeIyG9a4stMeQ07mOV/Ef4faj+eh4GJRKjJm7aUTYJCSAGY6klO"+
		"XNoEzB54XF4EY5pkMPfW73SmxJi9B0aHkZWDy2tzUlwvxZ/BfsDkUZnt6mI+qdDOtTG6JFItSQZotYGDBm6zPczwo3ZAGpr8gibTE6DjT7GG"+
		"NDEl26jgAJ3aAdBrf7Yb0vWEYizOJK4SO/Ud+4/WxXDby7xbwlFYkgEtYbMO6PXozhRqDiotJ0CfdSExNHA9A37mR/bpNOKyhArfyvSBIJnU"+
		"QgOw5wMBq+GOP5n78E99a5rY4FXGUmM3LGdp/CvkGITYf04SWHkZAEueYH96Ys5jrHlIZQA2k9j02Ji+SL82DJFH8LDh77fgh9zh0wAjCAqY"+
		"7/r72434RDA97bfEZJavRmAENsgflsSVb8d9rQMBpWl3Xkb8mNlUOSf+LAXeXYQR42Z4yuUjwAUvk//+imuhsWF8ZCMkpb9wQ/6crVH4E5E3"+
		"f6If/Mt/DcenWlPNtvu2CJFatc8q31aSdnWhMN8U65SX3DBouDc8EXDFd5twy4VWMS5lhY6VbU/lS8T8oyhr+NIpstsKUmSh0EM1rGyUh2PN"+
		"gIYzoeBznHWagp2WO3nIbNYIcXEROBT8QpqA4Dqzxv665jwajGXmAawRvdZqzLqvCkeujekplZYoV0aXEnYEOIvfF7d4xay3qkx2NspooM4H"+
		"eZpiHknIWkUVhGVJBzBDLjLBjiGBK+TGHfH8Oadexhdet7ExyIWibSmamWQvffZkyl3WnMoVbTQ3lOks4Mca3sU5hp1iMepdu0rKoBh0NXcw"+
		"9F9hkiggDIkRNINq2rlvUypPiSmp8U8tDSMeG0YVSovFlA4DsjBwntJH45NgNbY/Rbu/hfe7QskTkBiTo2A+kmYSH75Uvf2UAXwBAT1PoE0s"+
		"qtYndF2Kbthl6GylV3j9NIKtIzHd/GwleExuM7KlI1H22P78br5zmh8D7V1aFcxPpftQhjch4abXuxEP4ahgfNmthdhoSvQykLhjbmG9Brvw"+
		"myaDRd/sHCTeSXmLqIybrd6tA8ZLJq2DLzKJEOlmfM9aIihLe/FLndfnTSkNK2et4o8vM3YjAmgOnrAo7JIp")
	proof := deB64(b, "lgFU4Jyo9GdHL7w31u3zXc8RQRnHVarZWNfd0lD45GvvQtwrZ1Y1OKB4T29a79UagPHOdk1S0k0hYAYQyyNAfRUzd"+
		"e1HP8R+2dms75gGZEnx2tXexEN+BVjRJfC8PR1lFJa6xvsEx5uSrOZzKmoMfCwcA55SMT5jFo4+KyWg2wP5OnFPx7XTdEKvf5YhpY0krQKiq"+
		"3OUu79EwjNF1xV1+iLxx2KEIyK7RSYxO1BHrKOGOEzxSUK00MA+YVHe+DvW")
	inputs := deB64(b, "aZ8tqrOeEJKt4AMqiRF/WJhIKTDC0HeDTgiJVLZ8OEtiLNj7hflFeVnNXPguxyoqkI/V7pGJtXBpH5N+RswQNA0b"+
		"23aM33aH0HKHOWoGY/T/L7TQzYFGJ3vTLiXDFZg1OVqkGOMvqAgonOrHGi6IgcALyUMyCKlL5BQY23SeILJpYKolybJNwJfbjxpg0Oz+D2fr"+
		"7r9XL1GMvgblu52bVQT1fR8uCRJfSsgA2OGw6k/MpKDCfMcjbR8jnZa8ROEvF4cohm7iV1788Vp2/2bdcEZRQSoaGV8pOmA9EkqzJVRABjkD"+
		"so40fnQcm2IzjBUOsX+uFExVan56/vl9VZVwB0wnee3Uxiredn0kOayiPB16yimxXCDet+M+0UKjmIlmXYpkrCDrH0dn53w+U3OHqMQxPDnU"+
		"pYBxadM1eI8xWFFxzaLkvega0q0DmEquyY02yiTqo+7Q4qaJVTLgu6/8ekzPxGKRi845NL8gRgaTtM3kidDzIQpyODZD0yeEZDY1M+3sUKHc"+
		"VkhoxTQBTMyKJPc+M5DeBL3uaWMrvxuL6q8+X0xeBt+9kguPUNtIYqUgPAaXvM2i041bWHTJ0dZLyDJVOyzGaXRaF4mNkAuh4Et6Zw5PuOpM"+
		"M2mI1oFKEZj7")
	for b.Loop() {
		result, err := Groth16Verify(vk, proof, inputs, ecc.BLS12_381)
		require.True(b, result)
		require.NoError(b, err)
	}
}

func BenchmarkGroth16Verify16inputsBLS(b *testing.B) {
	vk := deB64(b, "kY4NWaOoYItWtLKVQnxDh+XTsa0Yev5Ae3Q9vlQSKp6+IUtwS7GH5ZrZefmBEwWEqvAtYaSs5qW3riOiiRFoLp7MThW4"+
		"vCEhK0j8BZY5ZM/tnjB7mrLB59kGvzpW8PM/AoQRIWzyvO3Dxxfyj/UQcQRw+KakVRvrFca3Vy2K5cFwxYHwl6PFDM+OmGrlgOCoqZtY1SLO"+
		"d+ovmFOODKiHBZzDZhC/lRfjKVy4LzI7AXDuFn4tlWoT7IsJyy6lYNaWFfLjYZPAsrv1gXJ1NYat5B6E0Pnz5C67u2Uigmlol2D91re3oAqI"+
		"o+r8kiyFKOSBooG0cMN47zQor6qj0owuxJjn5Ymrcd/FCQ1ud4cKoUlNaGWIekSjxJEB87elMy5oEUlUzVI9ObMm+2SE3Udgws7pkMM8fgQU"+
		"QUqUVyc7sNCE9m/hQzlwtbXrNSS5Pb+6ow7aHMOavjVyaXiS0f6b1pwJpS1yT+K85UA1CLqqxCaEw5+8WAjMzBOrKmxBUpYApI4FBAIa/Sje"+
		"U/wYnljUUMTMfnBfCQ8MS01hFSQZSoPx1do8Zxn5Y3NPgpaomXDfpyVK9Q0U0NkqQqPsk+T+AroxQGxq9f/HOX5I5ZibF27dZ32tCbTKo22G"+
		"gspqtAv2iv06PubySY5lRIEYlCjr5j8Ahl9gFvN+22cIh1iGiuwByhPjGDgP5h78xZXCBoJekEYPcI2C0LtBch5pZC/JpS1kF9lBLndodhIl"+
		"utEr3mkKohR+D/czN/FTdxU2b82QqfZOHc+6rv2biEXy8AdoAMykj1dsIw7/d5M8XcgPiUzNko4H6p02Rt2R01MOYboTogaQH8lyU6o8c+iO"+
		"RRGEoZDTq4htC+Qa7AXTodvSmG33IrwJVGOKDMtvWI1VYdhWs32SB0W1d+BrFb0ObBGsz+Un7P+V8qerCMqu906BkbjdWmsKbKQBFC8/YDTd"+
		"Si92rIq1ISUQWn88AgW/q+u6KPxybU5EZgbA+EZwCDB6MyBNhHcrAvVFeX+kj1RY1Gx1kzCE3ldsT37sCbayFtyMMbL6gDQCoTadJX/jhs9w"+
		"gp0dZujwOk0Wefhgy1BUHXl/q+2nXAKPvKmli6Wo7/pYr/q13Gcsj7Z7WSKVn4Fm4XfkJD62q6paCxO51BlJQEcnpNPKS7+zjhmQlTRiEryD"+
		"8ve7KQzk20eb4TgIMR1hI5pnQmjGeT56xZySp2nDnYDsqsnXB5uQY8lyf6IYC/PHzEb3rSx91k0ZEu5w5IMrVK8otNzZHrUuM0aPdImpLQJ4"+
		"qEgvmezORpcUCq4SRp9bGl3/yzXE5tWZgn3Q6kXyjFMhu+foTYy1NV+HJbJI1nYMjeTr3f+RxSphIYWyMZ7sD3RgDzRk5iQqD1J+8rdOIZli"+
		"ObfrmWaro/BBxNvd1fPAlFEPiDegBcDaVWHS2A1FPIC9d+DU05vizrBfli6su9rCvSBNVnoDSBF2zeU+2NjXj7ycHYxCuZgl8dBu8FZjvjlD"+
		"UZCqfdq3PszQeo2X55trDJEHeVWaRoIcgiG2hfTN")
	proof := deB64(b, "jqPSA/XKqZDJnRSmM0sJxbrFv7GUcA45QMysIx1xTsI3+2iysF5Tr68565ZuO65qjo2lklZpQo+wtyKSA/56EaKOJ"+
		"ZCZhSvDdBEdvVYJCjmWusuK5qav7xZO0w5W1qRiEgIdcGUz5V7JHqfRf4xI6/uUD846alyzzNjxQtKErqJbRw6yyBO6j6box363pinjiMTzU"+
		"4w/qltzFuOEpKxy/H3vyH8RcsF24Ou/Rb6vfR7cSLtLwCsf/BMtPcsQfdRK")
	inputs := deB64(b, "aZ8tqrOeEJKt4AMqiRF/WJhIKTDC0HeDTgiJVLZ8OEtiLNj7hflFeVnNXPguxyoqkI/V7pGJtXBpH5N+RswQNA0b"+
		"23aM33aH0HKHOWoGY/T/L7TQzYFGJ3vTLiXDFZg1OVqkGOMvqAgonOrHGi6IgcALyUMyCKlL5BQY23SeILJpYKolybJNwJfbjxpg0Oz+D2fr"+
		"7r9XL1GMvgblu52bVQT1fR8uCRJfSsgA2OGw6k/MpKDCfMcjbR8jnZa8ROEvF4cohm7iV1788Vp2/2bdcEZRQSoaGV8pOmA9EkqzJVRABjkD"+
		"so40fnQcm2IzjBUOsX+uFExVan56/vl9VZVwB0wnee3Uxiredn0kOayiPB16yimxXCDet+M+0UKjmIlmXYpkrCDrH0dn53w+U3OHqMQxPDnU"+
		"pYBxadM1eI8xWFFxzaLkvega0q0DmEquyY02yiTqo+7Q4qaJVTLgu6/8ekzPxGKRi845NL8gRgaTtM3kidDzIQpyODZD0yeEZDY1M+3sUKHc"+
		"VkhoxTQBTMyKJPc+M5DeBL3uaWMrvxuL6q8+X0xeBt+9kguPUNtIYqUgPAaXvM2i041bWHTJ0dZLyDJVOyzGaXRaF4mNkAuh4Et6Zw5PuOpM"+
		"M2mI1oFKEZj7Xqf/yAmy/Le3GfJnMg5vNgE7QxmVsjuKUP28iN8rdi4=")
	for b.Loop() {
		result, err := Groth16Verify(vk, proof, inputs, ecc.BLS12_381)
		require.True(b, result)
		require.NoError(b, err)
	}
}

func TestGroth16VerifyOKBn256(t *testing.T) {
	for i, test := range []struct {
		vk     string
		proof  string
		inputs string
		ok     bool
	}{
		{
			vk: "LDCJzjgi5HtcHEXHfU8TZz+ZUHD2ZwsQ7JIEvzdMPYKYs9SoGkKUmg1yya4TE0Ms7x+KOJ4Ze/CPfKp2s5jbniFNM71N/YlHVbNk" +
				"ytLtQi1DzReSh9SNBsvskdY5mavQJe+67PuPVEYnx+lJ97qIG8243njZbGWPqUJ2Vqj49NAunhqX+eIkK3zAB3IPWls3gruzX2t9" +
				"wrmyE9cVVvf1kgWx63PsQV37qdH0KcFRpCH89k4TPS6fLmqdFxX3YGHCGFTpr6tLogvjbUFJPT98kJ/xck0C0B/s8PTVKdao4VQH" +
				"T4DBIO8+GB3CQVh6VV4EcMLtDWWNxF4yloAlKcFT0Q4AzJSimpFqd/SwSz9Pb7uk5srte3nwphVamC+fHlJt",
			proof: "GQPBoHuCPcIosF+WZKE5jZV13Ib4EdjLnABncpSHcMKBZl0LhllnPxcuzExIQwhxcfXvFFAjlnDGpKauQ9OQsjBKUBsdBZnGi" +
				"V2Sg4TSdyHuLo2AbRRqJN0IV3iH3On8I4ngnL30ZAxVyGQH2EK58aUZGxMbbXGR9pQdh99QaiE=",
			inputs: "IfZhAypdtgvecKDWzVyRuvXatmFf2ZYcMWVkCJ0/MQo=",
			ok:     true,
		},
		{
			"oNme33MLprvAodIU3H8rA1WVUVN6IwJtouPFt3rD76EhTyetvNmF9cCLETzYB4K9YC4EIAnZywPddo8kG70hDAeEOBf1FwaXr53SXD0A2pGbHJPRuTTj21tXNXAu7D6MkGKUMCACqi7buNhBbz0X7SKNBgh2Lwxo9CFcv4VBCzkljSjsy0dXI1nUz8oAN99WPRgEqsGtH6wFSllr/AMgKxUVjbIGGpcKvAgvuOfP6HYRsVR8XP7Ecnh/A/aARw4jCycuzPPhJlgjUxs+xCwF/AizkOvYoKFAdFlLKACJIFsIOU8NmXM08eizRXg96hvpfbCVjlWYE1hI90EnJLKTBg==",
			"G3CEMZl39+IlulSqUjmLUP4tB8pnGGteKKU8AlMzkMMcThMVl9rOa5G3DDcm4iF3BFxS3ubW5JADnVvPhX9wNQXMMjh+BKVLUIaC8gwtwciCscaCMw6cIA5ltWuoWX9jB/ig6Yqg4Hc6u2/9XzKfCWG42Si+BkTh8X0DAQ2fAzY=",
			"",
			true},
		{
			"LDCJzjgi5HtcHEXHfU8TZz+ZUHD2ZwsQ7JIEvzdMPYKYs9SoGkKUmg1yya4TE0Ms7x+KOJ4Ze/CPfKp2s5jbniFNM71N/YlHVbNkytLtQi1DzReSh9SNBsvskdY5mavQJe+67PuPVEYnx+lJ97qIG8243njZbGWPqUJ2Vqj49NAunhqX+eIkK3zAB3IPWls3gruzX2t9wrmyE9cVVvf1kgWx63PsQV37qdH0KcFRpCH89k4TPS6fLmqdFxX3YGHCGFTpr6tLogvjbUFJPT98kJ/xck0C0B/s8PTVKdao4VQHT4DBIO8+GB3CQVh6VV4EcMLtDWWNxF4yloAlKcFT0Q4AzJSimpFqd/SwSz9Pb7uk5srte3nwphVamC+fHlJt",
			"GQPBoHuCPcIosF+WZKE5jZV13Ib4EdjLnABncpSHcMKBZl0LhllnPxcuzExIQwhxcfXvFFAjlnDGpKauQ9OQsjBKUBsdBZnGiV2Sg4TSdyHuLo2AbRRqJN0IV3iH3On8I4ngnL30ZAxVyGQH2EK58aUZGxMbbXGR9pQdh99QaiE=",
			"IfZhAypdtgvecKDWzVyRuvXatmFf2ZYcMWVkCJ0/MQo=",
			true},
		{
			"nV33RK4rTODU42ZKyFxTFl9d86FrP+h6acIdT4m/rfAZBBfPWLxcjYyMUBHlXgM/jTBDOH7dTGL1zqbRLr1eGwR9zemG9LJnqPl/eiJ0LZLZKz3/iDde8p1zj5DRvar2Fa6rv0WJRJR32+22iaZHrD/64/SyFb2j5f12ipT5S2Eq+SnYSSr8HhStVu3s4VFK57nhi2aRFUgXWkGaqhJJ6ie6zlaVf52s79qdlBUOuTTVsATXa58FVXVSHJb5wanwK578EWV/BulP1TCq5y0q7k6YCZV15Nu9FHpzIUow9Ged8l8LwPmHxki+/S2MnA0v2mFgdC1ZXE+BesWOx3tXThrq7st5+lUMmaY5S76l0yuzzHoeSZEZAoS6heG8WWUGJW2QBTzq3GVaubssoR/HtIT1dGPswNTF8HY78BPYb8M=",
			"pE0Y4zE1A5UvC8ATBJPMOAFm3dDQABrhu5VxxGj5GOom+pFpQsVZe6mMLT8ZwyWXQmJ+od+cYtUOH2Fxiem1yCdMkGi64f5k9qWwYDRrtIxXHhj9f43pHKylJV0Z1jCJgpebExv+hPmwNYh3cWxHxywYMEXwSutAjyqi8Swl1O0=",
			"IfZhAypdtgvecKDWzVyRuvXatmFf2ZYcMWVkCJ0/MQou2EkLZ569itz7cL7GQnzipNe1JRCQ/QK8UXr8IG8k0g==",
			true},
		{
			"q1U6rGRU+/F+e3xx6oCBeokQsIVIQDESLC/l+PK8WX0Rw2NglYMRP6In971j7/puuDsJwZm2AK7fTcmKQd5v/g6InaZLdKGQK1C+52BbMyoGsueMfmML9pJmJvivE2N7m9WyiixGvDawwhz7JYst5Jr3ChczYgLAS7b2WQscCRcm8ZE9FKTNdfu4WAYEDEl6IONCFnNiYw6wR8Kc2J7Z9BkHcOzQj2X4PuxZGdPeWF+7vB1dXQZZlCBScImWeVPNKaYF7H3Ot0wOYzigQXYkSZD5v9vBWXN/f1YajxR1kjMrSZ90fx4WapypjaLREbaNgOcAN5niSzTmwymK3e3Idgpt+Pdq34YPOZ5lDx5+3gMWYl1QYeWi6miDIasts0owjZVOTUXf292iZDsjSWaYurcLNEH3Rf8lTCKSMyosjkAV2Fkselh0/Vy4jkPj3hTFeIpY3inMmI+N6Nvwxv50ZpaUIQmU5rlia4pS8MSMxgO+khyyPKb1OUHbFTgDHf6p",
			"mtMjjG0IVNq3t7ICtij1nLFH1UjFu6FaNZmdXam0gdCY/efE2wVHSctp1vqgVcfgiSpl/WCCcaQ2CQGbirLE/wVj2w0eiVEUy6gs9aK01nS0bx4ErymppOugPDPXGCV6FtiAM6Cq+fMbOKhhhDPutn6dIntO3gSqtsWL0KreCPY=",
			"IfZhAypdtgvecKDWzVyRuvXatmFf2ZYcMWVkCJ0/MQou2EkLZ569itz7cL7GQnzipNe1JRCQ/QK8UXr8IG8k0hW3GAoAXun/Pwk0Jq9fst26FET2RehSLbxUdvZQLjEzCJmfahCreklII85wEMuwYctsOv/3JWgvrI8hmn8sWjw=",
			true},
		{
			"kikeX/IlYwL5e1nkJ3FbD4/IantLORJJtmgMPywFCoCdYF304ka26knjzRodk7mujA0HqXxknBkzZSqQuuqcSSZkVOudYO1dwOw73U9SlL/nF4UWP4YOkVfKntqi825LkCrBcl3rw7zhMHorldspOwWNVybCLGt86Zbd92hze3wcSIk2GqMxtoBQiurYqlt1SARwIn06tRExJ0YULl/7/4qrYi+Tsjy0xQwltrHJQy/eeVNNq9x/GrVJemN2r+SUK6lF6kWLJzTH5jUUu7HvpUCtRlGJ3JRZIAJ7qChT3v+fD5LNjz86Ei1ItV4Wgqk/iXAAXeORUM3T3RD2NepVgS3U35HXhJ1ZtzJZoFUQyQjlGYp6+U6q8+rXYSQkZjEHmQh/Atr0lb1QRIG+PK+mLj1Nxwb/aibt7z7WtRTfj3mudYCEjXA/ceDVtz8FKkeTo9dcVYGsf/VJaxnteVa7B4UMRQjX0gdNKC+AEjKxcRk373IqLBCCtukQYBG8o2TKFYfJcJuI1geuGtNij1jwgmLxl7hH3Y0JqMZPoS9vKwudBL5Nnt38g/rfNTyoe7UIJV246oAOkuuskfxNUHeEnILjkHYhjfIuuPMGUG4FK2LHkscqib6eHo3wSy31I65urmsiHROrMEtfUdYcGDJxV+IyrTUs2m5KQgmlXo8jvUo=",
			"HUaOoIyn6KUkVXzttFMcRobGBZUtpRjAQMId9n3ABZKPUk8Rv0EPKmzp/1Ut0LNJ5cNI48VwD6/kVbhNgoBbdQijsqDktFjjeJtzj6KF2TuHOlcHL6s7dc38cvNVD0O1ISiJlLdrc5QKcXePAGJK/YaD/CUQfKnijgCGDREUDXo=",
			"IfZhAypdtgvecKDWzVyRuvXatmFf2ZYcMWVkCJ0/MQou2EkLZ569itz7cL7GQnzipNe1JRCQ/QK8UXr8IG8k0hW3GAoAXun/Pwk0Jq9fst26FET2RehSLbxUdvZQLjEzCJmfahCreklII85wEMuwYctsOv/3JWgvrI8hmn8sWjwELKgjqMSLQgJ7u5ZHe9PDhBgF/3dliXhY1jUsXFvkig6pfEr0bFuXhXcg2R+vuPErhG0w08SPNOi2SjvPngYpJ8nsOT98G6EDlyNaHtJuRC/xOA+6ftL/k2+hzFsd+1Ui5/1NGTfDxLDndEE1NQ+opvk79ZeaY+qPP7pEc0uQMw==",
			true},
		{
			"qladoBQdTIMTqRD5nLplvCqxlHwZrFv7scw72CqqjkMsK1J9mo3aYYCS8MXIl224CTjyJF/IQ2nFONXeSUdoFAFN1RfVylmeW3Qbeqj8ePv9JA+LjpgE68Zr9u85DqI4hK+0BABhe1S+m79CtKaSOAt7pJNDmiNqEKsGYDBhYBIBLAdgsmLkOfA197v06p/UaXCOXvNQMjprjuvt3SCrkoV8jZU/cSrB+cPPE8s4OgVl0eXr08YRmmDym/veV4eTI1WV59Qxb3uZCOtJFPwwd6gTiXvnGhhWrgJPwjAIJhmFDEUI19IHTSgvgBIysXEZN+9yKiwQgrbpEGARvKNkyhWHyXCbiNYHrhrTYo9Y8IJi8Ze4R92NCajGT6EvbysLnQS+TZ7d/IP63zU8qHu1CCVduOqADpLrrJH8TVB3hJyC45B2IY3yLrjzBlBuBStix5LHKom+nh6N8Est9SOubq5rIh0TqzBLX1HWHBgycVfiMq01LNpuSkIJpV6PI71KmXEqw09G6gPAM1465XNSWZV0gLUTctlavUWfG7jILPafEBGrM69wVNmnBm0lSOq9fBTatb8Ivm4+SDNHALraRo1guaelsx+MuKox7hj5WkuIjo7PSprYHsM6Wc3/10VpJQwX6/Dp5J0MZzYscKLlGX5DXXbaQLcIT6I+cjioLcIwK20hLc56+CaCSZRyKMB9IWUmavploHQrjBW+vyyOPQNIbqUTWjTVHJ9QCCdVxUWP+0yHwCjZUymqnVoG6HrPkYvh0nuIsz552K6SWFMuhddTW+JN/uUIpniAKCI4WwafIl/0mH/DRktCA5uQdpevX62mWKfyYGL1if6TV20CgSMvo6fiK+yC5GCzMKtsHxMFWW5fYkjQ2b/C8RWjCySpDQPFVqJwr5uMYxdqtsSs7ysGZfpoRZS1SDbVgFVL1E6d/ECJMiqIOM0OH0uBRzF7B3q2BT5GChq/naHPEucCcBU8HgairQQ3uV+V+UyWbYmrwjGQZSg0pSQ3Jff41MGe",
			"CfEpVT6b8+4NAeDs3QwiSN7zqfxzAkQdIu8eBXzoAQIS+AgJcYppUx7COvtbWa7TDtaER1ydtoYWBcBtRMvrHQJ64u4XmLooTwikzECPz+VRcYknrGEoyGeZanNFWEwgplf9bX3JvW1RshlAfN7iJESdqBCmUNsrObHNxhHFJRo=",
			"IfZhAypdtgvecKDWzVyRuvXatmFf2ZYcMWVkCJ0/MQou2EkLZ569itz7cL7GQnzipNe1JRCQ/QK8UXr8IG8k0hW3GAoAXun/Pwk0Jq9fst26FET2RehSLbxUdvZQLjEzCJmfahCreklII85wEMuwYctsOv/3JWgvrI8hmn8sWjwELKgjqMSLQgJ7u5ZHe9PDhBgF/3dliXhY1jUsXFvkig6pfEr0bFuXhXcg2R+vuPErhG0w08SPNOi2SjvPngYpJ8nsOT98G6EDlyNaHtJuRC/xOA+6ftL/k2+hzFsd+1Ui5/1NGTfDxLDndEE1NQ+opvk79ZeaY+qPP7pEc0uQMxnEIHHpcYsRt9dal9bSQpE5hWKu1nOBzIVmXb/Ef51YDg+nW5w9a2tEAY2zQCZ/z3sFs7FwAZ5TXhDfhYR5sQ8tZaF3FWh+Yzf5hgMmXWrApp/arwPszNhKCoxScnhPSgfcxYSdauqvp5+vcacJFY1OkWG6tQ6iuh5CZSdX645oA7oNr9d5kXYSewhTflcV9pufeaH21BtEipHpO3sNRkUngnHC+uj1D8ReSgcCofnv8s0mVme9Ml64r6CbeaHK+x5Mc9bolN96XJZ137xkPDpev+RVrVK6ZIFrH2hFl8/vGoBwWDlmjmVzUt4YQdsCavsf7c0vBa7d33EcvyvQMDY=",
			true},
	} {
		t.Run(nameTest(test.ok, i), func(t *testing.T) {
			vk, err := base64.StdEncoding.DecodeString(test.vk)
			require.NoError(t, err)
			proof, err := base64.StdEncoding.DecodeString(test.proof)
			require.NoError(t, err)
			inputs, err := base64.StdEncoding.DecodeString(test.inputs)
			require.NoError(t, err)
			result, err := Groth16Verify(vk, proof, inputs, ecc.BN254)
			if test.ok {
				require.NoError(t, err)
				assert.True(t, result)
			} else {
				assert.Error(t, err)
				assert.False(t, result)
			}
		})
	}
}

func TestGroth16VerifyFailBn256(t *testing.T) {
	vkFail0 := make([]byte, 256)
	for i := range vkFail0 {
		vkFail0[i] = 1
	}
	vkFail1 := make([]byte, 256+32)
	for i := range vkFail0 {
		vkFail0[i] = 1
	}
	vkFail15 := make([]byte, 256+32*15)
	for i := range vkFail0 {
		vkFail0[i] = 1
	}
	vkFail16 := make([]byte, 256+32*16)
	for i := range vkFail0 {
		vkFail0[i] = 1
	}
	for i, test := range []struct {
		vk     string
		proof  string
		inputs string
		error  string
	}{
		{
			vk: base64.StdEncoding.EncodeToString(vkFail0),
			proof: "CfEpVT6b8+4NAeDs3QwiSN7zqfxzAkQdIu8eBXzoAQIS+AgJcYppUx7COvtbWa7TDtaER1ydtoYWBcBtRMvrHQJ64u4XmLoo" +
				"TwikzECPz+VRcYknrGEoyGeZanNFWEwgplf9bX3JvW1RshlAfN7iJESdqBCmUNsrObHNxhHFJRo=",
			inputs: "",
			error:  "invalid point: subgroup check failed",
		},
		{
			vk: base64.StdEncoding.EncodeToString(vkFail1),
			proof: "CfEpVT6b8+4NAeDs3QwiSN7zqfxzAkQdIu8eBXzoAQIS+AgJcYppUx7COvtbWa7TDtaER1ydtoYWBcBtRMvrHQJ64u4XmLoo" +
				"TwikzECPz+VRcYknrGEoyGeZanNFWEwgplf9bX3JvW1RshlAfN7iJESdqBCmUNsrObHNxhHFJRo=",
			inputs: "c9BSUPtO0xjPxWVNkEMfXe7O4UZKpaH/nLIyQJj7iA4=",
			error:  "invalid compressed coordinate: square root doesn't exist",
		},
		{
			vk: base64.StdEncoding.EncodeToString(vkFail15),
			proof: "CfEpVT6b8+4NAeDs3QwiSN7zqfxzAkQdIu8eBXzoAQIS+AgJcYppUx7COvtbWa7TDtaER1ydtoYWBcBtRMvrHQJ64u4XmLoo" +
				"TwikzECPz+VRcYknrGEoyGeZanNFWEwgplf9bX3JvW1RshlAfN7iJESdqBCmUNsrObHNxhHFJRo=",
			inputs: "I8C5RcBDPi2n4omt9oOV2rZk9T9xlSV8PQvLeVHjGb00fCVz7AHOIjLJ03ZCTLQwEKkAk9tQWJ6gFTBnG2+0DDHlXcVkwpMa" +
				"fcpS2diKFe0T4fRb0t9mxNzOFiRVcJoeMU1zb/rE4dIMm9rbEPSDnVSOd8tHNnJDkT+/NcNsQ2w0UEVJJRAEnC7G0Y3522RlDLxp" +
				"TZ6w0U/9V0pLNkFgDCkFBKvpaEfPDJjoEVyCUWDC1ts9LIR43xh3ZZBdcO/HATHoLzxM3Ef11qF+riV7WDPEJfK11u8WGazzCAFh" +
				"sx0aKkkbnKl7LnypBzwRvrG2JxdLI/oXL0eoIw9woVjqrg6elHudnHDXezDVXjRWMPaU+L3tOW9aqN+OdP4AhtpgT2CoRCjrOIU3" +
				"MCFqsrCK9bh33PW1gtNeHC78mIetQM5LWZHtw4KNwafTrQ+GCKPelJhiC2x7ygBtat5rtBsJAVF5wjssLPZx/7fqNqifXB7WyMV7" +
				"J1M8LBQVXj5kLoS9bpmNHlERRSadC0DEUbY9xhIG2xo7R88R0sq04a299MFv8XJNd+IdueYiMiGF5broHD4UUhPxRBlBO3lOfDTP" +
				"nRSUGS3Sr6GxwCjKO3MObz/6RNxCk9SnQ4NccD17hS/m",
			error: "invalid compressed coordinate: square root doesn't exist",
		},
		{
			vk: base64.StdEncoding.EncodeToString(vkFail16),
			proof: "CfEpVT6b8+4NAeDs3QwiSN7zqfxzAkQdIu8eBXzoAQIS+AgJcYppUx7COvtbWa7TDtaER1ydtoYWBcBtRMvrHQJ64u4XmLoo" +
				"TwikzECPz+VRcYknrGEoyGeZanNFWEwgplf9bX3JvW1RshlAfN7iJESdqBCmUNsrObHNxhHFJRo=",
			inputs: "I8C5RcBDPi2n4omt9oOV2rZk9T9xlSV8PQvLeVHjGb00fCVz7AHOIjLJ03ZCTLQwEKkAk9tQWJ6gFTBnG2+0DDHlXcVkwpMa" +
				"fcpS2diKFe0T4fRb0t9mxNzOFiRVcJoeMU1zb/rE4dIMm9rbEPSDnVSOd8tHNnJDkT+/NcNsQ2w0UEVJJRAEnC7G0Y3522RlDLxp" +
				"TZ6w0U/9V0pLNkFgDCkFBKvpaEfPDJjoEVyCUWDC1ts9LIR43xh3ZZBdcO/HATHoLzxM3Ef11qF+riV7WDPEJfK11u8WGazzCAFh" +
				"sx0aKkkbnKl7LnypBzwRvrG2JxdLI/oXL0eoIw9woVjqrg6elHudnHDXezDVXjRWMPaU+L3tOW9aqN+OdP4AhtpgT2CoRCjrOIU3" +
				"MCFqsrCK9bh33PW1gtNeHC78mIetQM5LWZHtw4KNwafTrQ+GCKPelJhiC2x7ygBtat5rtBsJAVF5wjssLPZx/7fqNqifXB7WyMV7" +
				"J1M8LBQVXj5kLoS9bpmNHlERRSadC0DEUbY9xhIG2xo7R88R0sq04a299MFv8XJNd+IdueYiMiGF5broHD4UUhPxRBlBO3lOfDTP" +
				"nRSUGS3Sr6GxwCjKO3MObz/6RNxCk9SnQ4NccD17hS/mEFt8d4ERZOfmuvD3A0RCPCnx3Fr6rHdm6j+cfn/NM6o=",
			error: "invalid compressed coordinate: square root doesn't exist",
		},
	} {
		t.Run(nameTest(false, i), func(t *testing.T) {
			vk := deB64(t, test.vk)
			proof := deB64(t, test.proof)
			inputs := deB64(t, test.inputs)
			ok, err := Groth16Verify(vk, proof, inputs, ecc.BN254)
			assert.False(t, ok)
			assert.Error(t, err)
			assert.EqualError(t, err, test.error)
		})
	}
}

func TestGroth16VerifyBN254Regression(t *testing.T) {
	key1 := deB64(t, "lqosVoUS5DoB/0S3iPB5gUE3XayQEOsYSc/rsKoRqOIqfn25C6ZzEH9aAkDbZr2tmaTpgPf7BBUKIaV7Kl5k2RAHtl+O"+
		"9HATW7jpFBL19G96kV4VFNs76eW9x4UiY09frZuVdJyYo6fSdw12ayMZZFVC5MGynBT9R3KtsSpNigkQLrU0qybigIf7T/TPcAyLav6fyThn6"+
		"5NSpYFFEqtR2CVhBG8nJGlraC//ZuIlfXjiQnAaFL8MsYcXvrsc6ch6B3eWzc4Dx9+IlsxT7hz+9b4+wrf9gdryoCUPYojGXtIHNkCE8Khr+0"+
		"qiQIPXKwSWBG0hk04allgiKZMRbvdvTLADvkQhJzaklyCu+bAkI6IhajELJcV7S1Jh6/r3k+28qXKnYVrLrDxh/r/8hqH1CF9N7f/cbn4U86/"+
		"LWfUOU7uZu4O9CRo9iozWHZVm/BBqhxq9rHxHqUXzTcdsF2O91iGPW6EQ+VqThpcIOZoTvH6IkCqkIBKsxX2/rJx/85awAyQsQRf25ls8dm/O"+
		"Bpi/wz6vv3sPF0dbkuogTDFUtk8a0nwcYsCebqNEzG6aC4CO0ddV2QsqxtKBwXlgijvLmCjK8zQfTCIxWGLP8qI+75xrqMZUxnV9xhLDlEYcp"+
		"27ECVd/JGbMjwVKcpsR7/dR2oXKMYs+MhP6LzArECQ4hEsFlyDngjUeJnNnVV4DraW6oo1M8bqIF9yTv/Ey43p9loja4PXlL9AwAr8NRJZDq4"+
		"EVkr3DNPVYFUlL7Z7lQPQxlTWPcIGamwcYPXJDEMJSxNlyuhrQcu8/uOtRuP2HxgokJ2kS71bsNWQ36Ekof4XRO/FsMIazVjCq+40TGd9IXoV"+
		"9KNIFkj8+MQyHGZ2j+OS9D95TFdXb6tGkCiLFfUjMB4xOzJHh1D+/6dGqxjhDDeUbuoUctnsm1kyZiCG94u2XefvcHmLKRcoRxZ+anAAmMECj"+
		"5m35sZ5L//LwaV080w==")
	key2 := deB64(t, "qq+txz7VJ4xNReXEeUsTx74+tw03DWwdWzsxK/P2xq4tR2OyzV9Ls4rzL9XU5WVefnGCosA78GsBcLp3a+kh1BFIZ2I"+
		"5xXj2vzLOhhpJ+oRRy6EXYgfMgDjzpk4ZkjGclibBIJ+b9CKjXywIBB4S8jwBcztNPlEAfYjuUzKqKQMqW8QkBr/VNEt6AswDpuYhdnKQLiNh"+
		"s50OdSdISsdhLBPXrhdXXuE8zoYdH34r/iP+Z8yXh3w0qo4Ro1aON+9PC36s7lZ13fcQlVJA3eLKAWSa520wWO52byYpvYqx2PgFt7vJZ3GQA"+
		"7z0UdU2/9yL8FjhH5scd6ssMqXl2l1/Sal2w2KFDiBCVdvjgF+xtnL3G4s/NTbA/4cTCc4VHKIGnwUVOwth9iL+6MMgrY3uphFG2zatAHPoXh"+
		"499SAiVMGo8YxwZdgS8G5FBS1tC7Wbl4hI8tkAjFTkHqL+HOqfqI+JCnA8m5ElLrjdx8M4NCfoqgl6iRrAAuI8On2xFx2PK6qJj6QHTU1X/m0"+
		"Tycozb5G082S7Y3wsLrj/Ma/G3zmrl1d77t0nsCTnrV09vD7/wRNgBnf0dzoSD/tP0gFQd69SWCr7L6xbI3ai0VPzF810iyDmJixmDfQQz1Em"+
		"jSMxgGoV1rpJ+PqtmrLMAr3AsCESNPA7k/Swnss1uuRWOZwYrdTPHKmwv4WGxQ/GYvBxT8ukwB4777yjZB2UETKtuIb4GI4axqu1xr4yoYlPL"+
		"TWuws7bm8ITfPgle0TImlyOjUYGltG6+ynnoxrZKwLGt/eM2yXHKMh6DyOk4/0iHrGuFEI+HAJ2x5UgfdLG6w+Ok4Spty7NbHqTwWrWbHHxCI"+
		"TszKsqh2QJBy15Vp1T1fJ3oXHBagoaBgxrmnf0m++vG38MOkNzbK+TT76cKbhZW3Absg9Isw4/oRooJqJAmo6fBGa98htU4OwJdgv6eeLmwXp"+
		"hYe3nsSGc40W6FH02Nw==")
	for i, test := range []struct {
		key                  []byte
		proof                string
		manifestCommitment   string
		electionPkX          string
		electionPkY          string
		cipherC1X            string
		cipherC1Y            string
		cipherC2X            string
		cipherC2Y            string
		voteKeyCommitment    string
		credentialCommitment string
		callerPK             string
		credentialGeneration int
		credentialActive     int
		electionTag          string
		validUntil           int
		ok                   bool
		error                string
	}{
		{
			key: key1,
			proof: "oaZgFIcIRQ5mgbUEIDoSYVHQb2Yh/6sJOfoSfiSP+BYR2n10T4W1/brIDYQCVpEEGxWIoW6uwxhwxiv4AAGR+CO0Tk/EKco/D" +
				"hLg0n7czS+acdvHEwukHIViLWYBFW/wH3YxknBpCygz8xT9i+Gbczmo825qfz3QPvF5JqaZYCk=",
			manifestCommitment:   "Lp+UaItHy8lwq2x/DzL5nyDWtZGYJRWNRM7yZLsXRzA=",
			electionPkX:          "DPZbLuP6oMmcBk9Hpz0+4rHsLbR0JkKr/gsUND1QKvU=",
			electionPkY:          "GxFyFrf4J8snH+zX0osZpVejfP3/2QsLcdK8nJn/deU=",
			cipherC1X:            "I3jh+q4SWxYMM/TneG40MeAt+sZUFKgn8OSDsAnxsDc=",
			cipherC1Y:            "CV4ebsZ0N0zCjbYg1Db/icg5r2f574mrrBqE8y7sHuE=",
			cipherC2X:            "LuqyiAtBpvNIgsNJ4rOmy6AziVcehbLiTLssuJaTz8k=",
			cipherC2Y:            "KJEVD67r6+5Lw7oC9JQZwKfDYLYE9fcS2BfxpEnrk18=",
			voteKeyCommitment:    "Gm6K4/fDZQcmixEZU/Hi07oPiODfRnruYkqlVsPKs8Q=",
			credentialCommitment: "BBZfAkH6i1UsYMAHCCKzIG8sV5Q1HweHoQvf6ARfTBQ=",
			callerPK:             "7bn3FNpC6UAbwXZ2MxAF13XXZ9wfjM17dGBErZkZ4Qgc",
			credentialGeneration: 1,
			credentialActive:     1,
			electionTag:          "AIncw/T6F6EaS6aTRgY81PpuhN3X67qVkVqzUxwefaA=",
			validUntil:           5376860,
			ok:                   true,
		},
		{ // BidHCWxKgkkUaLF7LEPp1FfA9e7f7g5gX8hHzCdjV5Df
			key: key1,
			proof: "oaZgFIcIRQ5mgbUEIDoSYVHQb2Yh/6sJOfoSfiSP+BYR2n10T4W1/brIDYQCVpEEGxWIoW6uwxhwxiv4AAGR+CO0Tk/EKco/D" +
				"hLg0n7czS+acdvHEwukHIViLWYBFW/wH3YxknBpCygz8xT9i+Gbczmo825qfz3QPvF5JqaZYCk=",
			manifestCommitment:   "Lp+UaItHy8lwq2x/DzL5nyDWtZGYJRWNRM7yZLsXRzA=",
			electionPkX:          "DPZbLuP6oMmcBk9Hpz0+4rHsLbR0JkKr/gsUND1QKvU=",
			electionPkY:          "GxFyFrf4J8snH+zX0osZpVejfP3/2QsLcdK8nJn/deU=",
			cipherC1X:            "I3jh+q4SWxYMM/TneG40MeAt+sZUFKgn8OSDsAnxsDc=",
			cipherC1Y:            "CV4ebsZ0N0zCjbYg1Db/icg5r2f574mrrBqE8y7sHuE=",
			cipherC2X:            "LuqyiAtBpvNIgsNJ4rOmy6AziVcehbLiTLssuJaTz8o=", // different from the first test case
			cipherC2Y:            "KJEVD67r6+5Lw7oC9JQZwKfDYLYE9fcS2BfxpEnrk18=",
			voteKeyCommitment:    "Gm6K4/fDZQcmixEZU/Hi07oPiODfRnruYkqlVsPKs8Q=",
			credentialCommitment: "BBZfAkH6i1UsYMAHCCKzIG8sV5Q1HweHoQvf6ARfTBQ=",
			callerPK:             "7bn3FNpC6UAbwXZ2MxAF13XXZ9wfjM17dGBErZkZ4Qgc",
			credentialGeneration: 1,
			credentialActive:     1,
			electionTag:          "AIncw/T6F6EaS6aTRgY81PpuhN3X67qVkVqzUxwefaA=",
			validUntil:           5376860,
			ok:                   false,
		},
		{ // 3k8iVN9mTVdoB3DywNp536oW9aR7L2ebrxJFB8NnTCfA
			key: key1,
			proof: "oaZgFIcIRQ5mgbUEIDoSYVHQb2Yh/6sJOfoSfiSP+BYR2n10T4W1/brIDYQCVpEEGxWIoW6uwxhwxiv4AAGR+CO0Tk/EKco/D" +
				"hLg0n7czS+acdvHEwukHIViLWYBFW/wH3YxknBpCygz8xT9i+Gbczmo825qfz3QPvF5JqaZYCk=",
			manifestCommitment:   "Lp+UaItHy8lwq2x/DzL5nyDWtZGYJRWNRM7yZLsXRzE=", // different manifest commitment
			electionPkX:          "DPZbLuP6oMmcBk9Hpz0+4rHsLbR0JkKr/gsUND1QKvU=",
			electionPkY:          "GxFyFrf4J8snH+zX0osZpVejfP3/2QsLcdK8nJn/deU=",
			cipherC1X:            "I3jh+q4SWxYMM/TneG40MeAt+sZUFKgn8OSDsAnxsDc=",
			cipherC1Y:            "CV4ebsZ0N0zCjbYg1Db/icg5r2f574mrrBqE8y7sHuE=",
			cipherC2X:            "LuqyiAtBpvNIgsNJ4rOmy6AziVcehbLiTLssuJaTz8k=",
			cipherC2Y:            "KJEVD67r6+5Lw7oC9JQZwKfDYLYE9fcS2BfxpEnrk18=",
			voteKeyCommitment:    "Gm6K4/fDZQcmixEZU/Hi07oPiODfRnruYkqlVsPKs8Q=",
			credentialCommitment: "BBZfAkH6i1UsYMAHCCKzIG8sV5Q1HweHoQvf6ARfTBQ=",
			callerPK:             "7bn3FNpC6UAbwXZ2MxAF13XXZ9wfjM17dGBErZkZ4Qgc",
			credentialGeneration: 1,
			credentialActive:     1,
			electionTag:          "AIncw/T6F6EaS6aTRgY81PpuhN3X67qVkVqzUxwefaA=",
			validUntil:           5376860,
			ok:                   false,
		},
		{ // 4HK4NzBCZWMtgmFESkZooq2sxphtT8UX9s1dzN8B5Kyz
			key: key1,
			proof: "oaZgFIcIRQ5mgbUEIDoSYVHQb2Yh/6sJOfoSfiSP+BYR2n10T4W1/brIDYQCVpEEGxWIoW6uwxhwxiv4AAGR+CO0Tk/EKco/D" +
				"hLg0n7czS+acdvHEwukHIViLWYBFW/wH3YxknBpCygz8xT9i+Gbczmo825qfz3QPvF5JqaZYCk=",
			manifestCommitment:   "Lp+UaItHy8lwq2x/DzL5nyDWtZGYJRWNRM7yZLsXRzA=",
			electionPkX:          "DPZbLuP6oMmcBk9Hpz0+4rHsLbR0JkKr/gsUND1QKvY=", // different election public key X
			electionPkY:          "GxFyFrf4J8snH+zX0osZpVejfP3/2QsLcdK8nJn/deU=",
			cipherC1X:            "I3jh+q4SWxYMM/TneG40MeAt+sZUFKgn8OSDsAnxsDc=",
			cipherC1Y:            "CV4ebsZ0N0zCjbYg1Db/icg5r2f574mrrBqE8y7sHuE=",
			cipherC2X:            "LuqyiAtBpvNIgsNJ4rOmy6AziVcehbLiTLssuJaTz8k=",
			cipherC2Y:            "KJEVD67r6+5Lw7oC9JQZwKfDYLYE9fcS2BfxpEnrk18=",
			voteKeyCommitment:    "Gm6K4/fDZQcmixEZU/Hi07oPiODfRnruYkqlVsPKs8Q=",
			credentialCommitment: "BBZfAkH6i1UsYMAHCCKzIG8sV5Q1HweHoQvf6ARfTBQ=",
			callerPK:             "7bn3FNpC6UAbwXZ2MxAF13XXZ9wfjM17dGBErZkZ4Qgc",
			credentialGeneration: 1,
			credentialActive:     1,
			electionTag:          "AIncw/T6F6EaS6aTRgY81PpuhN3X67qVkVqzUxwefaA=",
			validUntil:           5376860,
			ok:                   false,
		},
		{ // 9fQLkWG1p5ZQAwmtTZSMi6TUq7gtwrksv9daTFumFgfP
			key: key1,
			proof: "oaZgFIcIRQ5mgbUEIDoSYVHQb2Yh/6sJOfoSfiSP+BYR2n10T4W1/brIDYQCVpEEGxWIoW6uwxhwxiv4AAGR+CO0Tk/EKco/D" +
				"hLg0n7czS+acdvHEwukHIViLWYBFW/wH3YxknBpCygz8xT9i+Gbczmo825qfz3QPvF5JqaZYCk=",
			manifestCommitment:   "Lp+UaItHy8lwq2x/DzL5nyDWtZGYJRWNRM7yZLsXRzA=",
			electionPkX:          "DPZbLuP6oMmcBk9Hpz0+4rHsLbR0JkKr/gsUND1QKvU=",
			electionPkY:          "GxFyFrf4J8snH+zX0osZpVejfP3/2QsLcdK8nJn/deU=",
			cipherC1X:            "I3jh+q4SWxYMM/TneG40MeAt+sZUFKgn8OSDsAnxsDc=",
			cipherC1Y:            "CV4ebsZ0N0zCjbYg1Db/icg5r2f574mrrBqE8y7sHuE=",
			cipherC2X:            "LuqyiAtBpvNIgsNJ4rOmy6AziVcehbLiTLssuJaTz8k=",
			cipherC2Y:            "KJEVD67r6+5Lw7oC9JQZwKfDYLYE9fcS2BfxpEnrk18=",
			voteKeyCommitment:    "Gm6K4/fDZQcmixEZU/Hi07oPiODfRnruYkqlVsPKs8U=", // different from the first test case
			credentialCommitment: "BBZfAkH6i1UsYMAHCCKzIG8sV5Q1HweHoQvf6ARfTBQ=",
			callerPK:             "7bn3FNpC6UAbwXZ2MxAF13XXZ9wfjM17dGBErZkZ4Qgc",
			credentialGeneration: 1,
			credentialActive:     1,
			electionTag:          "AIncw/T6F6EaS6aTRgY81PpuhN3X67qVkVqzUxwefaA=",
			validUntil:           5376860,
			ok:                   false,
		},
		{ // GSgaRvEPvfumGBWLFrwJWvnDE6v8ZumMECZW4MpYE299
			key: key1,
			proof: "oaZgFIcIRQ5mgbUEIDoSYVHQb2Yh/6sJOfoSfiSP+BYR2n10T4W1/brIDYQCVpEEGxWIoW6uwxhwxiv4AAGR+CO0Tk/EKco/D" +
				"hLg0n7czS+acdvHEwukHIViLWYBFW/wH3YxknBpCygz8xT9i+Gbczmo825qfz3QPvF5JqaZYCk=",
			manifestCommitment:   "Lp+UaItHy8lwq2x/DzL5nyDWtZGYJRWNRM7yZLsXRzA=",
			electionPkX:          "DPZbLuP6oMmcBk9Hpz0+4rHsLbR0JkKr/gsUND1QKvU=",
			electionPkY:          "GxFyFrf4J8snH+zX0osZpVejfP3/2QsLcdK8nJn/deU=",
			cipherC1X:            "I3jh+q4SWxYMM/TneG40MeAt+sZUFKgn8OSDsAnxsDc=",
			cipherC1Y:            "CV4ebsZ0N0zCjbYg1Db/icg5r2f574mrrBqE8y7sHuE=",
			cipherC2X:            "LuqyiAtBpvNIgsNJ4rOmy6AziVcehbLiTLssuJaTz8k=",
			cipherC2Y:            "KJEVD67r6+5Lw7oC9JQZwKfDYLYE9fcS2BfxpEnrk18=",
			voteKeyCommitment:    "Gm6K4/fDZQcmixEZU/Hi07oPiODfRnruYkqlVsPKs8Q=",
			credentialCommitment: "BBZfAkH6i1UsYMAHCCKzIG8sV5Q1HweHoQvf6ARfTBU=", // different from the first test case
			callerPK:             "7bn3FNpC6UAbwXZ2MxAF13XXZ9wfjM17dGBErZkZ4Qgc",
			credentialGeneration: 1,
			credentialActive:     1,
			electionTag:          "AIncw/T6F6EaS6aTRgY81PpuhN3X67qVkVqzUxwefaA=",
			validUntil:           5376860,
			ok:                   false,
		},
		{ // HNPKjyfoDFhLywPtSRn1G41kY91ToaF3iPV1YtWKSBwh
			key: key1,
			proof: "GjSFejiiRt5q1nYtuZviUw8LyJLp+eFI35/BmxVrAGAXQ0sOg910MhU6xBmEFUpujBfWQr+IcBISm24JrpQwgSt0jZiqZCKRe" +
				"edXAujRayxS4cdbMe+Mxr12UPQ6ObrOKFDQGKcG33k4NojS6RR+HVlBCXX5SLMTyLGX2fClajo=", // different proof
			manifestCommitment:   "Lp+UaItHy8lwq2x/DzL5nyDWtZGYJRWNRM7yZLsXRzA=",
			electionPkX:          "DPZbLuP6oMmcBk9Hpz0+4rHsLbR0JkKr/gsUND1QKvU=",
			electionPkY:          "GxFyFrf4J8snH+zX0osZpVejfP3/2QsLcdK8nJn/deU=",
			cipherC1X:            "I3jh+q4SWxYMM/TneG40MeAt+sZUFKgn8OSDsAnxsDc=",
			cipherC1Y:            "CV4ebsZ0N0zCjbYg1Db/icg5r2f574mrrBqE8y7sHuE=",
			cipherC2X:            "LuqyiAtBpvNIgsNJ4rOmy6AziVcehbLiTLssuJaTz8k=",
			cipherC2Y:            "KJEVD67r6+5Lw7oC9JQZwKfDYLYE9fcS2BfxpEnrk18=",
			voteKeyCommitment:    "Gm6K4/fDZQcmixEZU/Hi07oPiODfRnruYkqlVsPKs8Q=",
			credentialCommitment: "FD7GHhEKoTvlON5RQi691uije5AQsaqW9PX14aewqBc=", // different from the first test case
			callerPK:             "7bn3FNpC6UAbwXZ2MxAF13XXZ9wfjM17dGBErZkZ4Qgc",
			credentialGeneration: 1,
			credentialActive:     1,
			electionTag:          "AIncw/T6F6EaS6aTRgY81PpuhN3X67qVkVqzUxwefaA=",
			validUntil:           5376860,
			ok:                   false,
		},
		{ // AV46CaYgTBVNMDAdjTGjLfWP76CUR245P8D1nBijU6WM
			key: key1,
			proof: "lnpGigtn43ETDDkp1iD26S1AADatUQUBLwo3tW9f6+wNJMDjWTYvh6xtsbPxbSndvS16fJBUB2Er995VhToKayZ9pjx6Nl9b9" +
				"U6COtK8HLKleFPysgkBlw0y8VqbQB7ersNIuOgszUBNviyKHf09BgfKvudqGcAjoymg1X2gHcY=", // different proof
			manifestCommitment:   "Lp+UaItHy8lwq2x/DzL5nyDWtZGYJRWNRM7yZLsXRzA=",
			electionPkX:          "DPZbLuP6oMmcBk9Hpz0+4rHsLbR0JkKr/gsUND1QKvU=",
			electionPkY:          "GxFyFrf4J8snH+zX0osZpVejfP3/2QsLcdK8nJn/deU=",
			cipherC1X:            "I3jh+q4SWxYMM/TneG40MeAt+sZUFKgn8OSDsAnxsDc=",
			cipherC1Y:            "CV4ebsZ0N0zCjbYg1Db/icg5r2f574mrrBqE8y7sHuE=",
			cipherC2X:            "LuqyiAtBpvNIgsNJ4rOmy6AziVcehbLiTLssuJaTz8k=",
			cipherC2Y:            "KJEVD67r6+5Lw7oC9JQZwKfDYLYE9fcS2BfxpEnrk18=",
			voteKeyCommitment:    "Gm6K4/fDZQcmixEZU/Hi07oPiODfRnruYkqlVsPKs8Q=",
			credentialCommitment: "EhWbvMc3G1pgmytDPT+mRnbEi7d6GFpxm13whdVodZQ=", // different from the first test case
			callerPK:             "7bn3FNpC6UAbwXZ2MxAF13XXZ9wfjM17dGBErZkZ4Qgc",
			credentialGeneration: 2, // different from the first test case
			credentialActive:     1,
			electionTag:          "AIncw/T6F6EaS6aTRgY81PpuhN3X67qVkVqzUxwefaA=",
			validUntil:           5376860,
			ok:                   false,
		},
		{ // 8N4ahnuVKmUVEGTfHxDRbH9UEQF5t4Hu58HxeKNGkfSL
			key: key1,
			proof: "oaZgFIcIRQ5mgbUEIDoSYVHQb2Yh/6sJOfoSfiSP+BYR2n10T4W1/brIDYQCVpEEGxWIoW6uwxhwxiv4AAGR+CO0Tk/EKco/D" +
				"hLg0n7czS+acdvHEwukHIViLWYBFW/wH3YxknBpCygz8xT9i+Gbczmo825qfz3QPvF5JqaZYCk=",
			manifestCommitment:   "Lp+UaItHy8lwq2x/DzL5nyDWtZGYJRWNRM7yZLsXRzA=",
			electionPkX:          "DPZbLuP6oMmcBk9Hpz0+4rHsLbR0JkKr/gsUND1QKvU=",
			electionPkY:          "GxFyFrf4J8snH+zX0osZpVejfP3/2QsLcdK8nJn/deU=",
			cipherC1X:            "I3jh+q4SWxYMM/TneG40MeAt+sZUFKgn8OSDsAnxsDc=",
			cipherC1Y:            "CV4ebsZ0N0zCjbYg1Db/icg5r2f574mrrBqE8y7sHuE=",
			cipherC2X:            "LuqyiAtBpvNIgsNJ4rOmy6AziVcehbLiTLssuJaTz8k=",
			cipherC2Y:            "KJEVD67r6+5Lw7oC9JQZwKfDYLYE9fcS2BfxpEnrk18=",
			voteKeyCommitment:    "Gm6K4/fDZQcmixEZU/Hi07oPiODfRnruYkqlVsPKs8Q=",
			credentialCommitment: "BBZfAkH6i1UsYMAHCCKzIG8sV5Q1HweHoQvf6ARfTBQ=",
			callerPK:             "7bn3FNpC6UAbwXZ2MxAF13XXZ9wfjM17dGBErZkZ4Qgc",
			credentialGeneration: 1,
			credentialActive:     0, // different from the first test case
			electionTag:          "AIncw/T6F6EaS6aTRgY81PpuhN3X67qVkVqzUxwefaA=",
			validUntil:           5376860,
			ok:                   false,
		},
		{ // 6w8To3owEbrv71eyokgEdrVXjzdipm3WZH9TowTP11XF (should not fail, height verification in script fails)
			key: key1,
			proof: "qUda8AjcDFtOKknmsxcYNuzC4PHa/Edv74+y2brlSKsf/8psywiEvKTdAIv3ou2cD7acL0lEGU8GQ6s/FyMHpyrWybru09PBl" +
				"GMj9q2ozvywdxlbb89s2MVBgL9BvdmEJAc0n12a3NXdjl4IWutR75WDFRl4G6I3JmJsCfbqiNg=", // different proof
			manifestCommitment:   "Lp+UaItHy8lwq2x/DzL5nyDWtZGYJRWNRM7yZLsXRzA=",
			electionPkX:          "DPZbLuP6oMmcBk9Hpz0+4rHsLbR0JkKr/gsUND1QKvU=",
			electionPkY:          "GxFyFrf4J8snH+zX0osZpVejfP3/2QsLcdK8nJn/deU=",
			cipherC1X:            "I3jh+q4SWxYMM/TneG40MeAt+sZUFKgn8OSDsAnxsDc=",
			cipherC1Y:            "CV4ebsZ0N0zCjbYg1Db/icg5r2f574mrrBqE8y7sHuE=",
			cipherC2X:            "LuqyiAtBpvNIgsNJ4rOmy6AziVcehbLiTLssuJaTz8k=",
			cipherC2Y:            "KJEVD67r6+5Lw7oC9JQZwKfDYLYE9fcS2BfxpEnrk18=",
			voteKeyCommitment:    "Gm6K4/fDZQcmixEZU/Hi07oPiODfRnruYkqlVsPKs8Q=",
			credentialCommitment: "L6dFmoscOjPKUHw/7u+PqezJatFtJgevlzoaiFMm/Lc=", // different from the first test case
			callerPK:             "7bn3FNpC6UAbwXZ2MxAF13XXZ9wfjM17dGBErZkZ4Qgc",
			credentialGeneration: 1,
			credentialActive:     1,
			electionTag:          "AIncw/T6F6EaS6aTRgY81PpuhN3X67qVkVqzUxwefaA=",
			validUntil:           5374859, // different from the first test case
			ok:                   true,
		},
		{ // 8KQnsvBTM7oJ3xG5oEQuLVSavsX2F5dnSp1nd71mGtTx
			key: key1,
			proof: "oaZgFIcIRQ5mgbUEIDoSYVHQb2Yh/6sJOfoSfiSP+BYR2n10T4W1/brIDYQCVpEEGxWIoW6uwxhwxiv4AAGR+CO0Tk/EKco/D" +
				"hLg0n7czS+acdvHEwukHIViLWYBFW/wH3YxknBpCygz8xT9i+Gbczmo825qfz3QPvF5JqaZYCg=", // tampered proof
			manifestCommitment:   "Lp+UaItHy8lwq2x/DzL5nyDWtZGYJRWNRM7yZLsXRzA=",
			electionPkX:          "DPZbLuP6oMmcBk9Hpz0+4rHsLbR0JkKr/gsUND1QKvU=",
			electionPkY:          "GxFyFrf4J8snH+zX0osZpVejfP3/2QsLcdK8nJn/deU=",
			cipherC1X:            "I3jh+q4SWxYMM/TneG40MeAt+sZUFKgn8OSDsAnxsDc=",
			cipherC1Y:            "CV4ebsZ0N0zCjbYg1Db/icg5r2f574mrrBqE8y7sHuE=",
			cipherC2X:            "LuqyiAtBpvNIgsNJ4rOmy6AziVcehbLiTLssuJaTz8k=",
			cipherC2Y:            "KJEVD67r6+5Lw7oC9JQZwKfDYLYE9fcS2BfxpEnrk18=",
			voteKeyCommitment:    "Gm6K4/fDZQcmixEZU/Hi07oPiODfRnruYkqlVsPKs8Q=",
			credentialCommitment: "BBZfAkH6i1UsYMAHCCKzIG8sV5Q1HweHoQvf6ARfTBQ=",
			callerPK:             "7bn3FNpC6UAbwXZ2MxAF13XXZ9wfjM17dGBErZkZ4Qgc",
			credentialGeneration: 1,
			credentialActive:     1,
			electionTag:          "AIncw/T6F6EaS6aTRgY81PpuhN3X67qVkVqzUxwefaA=",
			validUntil:           5376860,
			ok:                   false,
		},
		{ // AEu8xoyfdDaBZbkhbD3icB5nkFMnkMYbgKMhN5cv5v9q
			key: key2,
			proof: "haEhuokoaENQViF9yhkTxMe/BrRkQ3BRSU1TF7CIJqWBZ1IpxM2e0bbF8BGPq/mk82UEwgiBO7Tx3Gp/VP/DxSghybcuhpQEH" +
				"eywJZhLLQpKvf0AZOFiAfaxLDOzT0FJnptxU5jXqyzDjQqBvszZGBH8TyYMnTEhDlSG6SqqI4s=",
			manifestCommitment:   "Lp+UaItHy8lwq2x/DzL5nyDWtZGYJRWNRM7yZLsXRzA=",
			electionPkX:          "DPZbLuP6oMmcBk9Hpz0+4rHsLbR0JkKr/gsUND1QKvU=",
			electionPkY:          "GxFyFrf4J8snH+zX0osZpVejfP3/2QsLcdK8nJn/deU=",
			cipherC1X:            "I3jh+q4SWxYMM/TneG40MeAt+sZUFKgn8OSDsAnxsDc=",
			cipherC1Y:            "CV4ebsZ0N0zCjbYg1Db/icg5r2f574mrrBqE8y7sHuE=",
			cipherC2X:            "LuqyiAtBpvNIgsNJ4rOmy6AziVcehbLiTLssuJaTz8k=",
			cipherC2Y:            "KJEVD67r6+5Lw7oC9JQZwKfDYLYE9fcS2BfxpEnrk18=",
			voteKeyCommitment:    "Gm6K4/fDZQcmixEZU/Hi07oPiODfRnruYkqlVsPKs8Q=",
			credentialCommitment: "CumonWB08VU/atKh333I3hLqbJlTz7iw1ujD9Ps4cXc=",
			callerPK:             "Bn9xtZoPoFCZ5foA3H9uhfTSb7fo1yW1daUPGqRv3s26",
			credentialGeneration: 1,
			credentialActive:     1,
			electionTag:          "AIncw/T6F6EaS6aTRgY81PpuhN3X67qVkVqzUxwefaA=",
			validUntil:           5375567,
			ok:                   false,
			error:                "invalid compressed coordinate: square root doesn't exist",
		},
	} {
		t.Run(nameTest(test.ok, i), func(t *testing.T) {
			proof := deB64(t, test.proof)
			inputs := buildInputs(t, test.manifestCommitment, test.electionPkX, test.electionPkY, test.cipherC1X,
				test.cipherC1Y, test.cipherC2X, test.cipherC2Y, test.voteKeyCommitment, test.credentialCommitment,
				test.callerPK, test.credentialGeneration, 1, test.credentialActive,
				test.electionTag, test.validUntil)
			result, err := Groth16Verify(test.key, proof, inputs, ecc.BN254)
			if test.ok {
				assert.True(t, result)
				assert.NoError(t, err)
			} else {
				assert.False(t, result)
				if test.error != "" {
					assert.Error(t, err)
					assert.EqualError(t, err, test.error)
				}
			}
		})
	}
}

func nameTest(ok bool, i int) string {
	name := "fail"
	if ok {
		name = "ok"
	}
	return fmt.Sprintf("%s_%d", name, i+1)
}

func deB64(t testing.TB, b64 string) []byte {
	data, err := base64.StdEncoding.DecodeString(b64)
	require.NoError(t, err)
	return data
}

func intField(i int) []byte {
	b := make([]byte, 32)
	binary.BigEndian.PutUint64(b[24:], uint64(i))
	return b
}

func callerField(t testing.TB, pk string) []byte {
	pkb, err := NewPublicKeyFromBase58(pk)
	require.NoError(t, err)
	d := sha256.Sum256(pkb.Bytes())
	d[0] = 0
	return d[:]
}

func buildInputs(t *testing.T,
	manifestCommitment, electionPkX, electionPkY, cipherC1X, cipherC1Y, cipherC2X, cipherC2Y, voteKeyCommitment,
	credentialCommitment, callerPK string,
	credentialGeneration, currentCredentialGeneration, credentialActive int,
	electionTag string,
	validUntil int) []byte {
	var inputs []byte
	inputs = append(inputs, deB64(t, manifestCommitment)...)
	inputs = append(inputs, deB64(t, electionPkX)...)
	inputs = append(inputs, deB64(t, electionPkY)...)
	inputs = append(inputs, deB64(t, cipherC1X)...)
	inputs = append(inputs, deB64(t, cipherC1Y)...)
	inputs = append(inputs, deB64(t, cipherC2X)...)
	inputs = append(inputs, deB64(t, cipherC2Y)...)
	inputs = append(inputs, deB64(t, voteKeyCommitment)...)
	inputs = append(inputs, deB64(t, credentialCommitment)...)
	inputs = append(inputs, callerField(t, callerPK)...)
	inputs = append(inputs, intField(credentialGeneration)...)
	inputs = append(inputs, intField(currentCredentialGeneration)...)
	inputs = append(inputs, intField(credentialActive)...)
	inputs = append(inputs, deB64(t, electionTag)...)
	inputs = append(inputs, intField(validUntil)...)
	return inputs
}
