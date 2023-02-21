package crypto

import "fmt"

type IncorrectLengthError struct {
	Name        string
	Len         int
	ExpectedLen int
}

func (e IncorrectLengthError) Error() string {
	return fmt.Sprintf("incorrect %s length %d, expected %d", e.Name, e.Len, e.ExpectedLen)
}

func NewIncorrectLengthError(name string, len int, expectedLen int) IncorrectLengthError {
	return IncorrectLengthError{
		Name:        name,
		Len:         len,
		ExpectedLen: expectedLen,
	}
}
