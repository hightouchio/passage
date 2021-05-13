package tunnel

import "fmt"

type requestErrors struct {
	errors []error
}

func newRequestErrors() *requestErrors {
	return &requestErrors{make([]error, 0)}
}

func (e *requestErrors) addError(m string, args ...interface{}) {
	e.errors = append(e.errors, fmt.Errorf(m, args...))
}

func (e *requestErrors) IsEmpty() bool {
	return len(e.errors) == 0
}

func (e *requestErrors) Error() string {
	if e.IsEmpty() {
		return ""
	}

	return e.errors[0].Error()
}
