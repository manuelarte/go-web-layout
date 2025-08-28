package rest

import (
	"fmt"
	"net/http"

	"github.com/manuelarte/ptrutils"
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
		Errors:   ptrutils.Ptr(errors),
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
