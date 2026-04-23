package service

type ValidationError struct {
	Message string
}

func (e ValidationError) Error() string { return e.Message }

func domainErr(msg string) error {
	return ValidationError{Message: msg}
}
