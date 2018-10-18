package client

import "github.com/pkg/errors"

var NoApiKeyError = errors.New("no api key provided")

type RequestError struct {
	Err  error
	Body string
}

func (a *RequestError) Error() string {
	if a.Body != "" {
		return errors.Wrap(a.Err, a.Body).Error()
	}
	return a.Err.Error()
}

type ParseError struct {
	Err error
}

func (a ParseError) Error() string {
	return a.Err.Error()
}
