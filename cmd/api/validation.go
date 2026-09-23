package main

import (
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
)

func (app *application) fieldErrors(err error) map[string]string {
	var vErrors validator.ValidationErrors

	if !errors.As(err, &vErrors) {
		return nil
	}

	result := make(map[string]string)

	for _, err := range vErrors {
		if _, ok := result[err.Field()]; !ok {
			result[err.Field()] = app.validationMessage(err.Tag(), err.Param())
		}
	}

	return result
}

func (app *application) validationMessage(tag, param string) string {
	switch tag {
	case "required":
		return "must be provided"
	case "max":
		return fmt.Sprintf("must not be more than %s characters", param)
	case "min":
		return fmt.Sprintf("must be at least %s characters", param)
	case "startswith":
		return fmt.Sprintf("must start with %s", param)
	default:
		return "is invalid"
	}
}
