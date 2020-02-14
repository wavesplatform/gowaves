package api

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type roll struct {
	apiKey string
	height proto.Height
}

func (a *roll) RollbackToHeight(apiKey string, height proto.Height) error {
	a.apiKey = apiKey
	a.height = height
	return nil
}

func TestRollbackToHeight(t *testing.T) {
	r := &roll{}
	f := RollbackToHeight(r)
	req := httptest.NewRequest("POST", "/blocks/rollback", strings.NewReader(`{"height": 100500}`))
	req.Header.Add(API_KEY, "apikey")
	resp := httptest.NewRecorder()
	f(resp, req)

	assert.Equal(t, "apikey", r.apiKey)
	assert.EqualValues(t, 100500, r.height)
}
