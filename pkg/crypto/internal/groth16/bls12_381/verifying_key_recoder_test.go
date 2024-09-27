package bls12381_test

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/consensys/gnark-crypto/ecc"
	gnark "github.com/consensys/gnark/backend/groth16"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bls12381 "github.com/wavesplatform/gowaves/pkg/crypto/internal/groth16/bls12_381"
)

func TestVerifyingKeyRecoding(t *testing.T) {
	for i, test := range []struct {
		our   string
		their string
	}{
		{
			"hwk883gUlTKCyXYA6XWZa8H9/xKIYZaJ0xEs0M5hQOMxiGpxocuX/8maSDmeCk3bo5ViaDBdO7ZBxAhLSe5k/5TFQyF5Lv7KN2t" +
				"LKnwgoWMqB16OL8WdbePIwTCuPtJNAFKoTZylLDbSf02kckMcZQDPF9iGh+JC99Pio74vDpwTEjUx5tQ99gNQwxULtztsqDRsPnE" +
				"vKvLmsxHt8LQVBkEBm2PBJFY+OXf1MNW021viDBpR10mX4WQ6zrsGL5L0GY4cwf4tlbh+Obit+LnN/SQTnREf8fPpdKZ1sa/ui3p" +
				"Gi8lMT6io4D7Ujlwx2RdCkBF+isfMf77HCEGsZANw0hSrO2FGg14Sl26xLAIohdaW8O7gEaag8JdVAZ3OVLd5Df1NkZBEr753Xb8W" +
				"waXsJjE7qxwINL1KdqA4+EiYW4edb7+a9bbBeOPtb67ZxmFqgyTNS/4obxahezNkjk00ytswsENg//Ee6dWBJZyLH+QGsaU2jO/W4W" +
				"vRyZhmKKPdipOhiz4Rlrd2XYgsfHsfWf5v4GOTL+13ZB24dW1/m39n2woJ+v686fXbNW85XP/r",
			"hwk883gUlTKCyXYA6XWZa8H9/xKIYZaJ0xEs0M5hQOMxiGpxocuX/8maSDmeCk3bhwk883gUlTKCyXYA6XWZa8H9/xKIYZaJ" +
				"0xEs0M5hQOMxiGpxocuX/8maSDmeCk3bo5ViaDBdO7ZBxAhLSe5k/5TFQyF5Lv7KN2tLKnwgoWMqB16OL8WdbePIwTCuPtJNAF" +
				"KoTZylLDbSf02kckMcZQDPF9iGh+JC99Pio74vDpwTEjUx5tQ99gNQwxULtztsqDRsPnEvKvLmsxHt8LQVBkEBm2PBJFY+OXf1M" +
				"NW021viDBpR10mX4WQ6zrsGL5L0GY4cwf4tlbh+Obit+LnN/SQTnREf8fPpdKZ1sa/ui3pGi8lMT6io4D7Ujlwx2RdChwk883gU" +
				"lTKCyXYA6XWZa8H9/xKIYZaJ0xEs0M5hQOMxiGpxocuX/8maSDmeCk3bkBF+isfMf77HCEGsZANw0hSrO2FGg14Sl26xLAIohda" +
				"W8O7gEaag8JdVAZ3OVLd5Df1NkZBEr753Xb8WwaXsJjE7qxwINL1KdqA4+EiYW4edb7+a9bbBeOPtb67ZxmFqAAAAAoMkzUv+KG" +
				"8WoXszZI5NNMrbMLBDYP/xHunVgSWcix/kBrGlNozv1uFr0cmYZiij3YqToYs+EZa3dl2ILHx7H1n+b+Bjky/td2QduHVtf5t/Z9" +
				"sKCfr+vOn12zVvOVz/6wAAAAAAAAAA",
		},
		{
			"kYYCAS8vM2T99GeCr4toQ+iQzvl5fI89mPrncYqx3C1d75BQbFk8LMtcnLWwntd6knkzSwcsialcheg69eZYPK8EzKRVI5FrR" +
				"HKi8rgB+R5jyPV70ejmYEx1neTmfYKODRmARr/ld6pZTzBWYDfrCkiS1QB+3q3M08OQgYcLzs/vjW4epetDCmk0K1CEGcWdh7y" +
				"Lzdqr7HHQNOpZI8mdj/7lR0IBqB9zvRfyTr+guUG22kZo4y2KINDp272xGglKEeTglTxyDUriZJNF/+T6F8w70MR/rV+flvuo6" +
				"EJ0+HA+A2ZnBbTjOIl9wjisBV+0jgld4oAppAOzvQ7eoIx2tbuuKVSdbJm65KDxl/T+boaYnjRm3omdETYnYRk3HAhrAeWpefX" +
				"+dM/k7PrcheInnxHUyjzSzqlN03xYjg28kdda9FZJaVsQKqdEJ/St9ivXlp7+dPDIOfm77haSFnvr33VwYH/KbIalfOJPRvBLzq" +
				"lHD8BxunNebMr6Gr6S+u+n",
			"kYYCAS8vM2T99GeCr4toQ+iQzvl5fI89mPrncYqx3C1d75BQbFk8LMtcnLWwntd6kYYCAS8vM2T99GeCr4toQ+iQzvl5fI89" +
				"mPrncYqx3C1d75BQbFk8LMtcnLWwntd6knkzSwcsialcheg69eZYPK8EzKRVI5FrRHKi8rgB+R5jyPV70ejmYEx1neTmfYKODR" +
				"mARr/ld6pZTzBWYDfrCkiS1QB+3q3M08OQgYcLzs/vjW4epetDCmk0K1CEGcWdh7yLzdqr7HHQNOpZI8mdj/7lR0IBqB9zvRfy" +
				"Tr+guUG22kZo4y2KINDp272xGglKEeTglTxyDUriZJNF/+T6F8w70MR/rV+flvuo6EJ0+HA+A2ZnBbTjOIl9wjisBV+0kYYCAS" +
				"8vM2T99GeCr4toQ+iQzvl5fI89mPrncYqx3C1d75BQbFk8LMtcnLWwntd6jgld4oAppAOzvQ7eoIx2tbuuKVSdbJm65KDxl/T+" +
				"boaYnjRm3omdETYnYRk3HAhrAeWpefX+dM/k7PrcheInnxHUyjzSzqlN03xYjg28kdda9FZJaVsQKqdEJ/St9ivXAAAAAZae/n" +
				"TwyDn5u+4WkhZ76991cGB/ymyGpXziT0bwS86pRw/AcbpzXmzK+hq+kvrvpwAAAAAAAAAA",
		},
		{
			"mY//hEITCBCZUJUN/wsOlw1iUSSOESL6PFSbN1abGK80t5jPNICNlPuSorio4mmWpf+4uOyv3gPZe54SYGM4pfhteqJpwFQxd" +
				"lpwXWyYxMTNaSLDj8VtSn/EJaSu+P6nFmWsda3mTYUPYMZzWE4hMqpDgFPcJhw3prArMThDPbR3Hx7E6NRAAR0LqcrdtsbDqu" +
				"2T0tto1rpnFILdvHL4PqEUfTmF2mkM+DKj7lKwvvZUbukqBwLrnnbdfyqZJryzGAMIa2JvMEMYszGsYyiPXZvYx6Luk54oWOl" +
				"OrwEKrCY4NMPwch6DbFq6KpnNSQwOpgRYCz7wpjk57X+NGJmo85tYKc+TNa1rT4/DxG9v6SHkpXmmPeHhzIIW8MOdkFjxB5o6Q" +
				"n8Fa0c6Tt6br2gzkrGr1eK5/+RiIgEzVhcRrqdY/p7PLmKXqawrEvIv9QZ3ijytPNwinlC8XdRLO/YvP33PjcI9WSMcHV6POP9" +
				"KPMo1rngaIPMegKgAvTEouNFKp4v3wAXRXX5xEjwXAmM5wyB/SAOaPPCK/emls9kqolHsaj7nuTTbrvSV8bqzUwzQ",
			"mY//hEITCBCZUJUN/wsOlw1iUSSOESL6PFSbN1abGK80t5jPNICNlPuSorio4mmWmY//hEITCBCZUJUN/wsOlw1iUSSOESL6" +
				"PFSbN1abGK80t5jPNICNlPuSorio4mmWpf+4uOyv3gPZe54SYGM4pfhteqJpwFQxdlpwXWyYxMTNaSLDj8VtSn/EJaSu+P6nFm" +
				"Wsda3mTYUPYMZzWE4hMqpDgFPcJhw3prArMThDPbR3Hx7E6NRAAR0LqcrdtsbDqu2T0tto1rpnFILdvHL4PqEUfTmF2mkM+DKj" +
				"7lKwvvZUbukqBwLrnnbdfyqZJryzGAMIa2JvMEMYszGsYyiPXZvYx6Luk54oWOlOrwEKrCY4NMPwch6DbFq6KpnNSQwOmY//hE" +
				"ITCBCZUJUN/wsOlw1iUSSOESL6PFSbN1abGK80t5jPNICNlPuSorio4mmWpgRYCz7wpjk57X+NGJmo85tYKc+TNa1rT4/DxG9v6" +
				"SHkpXmmPeHhzIIW8MOdkFjxB5o6Qn8Fa0c6Tt6br2gzkrGr1eK5/+RiIgEzVhcRrqdY/p7PLmKXqawrEvIv9QZ3AAAAAoo8rTzc" +
				"Ip5QvF3USzv2Lz99z43CPVkjHB1ejzj/SjzKNa54GiDzHoCoAL0xKLjRSqeL98AF0V1+cRI8FwJjOcMgf0gDmjzwiv3ppbPZKqJ" +
				"R7Go+57k02670lfG6s1MM0AAAAAAAAAAA",
		},
		{
			"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" +
				"nreKPLzN2IwJoDp6wKOySKQAAAAAAAAAA",
		},
		{
			"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",
		},
		{
			"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",
		},
	} {
		t.Run(fmt.Sprintf("#%d", i+1), func(t *testing.T) {
			b64r := base64.NewDecoder(base64.StdEncoding, bytes.NewReader([]byte(test.our)))
			vk := new(bls12381.BellmanVerifyingKeyBl12381)
			_, err := vk.ReadFrom(b64r)
			require.NoError(t, err, "failed to read VK")

			buf := new(bytes.Buffer)
			_, err = vk.WriteTo(buf)
			require.NoError(t, err, "failed to write gnark VK")
			assert.Equal(t, test.their, base64.StdEncoding.EncodeToString(buf.Bytes()))

			gvk := gnark.NewVerifyingKey(ecc.BLS12_381) // Read grank VK again using gnark.
			_, err = gvk.ReadFrom(bytes.NewReader(buf.Bytes()))
			require.NoError(t, err, "failed to read gnark VK")

			buf = new(bytes.Buffer)
			b64w := base64.NewEncoder(base64.StdEncoding, buf)
			_, err = gvk.WriteTo(b64w)
			require.NoError(t, err, "failed to write gnark VK")
			assert.Equal(t, test.their, buf.String())
		})
	}
}
