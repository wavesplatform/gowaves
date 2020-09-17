package fride

import "github.com/pkg/errors"

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
