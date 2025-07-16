package utils

type NonRetryableError struct {
	err error
}

func DoNotRetry(err error) error {
	return &NonRetryableError{err: err}
}

func (err *NonRetryableError) Error() string {
	return err.err.Error()
}
