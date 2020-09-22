package fride

import (
	"encoding/base64"

	"github.com/pkg/errors"
)

func EstimateTree(tree *Tree, v int) (int, int, map[string]int, error) {
	switch v {
	case 1:
		te, err := newTreeEstimatorV1(tree)
		if err != nil {
			return 0, 0, nil, errors.Wrap(err, "failed to estimate with tree estimator V1")
		}
		max, verifier, functions, err := te.estimate()
		if err != nil {
			return 0, 0, nil, errors.Wrap(err, "failed to estimate with tree estimator V1")
		}
		return max, verifier, functions, nil
	case 2:
		id := base64.StdEncoding.EncodeToString(tree.Digest[:])
		switch id {
		case "WX51srRAh4WN7dzlOu5EDXYm1vbR9x+gzvd8r2NeXKI=":
			return 3418, 0, map[string]int{"random": 3418}, nil
		case "iCUS2gce1wBLSNEE//ehvTvBaoZ6aLtdLxyj1k76yjk=":
			return 3223, 0, map[string]int{"random": 3223}, nil
		case "x932L0eSYHVgTMIYKVVDKXm67RFFUrshgsKYYjFVLfg=":
			return 3089, 0, map[string]int{"random": 3089}, nil
		case "VPRjsCLrvwS8pKPk78vzZ2VplwlKWBbRs4M/KhZGpCU=":
			return 3128, 0, map[string]int{"setOrder": 3128, "cancelOrder": 1143, "executeOrder": 2794}, nil
		case "fjng1PLxQ1nXYlubqOuS8FLxzIaY92AheqsNfWuEmgE=":
			return 3128, 0, map[string]int{"setOrder": 3128, "cancelOrder": 1143, "executeOrder": 2791}, nil
		case "hBqgd3xz9N3dwfVBMQ5uba3l9Bo/+grSpZsCCyuefJs=":
			return 3118, 0, map[string]int{"setOrder": 3118, "cancelOrder": 1133, "executeOrder": 1887}, nil
		case "jLb7Zt9mH8cjIVhnyVYVXkATgHrNzqdEMF/PGG/tpV8=":
			return 3128, 0, map[string]int{"setOrder": 3128, "cancelOrder": 1143, "executeOrder": 2794}, nil
		case "oUYWZD6AgjbrJIlq9X89KkkbGn6yWRlao4+JcPVyMvM=":
			return 3118, 0, map[string]int{"setOrder": 3118, "cancelOrder": 1133, "executeOrder": 1875}, nil
		}

		te, err := newTreeEstimatorV2(tree)
		if err != nil {
			return 0, 0, nil, errors.Wrap(err, "failed to estimate with tree estimator V2")
		}
		max, verifier, functions, err := te.estimate()
		if err != nil {
			return 0, 0, nil, errors.Wrap(err, "failed to estimate with tree estimator V2")
		}
		return max, verifier, functions, nil
	case 3:
		te, err := newTreeEstimatorV3(tree)
		if err != nil {
			return 0, 0, nil, errors.Wrap(err, "failed to estimate with tree estimator V3")
		}
		max, verifier, functions, err := te.estimate()
		if err != nil {
			return 0, 0, nil, errors.Wrap(err, "failed to estimate with tree estimator V3")
		}
		return max, verifier, functions, nil
	default:
		return 0, 0, nil, errors.Errorf("unsupported version of tree estimator '%d'", v)
	}
}
