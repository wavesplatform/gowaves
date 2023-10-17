package errors

import (
	"fmt"
	"net/http"
)

type peersError struct {
	genericError
}

type (
	InvalidIPAddressError      peersError
	PeerConnectionFailureError peersError
)

var (
	InvalidIPAddress = &InvalidIPAddressError{
		genericError: genericError{
			ID:       InvalidIPAddressErrorID,
			HttpCode: http.StatusBadRequest,
			Message:  "Invalid IP address",
		},
	}
)

func NewPeerConnectionFailureError(inner error) *PeerConnectionFailureError {
	return &PeerConnectionFailureError{
		genericError: genericError{
			ID:       PeerConnectionFailureErrorID,
			HttpCode: http.StatusServiceUnavailable,
			Message:  fmt.Sprintf("Failed to connect to peer: %v", inner),
		},
	}
}
