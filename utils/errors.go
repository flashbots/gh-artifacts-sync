package utils

type NonRetryableError struct {
	err error
}

func NoRetry(err error) error {
	return &NonRetryableError{err: err}
}

func (err *NonRetryableError) Error() string {
	return err.err.Error()
}
