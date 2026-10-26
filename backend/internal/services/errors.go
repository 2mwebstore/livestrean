// Package services holds the business logic. Services return *Error for
// expected failures; controllers map them to HTTP responses.
package services

type ErrorKind int

const (
	KindValidation ErrorKind = iota + 1
	KindUnauthorized
	KindForbidden
	KindNotFound
	KindConflict
)

type Error struct {
	Kind    ErrorKind
	Message string
	Fields  map[string]string
}

func (e *Error) Error() string { return e.Message }

func validationError(field, msg string) *Error {
	return &Error{Kind: KindValidation, Message: "Please check the highlighted fields", Fields: map[string]string{field: msg}}
}

func unauthorized(msg string) *Error { return &Error{Kind: KindUnauthorized, Message: msg} }
func forbidden(msg string) *Error    { return &Error{Kind: KindForbidden, Message: msg} }
func notFound(msg string) *Error     { return &Error{Kind: KindNotFound, Message: msg} }

func conflict(field, msg string) *Error {
	return &Error{Kind: KindConflict, Message: msg, Fields: map[string]string{field: msg}}
}

// RequestMeta identifies where a request came from, for sessions and audit logs.
type RequestMeta struct {
	IP        string
	UserAgent string
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
