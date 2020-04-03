package crypto

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroth16Verify(t *testing.T) {
	t.SkipNow()
	for _, test := range []struct {
		vk     string
		proof  string
		inputs string
		ok     bool
	}{
		//{"L6bNoQF7E5Hun46AwObA3rTiq0IJwgGoRv/ZJeeKZ5AY6QtPJDxjcYZWfu+X5CvP4/Tt3wMuEOcHtWLHrSYFxyzL/Pil2zI3ny8u/P7RrqVEate9yfukNDS9kOETgVssJq4jY7h1PMS9j01cGdO9a/3loe+frI5uNFPMyd18TW4IkUaodJXL2t9sGtSTyjYOOLZUMNtIXsIxuD76hTqZFRsdvsZsTj9XL4ZYWZL/quzmrxIJFcl2PJBGifk2kBpRDStINTUNaVXhkJL9Ey0DFwr/3U5xYEZjFvRANoXlbB4FU2Ew+EIq61USmD95LaupQ4MHnplxXVbOr2qdrvX+AyXaw9BLvV2/k7+z8yqBldDXBcRqB7kikhAEMDJ6Z91AG/Mhv2HcqqxPRoeEOFamzN9owKxu4Oy0SAEbKNaJ1ugSfL8V1LTuXSlwPgnLVSxvPv1QmbPqicPoiJ2++vfGrxpRzFtB/WwNnoRti0+nATVhRCIE928bY//YlkV/YFZhB55xQSsANucfvvZW79AUp8Qi7DO2/DvUgZoaCcij2WwZQl8N40cQS1I3cFeqZ8h/QmBc/BM20Y/+PQLnM7b3RAsUmA7yIbNFx9gxpilce/9TMY9GUyvd7cZGMU6hI5mfLevhQuG69WH/3znUQcUAwFZWENOUXB9Irqn7eoAHTcYTdzQLdnf24zPbP/UX48cVl1KYg/9SsJTLDHm1lKG05B/mqzfXlgfQwRfILUR40U4dL/nHfpUC8sdtacc5vL43",
		//	"FE/tn5hYgC97hG66O4v80DJYkMtJVO322XqNIC/o3JEbmDoEz4p+m8PibldJi+Jmja6vHErHz2JM8hv9HhfnwwG7+UrXx4jKftue7s2ZGgfh08pNwvirFvyeW3niYBZoIsPouBzzyguxsftNEnd2BYcUkv4zqrLqvCRiqzWKNQYvH7C15PSyI7gVbGXmSvVOOxKMkJLMKAUie9kWfs+I/A+H0ERPRHe8o10yR8VGKj4K+SdIYvKwEKMkHtZ15lkYClInVZk8s8RqQZ7Dm9QY3GtaVkKv0NYIfjUH3FHeVUkPHfDmrKY0XS0pDXLma3JryxGRiJ5+n9CuJbwc4785JA==",
		//	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABs=", true},
		{"hwk883gUlTKCyXYA6XWZa8H9/xKIYZaJ0xEs0M5hQOMxiGpxocuX/8maSDmeCk3bo5ViaDBdO7ZBxAhLSe5k/5TFQyF5Lv7KN2tLKnwgoWMqB16OL8WdbePIwTCuPtJNAFKoTZylLDbSf02kckMcZQDPF9iGh+JC99Pio74vDpwTEjUx5tQ99gNQwxULtztsqDRsPnEvKvLmsxHt8LQVBkEBm2PBJFY+OXf1MNW021viDBpR10mX4WQ6zrsGL5L0GY4cwf4tlbh+Obit+LnN/SQTnREf8fPpdKZ1sa/ui3pGi8lMT6io4D7Ujlwx2RdCkBF+isfMf77HCEGsZANw0hSrO2FGg14Sl26xLAIohdaW8O7gEaag8JdVAZ3OVLd5Df1NkZBEr753Xb8WwaXsJjE7qxwINL1KdqA4+EiYW4edb7+a9bbBeOPtb67ZxmFqgyTNS/4obxahezNkjk00ytswsENg//Ee6dWBJZyLH+QGsaU2jO/W4WvRyZhmKKPdipOhiz4Rlrd2XYgsfHsfWf5v4GOTL+13ZB24dW1/m39n2woJ+v686fXbNW85XP/r",
			"lvQLU/KqgFhsLkt/5C/scqs7nWR+eYtyPdWiLVBux9GblT4AhHYMdCgwQfSJcudvsgV6fXoK+DUSRgJ++Nqt+Wvb7GlYlHpxCysQhz26TTu8Nyo7zpmVPH92+UYmbvbQCSvX2BhWtvkfHmqDVjmSIQ4RUMfeveA1KZbSf999NE4qKK8Do+8oXcmTM4LZVmh1rlyqznIdFXPN7x3pD4E0gb6/y69xtWMChv9654FMg05bAdueKt9uA4BEcAbpkdHF",
			"LcMT3OOlkHLzJBKCKjjzzVMg+r+FVgd52LlhZPB4RFg=", true},
		{"hwk883gUlTKCyXYA6XWZa8H9/xKIYZaJ0xEs0M5hQOMxiGpxocuX/8maSDmeCk3bo5ViaDBdO7ZBxAhLSe5k/5TFQyF5Lv7KN2tLKnwgoWMqB16OL8WdbePIwTCuPtJNAFKoTZylLDbSf02kckMcZQDPF9iGh+JC99Pio74vDpwTEjUx5tQ99gNQwxULtztsqDRsPnEvKvLmsxHt8LQVBkEBm2PBJFY+OXf1MNW021viDBpR10mX4WQ6zrsGL5L0GY4cwf4tlbh+Obit+LnN/SQTnREf8fPpdKZ1sa/ui3pGi8lMT6io4D7Ujlwx2RdCkBF+isfMf77HCEGsZANw0hSrO2FGg14Sl26xLAIohdaW8O7gEaag8JdVAZ3OVLd5Df1NkZBEr753Xb8WwaXsJjE7qxwINL1KdqA4+EiYW4edb7+a9bbBeOPtb67ZxmFqgyTNS/4obxahezNkjk00ytswsENg//Ee6dWBJZyLH+QGsaU2jO/W4WvRyZhmKKPdipOhiz4Rlrd2XYgsfHsfWf5v4GOTL+13ZB24dW1/m39n2woJ+v686fXbNW85XP/r",
			"lvQLU/KqgFhsLkt/5C/scqs7nWR+eYtyPdWiLVBux9GblT4AhHYMdCgwQfSJcudvsgV6fXoK+DUSRgJ++Nqt+Wvb7GlYlHpxCysQhz26TTu8Nyo7zpmVPH92+UYmbvbQCSvX2BhWtvkfHmqDVjmSIQ4RUMfeveA1KZbSf999NE4qKK8Do+8oXcmTM4LZVmh1rlyqznIdFXPN7x3pD4E0gb6/y69xtWMChv9654FMg05bAdueKt9uA4BEcAbpkdHF",
			"cmzVCcRVnckw3QUPhmG4Bkppeg4K50oDQwQ9EH+Fq1s=", false},
		{"hwk883gUlTKCyXYA6XWZa8H9/xKIYZaJ0xEs0M5hQOMxiGpxocuX/8maSDmeCk3bo5ViaDBdO7ZBxAhLSe5k/5TFQyF5Lv7KN2tLKnwgoWMqB16OL8WdbePIwTCuPtJNAFKoTZylLDbSf02kckMcZQDPF9iGh+JC99Pio74vDpwTEjUx5tQ99gNQwxULtztsqDRsPnEvKvLmsxHt8LQVBkEBm2PBJFY+OXf1MNW021viDBpR10mX4WQ6zrsGL5L0GY4cwf4tlbh+Obit+LnN/SQTnREf8fPpdKZ1sa/ui3pGi8lMT6io4D7Ujlwx2RdCkBF+isfMf77HCEGsZANw0hSrO2FGg14Sl26xLAIohdaW8O7gEaag8JdVAZ3OVLd5Df1NkZBEr753Xb8WwaXsJjE7qxwINL1KdqA4+EiYW4edb7+a9bbBeOPtb67ZxmFqgyTNS/4obxahezNkjk00ytswsENg//Ee6dWBJZyLH+QGsaU2jO/W4WvRyZhmKKPdipOhiz4Rlrd2XYgsfHsfWf5v4GOTL+13ZB24dW1/m39n2woJ+v686fXbNW85XP/r",
			"lvQLU/KqgFhsLkt/5C/scqs7nWR+eYtyPdWiLVBux9GblT4AhHYMdCgwQfSJcudvsgV6fXoK+DUSRgJ++Nqt+Wvb7GlYlHpxCysQhz26TTu8Nyo7zpmVPH92+UYmbvbQCSvX2BhWtvkfHmqDVjmSIQ4RUMfeveA1KZbSf999NE4qKK8Do+8oXcmTM4LZVmh1rlyqznIdFXPN7x3pD4E0gb6/y69xtWMChv9654FMg05bAdueKt9uA4BEcAbpkdHF",
			"cmzVCcRVnckw3QUPhmG4Bkppeg4K50oDQwQ9EH+Fq1s=", false},
	} {
		vk, err := base64.StdEncoding.DecodeString(test.vk)
		require.NoError(t, err)
		proof, err := base64.StdEncoding.DecodeString(test.proof)
		require.NoError(t, err)
		inputs, err := base64.StdEncoding.DecodeString(test.inputs)
		require.NoError(t, err)
		ok, err := Groth16Verify(vk, proof, inputs)
		if test.ok {
			require.NoError(t, err)
			assert.True(t, ok)
		} else {
			assert.False(t, ok)
			assert.Error(t, err)
		}
	}
}
