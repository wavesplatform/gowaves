package main

import "github.com/pkg/errors"

type Directives struct {
	lib         int64
	scriptType  int64
	contentType int64
}

func newDirectives(lib int64, scriptType int64, contentType int64) (Directives, error) {
	return Directives{lib, scriptType, contentType}, nil
}

func newStdLibVersion(ver int64) (int64, error) {
	if 1 > ver || ver > 6 {
		return 0, errors.Errorf("not correct STDLIB_VERSION: %d", ver)
	}
	return ver, nil
}

func newScriptType(scriptType string) (int64, error) {
	switch scriptType {
	case "ASSET":
		return 1, nil
	case "ACCOUNT":
		return 2, nil
	default:
		return 0, errors.Errorf("undefined SCRIPT_TYPE: %s", scriptType)
	}
}

func newContentType(contentType string) (int64, error) {
	switch contentType {
	case "DAPP":
		return 2, nil
	case "EXPRESSION":
		return 1, nil
	default:
		return 0, errors.Errorf("undefined CONTENT_TYPE: %s", contentType)
	}
}
