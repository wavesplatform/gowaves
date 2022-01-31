package miner

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

type Features []settings.Feature

func FeaturesToInt16(a []settings.Feature) []int16 {
	var out []int16
	for _, v := range a {
		out = append(out, int16(v))
	}
	return out
}

func ParseVoteFeatures(s string) (Features, error) {
	if s == "" {
		return Features{}, nil
	}
	split := strings.Split(s, ",")
	var out Features
	for _, val := range split {
		f, err := parseFeature(val)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, nil
}

func parseFeature(s string) (settings.Feature, error) {
	u, err := strconv.ParseInt(s, 10, 16)
	if err != nil {
		return 0, err
	}
	return settings.Feature(u), nil
}

func ParseReward(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.ParseInt(s, 10, 64)
}

type featureState interface {
	IsActivated(featureID int16) (bool, error)
	IsApproved(featureID int16) (bool, error)
}

func ValidateFeatures(state featureState, features Features) (Features, error) {
	out := Features{}
	for _, feature := range features {
		info, ok := settings.FeaturesInfo[feature]
		if !ok {
			return nil, errors.Errorf("unknown feature %d", feature)
		}
		if !info.Implemented {
			return nil, errors.Errorf("feature '%s'(%d) not implemented ", info.Description, feature)
		}

		activated, err := state.IsActivated(int16(feature))
		if err != nil {
			return nil, err
		}
		if activated {
			continue
		}

		approved, err := state.IsApproved(int16(feature))
		if err != nil {
			return nil, err
		}
		if approved {
			// feature approved, no need for voting
			continue
		}

		out = append(out, feature)
	}
	return out, nil
}
