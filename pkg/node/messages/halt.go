package messages

type HaltMessage struct {
}

func NewHaltMessage() *HaltMessage {
	return &HaltMessage{}
}

func (HaltMessage) Internal() {
}
