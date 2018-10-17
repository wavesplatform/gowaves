package client

type RequestError struct {
	Err error
}

func (a *RequestError) Error() string {
	return a.Err.Error()
}

type ParseError struct {
	Err error
}

func (a ParseError) Error() string {
	return a.Err.Error()
}
