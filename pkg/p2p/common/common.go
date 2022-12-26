package common

type DuplicateChecker interface {
	Add(peerID string, message []byte) (isNew bool)
}
