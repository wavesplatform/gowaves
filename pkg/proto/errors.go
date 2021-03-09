package proto

// This struct describes "info" level of message, which users shouldn't even know about

type InfoMsg struct {
	err error
}

func NewInfoMsg(err error) error {
	return &InfoMsg{err: err}
}

func (im *InfoMsg) Error() string {
	return im.err.Error()
}

func (im *InfoMsg) IsNil() bool {
	return im.err == nil
}
