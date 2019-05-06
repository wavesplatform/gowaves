package api

type BadRequestError struct {
	error
}

type AuthError struct {
	error
}

type InternalError struct {
	error
}
