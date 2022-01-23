package errors

import "net/http"

var (
	// ErrorGeneric generic error key no status
	ErrorGeneric = Error{Key: "ERROR.UNKNOWN"}

	// ErrorNotAuthorized - returns 401
	ErrorNotAuthorized = Error{Key: "ERROR.NOT_LOGGED_IN", Status: http.StatusUnauthorized}

	ErrorForbidden = Error{Key: "ERROR.FORBIDDEN", Status: http.StatusForbidden}
)
