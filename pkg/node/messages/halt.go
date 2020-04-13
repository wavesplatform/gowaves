package messages

type HaltMessage struct {
	response chan struct{}
}

func NewHaltMessage(response chan struct{}) *HaltMessage {
	return &HaltMessage{
		response: response,
	}
}

func (HaltMessage) Internal() {
}

func (a *HaltMessage) Complete() {
	close(a.response)
}
