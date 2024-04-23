package consensus

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/settings"
)

type timeMock struct{}

func (t timeMock) Now() time.Time { return time.Now().UTC() }

func TestValidator_ShouldIncludeNewBlockFieldsOfLightNodeFeature(t *testing.T) {
	tests := []struct {
		lightNodeIsActiveAtHeight           bool
		lightNodeActivationHeight           uint64
		lightNodeBlockFieldsAbsenceInterval uint64
		blockHeight                         uint64
		expectedResult                      bool
	}{
		{
			lightNodeIsActiveAtHeight:           false,
			lightNodeActivationHeight:           0,
			lightNodeBlockFieldsAbsenceInterval: 0,
			blockHeight:                         1,
			expectedResult:                      false,
		},
		{
			lightNodeIsActiveAtHeight:           true,
			lightNodeActivationHeight:           1,
			lightNodeBlockFieldsAbsenceInterval: 1,
			blockHeight:                         2,
			expectedResult:                      true,
		},
		{
			lightNodeIsActiveAtHeight:           true,
			lightNodeActivationHeight:           10,
			lightNodeBlockFieldsAbsenceInterval: 5,
			blockHeight:                         14,
			expectedResult:                      false,
		},
		{
			lightNodeIsActiveAtHeight:           true,
			lightNodeActivationHeight:           10,
			lightNodeBlockFieldsAbsenceInterval: 5,
			blockHeight:                         16,
			expectedResult:                      true,
		},
	}
	for i, tt := range tests {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			sip := &stateInfoProviderMock{
				NewestIsActiveAtHeightFunc: func(featureID int16, height uint64) (bool, error) {
					require.Equal(t, int16(settings.LightNode), featureID)
					require.Equal(t, tt.blockHeight, height)
					return tt.lightNodeIsActiveAtHeight, nil
				},
				NewestActivationHeightFunc: func(featureID int16) (uint64, error) {
					require.Equal(t, int16(settings.LightNode), featureID)
					return tt.lightNodeActivationHeight, nil
				},
			}
			sets := *settings.TestNetSettings // copy of testnet settings
			sets.LightNodeBlockFieldsAbsenceInterval = tt.lightNodeBlockFieldsAbsenceInterval
			v := NewValidator(sip, &sets, timeMock{})
			result, err := v.ShouldIncludeNewBlockFieldsOfLightNodeFeature(tt.blockHeight)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}
