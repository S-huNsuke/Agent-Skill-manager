package errors

import "fmt"

type AppError struct {
	Operation string `json:"operation"`
	Message   string `json:"message"`
}

func Wrap(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}

func Present(err error) any {
	if err == nil {
		return nil
	}
	return AppError{
		Operation: "desktop-runtime",
		Message:   err.Error(),
	}
}
