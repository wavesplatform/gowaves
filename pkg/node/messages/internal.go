package messages

func NewInternalChannel() chan InternalMessage {
	return make(chan InternalMessage, 100)
}

type InternalMessage interface {
	Internal()
}
