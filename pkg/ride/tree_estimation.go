package ride

import (
	"encoding/base64"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
)

type TreeEstimation struct {
	Estimation int            `cbor:"0,keyasint"`
	Verifier   int            `cbor:"1,keyasint,omitempty"`
	Functions  map[string]int `cbor:"2,keyasint,omitempty"`
}

func EstimateTree(tree *ast.Tree, v int) (TreeEstimation, error) {
	switch v {
	case 1:
		te, err := newTreeEstimatorV1(tree)
		if err != nil {
			return TreeEstimation{}, errors.Wrapf(err, "failed to estimate with tree estimator V%d", v)
		}
		max, verifier, functions, err := te.estimate()
		if err != nil {
			return TreeEstimation{}, errors.Wrapf(err, "failed to estimate with tree estimator V%d", v)
		}
		return TreeEstimation{Estimation: max, Verifier: verifier, Functions: functions}, nil
	case 2:
		id := base64.StdEncoding.EncodeToString(tree.Digest[:])
		switch id {
		case "WX51srRAh4WN7dzlOu5EDXYm1vbR9x+gzvd8r2NeXKI=":
			return TreeEstimation{Estimation: 3418, Functions: map[string]int{"random": 3418}}, nil
		case "iCUS2gce1wBLSNEE//ehvTvBaoZ6aLtdLxyj1k76yjk=":
			return TreeEstimation{Estimation: 3223, Functions: map[string]int{"random": 3223}}, nil
		case "x932L0eSYHVgTMIYKVVDKXm67RFFUrshgsKYYjFVLfg=":
			return TreeEstimation{Estimation: 3089, Functions: map[string]int{"random": 3089}}, nil
		case "VPRjsCLrvwS8pKPk78vzZ2VplwlKWBbRs4M/KhZGpCU=":
			return TreeEstimation{Estimation: 3128, Functions: map[string]int{"setOrder": 3128, "cancelOrder": 1143, "executeOrder": 2794}}, nil
		case "fjng1PLxQ1nXYlubqOuS8FLxzIaY92AheqsNfWuEmgE=":
			return TreeEstimation{Estimation: 3128, Functions: map[string]int{"setOrder": 3128, "cancelOrder": 1143, "executeOrder": 2791}}, nil
		case "hBqgd3xz9N3dwfVBMQ5uba3l9Bo/+grSpZsCCyuefJs=":
			return TreeEstimation{Estimation: 3118, Functions: map[string]int{"setOrder": 3118, "cancelOrder": 1133, "executeOrder": 1887}}, nil
		case "jLb7Zt9mH8cjIVhnyVYVXkATgHrNzqdEMF/PGG/tpV8=":
			return TreeEstimation{Estimation: 3128, Functions: map[string]int{"setOrder": 3128, "cancelOrder": 1143, "executeOrder": 2794}}, nil
		case "oUYWZD6AgjbrJIlq9X89KkkbGn6yWRlao4+JcPVyMvM=":
			return TreeEstimation{Estimation: 3118, Functions: map[string]int{"setOrder": 3118, "cancelOrder": 1133, "executeOrder": 1875}}, nil
		}

		te, err := newTreeEstimatorV2(tree)
		if err != nil {
			return TreeEstimation{}, errors.Wrapf(err, "failed to estimate with tree estimator V%d", v)
		}
		max, verifier, functions, err := te.estimate()
		if err != nil {
			return TreeEstimation{}, errors.Wrapf(err, "failed to estimate with tree estimator V%d", v)
		}
		return TreeEstimation{Estimation: max, Verifier: verifier, Functions: functions}, nil
	case 3:
		te, err := newTreeEstimatorV3(tree)
		if err != nil {
			return TreeEstimation{}, errors.Wrapf(err, "failed to estimate with tree estimator V%d", v)
		}
		max, verifier, functions, err := te.estimate()
		if err != nil {
			return TreeEstimation{}, errors.Wrapf(err, "failed to estimate with tree estimator V%d", v)
		}
		return TreeEstimation{Estimation: max, Verifier: verifier, Functions: functions}, nil
	case 4:
		te, err := newTreeEstimatorV4(tree)
		if err != nil {
			return TreeEstimation{}, errors.Wrapf(err, "failed to estimate with tree estimator V%d", v)
		}
		max, verifier, functions, err := te.estimate()
		if err != nil {
			return TreeEstimation{}, errors.Wrapf(err, "failed to estimate with tree estimator V%d", v)
		}
		return TreeEstimation{Estimation: max, Verifier: verifier, Functions: functions}, nil
	default:
		return TreeEstimation{}, errors.Errorf("unsupported version of tree estimator '%d'", v)
	}
}
