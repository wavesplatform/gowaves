package errs

type ValidationError interface {
	ValidationError()
}

type ValidationErrorImpl struct {
}

func (ValidationErrorImpl) ValidationError() {
}

func IsValidationError(err error) bool {
	_, ok := err.(ValidationError)
	return ok
}

type BlockValidationError struct {
	ValidationErrorImpl
	message string
}

func (a BlockValidationError) Error() string {
	return a.message
}

func NewBlockValidationError(message string) *BlockValidationError {
	return &BlockValidationError{
		message: message,
	}
}
