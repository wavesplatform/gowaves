package proto

//type OptionalError interface {
//	Err() error
//	String() string
//	IsNil() bool
//}

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

// This struct describes "error" level of message, which users should know about
type ErrorMsg struct {
	err error
}

func NewErrorMsg(err error) error {
	return &ErrorMsg{err: err}
}

func (em *ErrorMsg) Error() string {
	return em.err.Error()
}

func (em *ErrorMsg) IsNil() bool {
	return em.err == nil
}
