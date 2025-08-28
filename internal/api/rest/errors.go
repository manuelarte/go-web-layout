package rest

import (
	"net/http"
	"strconv"
)

var _ error = new(ValidationError)

type ValidationError struct {
	errors map[string][]error
}

func (v ValidationError) Error() string {
	msg := ""
	for key, errs := range v.errors {
		msg += key + ": "
		for _, err := range errs {
			return msg + err.Error()
		}
	}

	return msg
}

func (v ValidationError) ErrorResponse() ErrorResponse {
	details := make(map[string][]string)

	for key, errs := range v.errors {
		for _, err := range errs {
			details[key] = append(details[key], err.Error())
		}
	}

	return ErrorResponse{
		Code:    strconv.Itoa(http.StatusBadRequest),
		Details: details,
		Message: "validation error",
	}
}
