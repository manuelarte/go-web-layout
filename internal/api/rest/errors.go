package rest

import (
	"fmt"
	"net/http"
	"strings"
)

var _ error = new(ValidationError)

type ValidationError struct {
	errors map[string][]error
}

func (v ValidationError) Error() string {
	var msg strings.Builder
	for key, errs := range v.errors {
		msg.WriteString(key + ": ")

		for _, err := range errs {
			return msg.String() + err.Error()
		}
	}

	return msg.String()
}

func (v ValidationError) ErrorResponse(traceID string) ErrorResponse {
	errors := make([]Error, len(v.errors))

	for key, errs := range v.errors {
		for i, err := range errs {
			errors[i] = Error{
				Detail:  err.Error(),
				Pointer: key,
			}
		}
	}

	return ErrorResponse{
		Type:     "ValidationError",
		Title:    "Validation Error",
		Detail:   "Validation Error",
		Status:   http.StatusBadRequest,
		Errors:   new(errors),
		Instance: traceID,
	}
}

func (e *InvalidParamFormatError) ErrorResponse(traceID string) ErrorResponse {
	return ErrorResponse{
		Type:     "InvalidParameterValue",
		Title:    "Invalid Parameter Value",
		Detail:   fmt.Sprintf("%s: %s", e.ParamName, e.Err.Error()),
		Status:   http.StatusBadRequest,
		Instance: traceID,
	}
}
