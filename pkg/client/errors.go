package client

import "github.com/pkg/errors"

var NoApiKeyError = errors.New("no api key provided")

type RequestError struct {
	Err  error
	Body string
}

func newRequestError(err error, body string) *RequestError {
	return &RequestError{Err: err, Body: body}
}

func (e *RequestError) Unwrap() error {
	return e.Err
}

func (e *RequestError) Error() string {
	if e.Body != "" {
		return errors.Wrap(e.Err, e.Body).Error()
	}
	return e.Err.Error()
}

type ParseError struct {
	Err error
}

func newParseError(err error) *ParseError {
	return &ParseError{Err: err}
}

func (e ParseError) Unwrap() error {
	return e.Err
}

func (e ParseError) Error() string {
	return e.Err.Error()
}
