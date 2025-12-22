package responder

// Response represents an HTTP response with status, body, message, and error.
// It can be used to encapsulate both successful and error responses.
type Response interface {
	// Status returns the HTTP status code of the response.
	Status() int
}

// SuccessResponse represents a successful HTTP response with status, body.
type SuccessResponse struct {
	// status represents the HTTP status code of the response.
	status int
	// body contains the response payload to be sent to the client.
	body any
}

// Status returns the HTTP status code of the successful response.
func (r SuccessResponse) Status() int {
	return r.status
}

// ErrorResponse represents an HTTP response with status, message, and error.
type ErrorResponse struct {
	// status represents the HTTP status code of the response.
	status int
	// message is a human-readable message associated with an error.
	message string
	// err holds the internal error for logging purposes.
	err error
}

// Status returns the HTTP status code of the error response.
func (r ErrorResponse) Status() int {
	return r.status
}

// Error returns the internal error associated with the error response.
func (r ErrorResponse) Error() string {
	return r.err.Error()
}

// Error creates a new error Response with the given status code, message, and error.
// The message is intended to be sent to the client, while the error is for internal logging.
func Error(status int, err error, message string) Response {
	return ErrorResponse{
		status:  status,
		err:     err,
		message: message,
	}
}

// Success creates a new successful Response with the given status code and body.
func Success(status int, body any) Response {
	return SuccessResponse{
		status: status,
		body:   body,
	}
}
