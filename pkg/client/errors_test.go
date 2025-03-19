package client

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseError_Error(t *testing.T) {
	txt := "parse error"
	inner := errors.New(txt)
	err := newParseError(inner)
	assert.Equal(t, txt, err.Error())
	assert.ErrorIs(t, err, inner)
	assert.ErrorAs(t, err, new(*ParseError))
}

func TestRequestError_Error(t *testing.T) {
	txt := "request error"
	inner := errors.New(txt)
	err := newRequestError(inner, "")
	assert.Equal(t, txt, err.Error())
	assert.ErrorIs(t, err, inner)
	assert.ErrorAs(t, err, new(*RequestError))
}
