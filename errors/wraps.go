package errors

import "net/http"

var (
	ErrorRecordNotFound = Error{Key: "ERROR.RECORD_NOT_FOUND", Status: 404}

	ErrorUnprocessable = Error{Key: "ERROR.UNPROCESSABLE", Status: 422}

	ErrorInvalidFields = Error{Key: "ERROR.INVALID_FIELDS", Status: http.StatusNotAcceptable}
)

func WrapInvalidFields(err error) error {
	return Wrap(ErrorInvalidFields, err)
}

func WrapNotFound(err error) error {
	return WrapWithStatus(ErrorRecordNotFound, err, http.StatusNotFound)
}

func WrapForbidden(err error) error {
	return Wrap(ErrorForbidden, err)
}

func WrapUnprocessable(err error) error {
	return WrapWithStatus(ErrorUnprocessable, err, http.StatusUnprocessableEntity)
}

func WrapGeneric(err error) error {
	return WrapWithStatus(ErrorGeneric, err, 500)
}

// Unwrap attempts to unwind the error all the way back
func Unwrap(err error) Error {
	if ae, ok := err.(Error); ok {
		if e, ok := ae.Err.(Error); ok {
			return Unwrap(e)
		}
		return ae
	}
	return ErrorGeneric.NewError(err)
}
