package util

import (
	"strconv"
	"strings"

	"github.com/wavesplatform/gowaves/pkg/settings"
)

type Features []settings.Feature

func (a Features) Features() []int16 {
	var out []int16
	for _, v := range a {
		out = append(out, int16(v))
	}
	return out
}

func ParseVoteFeatures(s string) (Features, error) {
	splitted := strings.Split(s, ",")
	var out Features
	for _, val := range splitted {
		f, err := parseFeature(val)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, nil
}

func parseFeature(s string) (settings.Feature, error) {
	u, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return settings.Feature(u), nil
}

func ParseReward(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}
