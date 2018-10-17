package client

import (
	"errors"
	"github.com/magiconair/properties/assert"
	"testing"
)

func TestParseError_Error(t *testing.T) {
	txt := "parse error"
	err := ParseError{errors.New(txt)}
	assert.Equal(t, txt, err.Error())
}

func TestRequestError_Error(t *testing.T) {
	txt := "request error"
	err := RequestError{Err: errors.New(txt)}
	assert.Equal(t, txt, err.Error())
}
