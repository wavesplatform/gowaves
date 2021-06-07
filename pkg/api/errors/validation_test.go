package errors

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestStateCheckFailedError_MarshalJSON(t *testing.T) {
	value := StateCheckFailedError{
		validationErrorWithTransaction: validationErrorWithTransaction{
			validationError: validationError{
				genericError: genericError{
					ID:       StateCheckFailedErrorID,
					HttpCode: http.StatusBadRequest,
					Message:  "some message",
				},
			},
			Transaction: nil,
		},
		embeddedFields: map[string]interface{}{
			"extra_field": "value",
			"extra_int":   1,
		},
	}

	marshaled, err := value.MarshalJSON()
	assert.NoError(t, err)

	var unmarshaled map[string]interface{}
	err = json.Unmarshal(marshaled, &unmarshaled)
	assert.NoError(t, err)

	assert.Equal(t, float64(StateCheckFailedErrorID), unmarshaled["error"].(float64))
	assert.Equal(t, "some message", unmarshaled["message"])
	assert.Equal(t, "value", unmarshaled["extra_field"])
	assert.Equal(t, float64(1), unmarshaled["extra_int"].(float64))
}
