// Package responder provides a flexible and configurable way to send HTTP responses
// with different content types and status codes. It supports JSON, text, HTML, CSV, and XML
// responses, and allows customization of error message formatting and content formatting.
// It may be useful when writing web servers without a full-fledged web framework
// and avoid boilerplate code.
package responder

import (
	"encoding"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/mickaelvieira/responder/internal"
)

type responseWriter http.ResponseWriter

const (
	// TextContentType is the content type for plain text responses
	TextContentType = "text/plain; charset=utf-8"
	// CSVContentType is the content type for CSV responses
	CSVContentType = "text/csv; charset=utf-8"
	// HTMLContentType is the content type for HTML responses
	HTMLContentType = "text/html; charset=utf-8"
	// JSONContentType is the content type for JSON responses
	JSONContentType = "application/json; charset=utf-8"
	// XMLContentType is the content type for XML responses
	XMLContentType = "application/xml; charset=utf-8"
)

const (
	status200 = http.StatusOK
	status201 = http.StatusCreated
	status202 = http.StatusAccepted
	status204 = http.StatusNoContent
	status301 = http.StatusMovedPermanently
	status302 = http.StatusFound
	status303 = http.StatusSeeOther
	status307 = http.StatusTemporaryRedirect
	status400 = http.StatusBadRequest
	status401 = http.StatusUnauthorized
	status403 = http.StatusForbidden
	status404 = http.StatusNotFound
	status500 = http.StatusInternalServerError
)

//nolint:revive // revive complains about the cognitive-complexity but to be fair, it is not that hard to read.
func defaultDataFormatter(c any) []byte {
	if c == nil {
		return []byte{}
	}

	switch v := c.(type) {
	case string:
		return []byte(v)
	case []byte:
		return v
	case xml.Marshaler:
		// Create a simple encoder to marshal XML
		b, err := xml.Marshal(v)
		if err != nil {
			return fmt.Appendf(nil, "received invalid content - %s", err)
		}

		return b
	case json.Marshaler:
		b, err := v.MarshalJSON()
		if err != nil {
			return fmt.Appendf(nil, "received invalid content - %s", err)
		}

		return b
	case encoding.TextMarshaler:
		b, err := v.MarshalText()
		if err != nil {
			return fmt.Appendf(nil, "received invalid content - %s", err)
		}

		return b
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Appendf(nil, "received invalid content - %s", err)
		}

		return b
	}
}

// ErrorFormatter defines a function type for formatting error messages
// before sending them in the response.
// It receives the original error message as any type and returns
// the formatted message as an any type.
// The output of this function is passed to the DataFormatter.
// The default error formatter converts the message to a string.
type ErrorFormatter func(any) any

// DataFormatter defines a function type for formatting
// the data before sending it in the response.
// It receives the original data as an any type and returns
// the formatted data as a []byte.
type DataFormatter func(any) []byte

// stringFormatter is the default error formatter that converts
// the error message to a string.
var stringFormatter = func(message any) any {
	return internal.MessageToString(message)
}

// OptionsModifier defines a function type for modifying Responder options.
type OptionsModifier func(*options)

// WithLogger sets a logger for the responder
func WithLogger(l *slog.Logger) OptionsModifier {
	return func(o *options) {
		o.logger = l
	}
}

// WithDataFormatter sets a custom data formatter
func WithDataFormatter(f DataFormatter) OptionsModifier {
	return func(o *options) {
		o.dataFormatter = f
	}
}

// WithErrorFormatter sets a custom error message formatter
func WithErrorFormatter(f ErrorFormatter) OptionsModifier {
	return func(o *options) {
		o.errorFormatter = f
	}
}

// options holds the configuration options for the Responder.
type options struct {
	logger         *slog.Logger
	dataFormatter  DataFormatter
	errorFormatter ErrorFormatter
}

// Responder defines the interface for sending HTTP responses.
type Responder interface {
	// Send200 sends a 200 OK response.
	// It takes as second argument the data to be sent to the client.
	Send200(responseWriter, any)

	// Send201 sends a 201 Created response.
	// It takes as second argument the data to be sent to the client.
	Send201(responseWriter, any)

	// Send202 sends a 202 Accepted response.
	// It takes as second argument the data to be sent to the client.
	Send202(responseWriter, any)

	// Send204 sends a 204 No Content response.
	Send204(responseWriter)

	// Redirect301 sends a 301 Moved Permanently response to the given URL.
	Redirect301(responseWriter, *http.Request, string)

	// Redirect302 sends a 302 Found response to the given URL.
	Redirect302(responseWriter, *http.Request, string)

	// Redirect303 sends a 303 See Other response to the given URL.
	Redirect303(responseWriter, *http.Request, string)

	// Redirect307 sends a 307 Temporary Redirect response to the given URL.
	Redirect307(responseWriter, *http.Request, string)

	// Send400 sends a 400 Bad Request response. It takes as second argument
	// the error that caused the bad request, and as third argument a message
	// to be sent to the client.
	// The error will be logged if a logger was provided.
	Send400(responseWriter, error, any)

	// Send401 sends a 401 Unauthorized response. It takes as second argument
	// the error that caused the unauthorized response, and as third argument
	// a message to be sent to the client.
	// The error will be logged if a logger was provided.
	Send401(responseWriter, error, any)

	// Send403 sends a 403 Forbidden response. It takes as second argument
	// the error that caused the forbidden response, and as third argument
	// a message to be sent to the client.
	// The error will be logged if a logger was provided.
	Send403(responseWriter, error, any)

	// Send404 sends a 404 Not Found response. It takes as second argument
	// the error that caused the not found response, and as third argument
	// a message to be sent to the client.
	// The error will be logged if a logger was provided.
	Send404(responseWriter, error, any)

	// Send500 sends a 500 Internal Server Error response.
	// It takes as second argument the error that caused the
	// internal server error, and as third argument
	// a message to be sent to the client.
	// The error will be logged if a logger was provided.
	Send500(responseWriter, error, any)

	// Send sends a response with the given status code and body.
	Send(responseWriter, Response)
}

// New creates a new Responder with the given content type and options.
func New(contentType string, optionsModifiers ...OptionsModifier) Responder {
	o := &options{
		errorFormatter: stringFormatter,
		dataFormatter:  defaultDataFormatter,
	}

	for _, modify := range optionsModifiers {
		modify(o)
	}

	return &responder{
		contentType: contentType,
		options:     o,
	}
}

type responder struct {
	contentType string
	options     *options
}

func (r responder) send(rw responseWriter, code int, body []byte) {
	rw.Header().Set("Content-Type", r.contentType)
	rw.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
	rw.WriteHeader(code)

	_, err := rw.Write(body)
	if err != nil && r.options.logger != nil {
		r.options.logger.Error("failed to write response",
			"status", code,
			"error", err,
		)
	}
}

func (r *responder) logError(err error, code int, message any) {
	if err == nil || r.options.logger == nil {
		return
	}

	r.options.logger.Error(internal.MessageToString(message),
		"status", code,
		"error", err,
	)
}

func (r *responder) Send(rw responseWriter, resp Response) {
	switch v := resp.(type) {
	case ErrorResponse:
		r.logError(v.err, v.status, v.message)
		r.send(rw, resp.Status(), r.options.dataFormatter(
			r.options.errorFormatter(v.message),
		))
	case SuccessResponse:
		r.send(rw, resp.Status(), r.options.dataFormatter(
			v.body,
		))
	default:
		r.logError(fmt.Errorf("unknown response type %T", resp),
			resp.Status(),
			"failed to send response",
		)
	}
}

func (r *responder) Send200(rw responseWriter, data any) {
	r.send(rw, status200, r.options.dataFormatter(data))
}

func (r *responder) Send201(rw responseWriter, data any) {
	r.send(rw, status201, r.options.dataFormatter(data))
}

func (r *responder) Send202(rw responseWriter, data any) {
	r.send(rw, status202, r.options.dataFormatter(data))
}

func (r *responder) Send204(rw responseWriter) {
	r.send(rw, status204, r.options.dataFormatter(nil))
}

func (responder) Redirect301(rw responseWriter, req *http.Request, loc string) {
	http.Redirect(rw, req, loc, status301)
}

func (responder) Redirect302(rw responseWriter, req *http.Request, loc string) {
	http.Redirect(rw, req, loc, status302)
}

func (responder) Redirect303(rw responseWriter, req *http.Request, loc string) {
	http.Redirect(rw, req, loc, status303)
}

func (responder) Redirect307(rw responseWriter, req *http.Request, loc string) {
	http.Redirect(rw, req, loc, status307)
}

func (r *responder) Send400(rw responseWriter, err error, message any) {
	r.logError(err, status400, message)
	r.send(rw, status400, r.options.dataFormatter(
		r.options.errorFormatter(message)),
	)
}

func (r *responder) Send401(rw responseWriter, err error, message any) {
	r.logError(err, status401, message)
	r.send(rw, status401, r.options.dataFormatter(
		r.options.errorFormatter(message)),
	)
}

func (r *responder) Send403(rw responseWriter, err error, message any) {
	r.logError(err, status403, message)
	r.send(rw, status403, r.options.dataFormatter(
		r.options.errorFormatter(message)),
	)
}

func (r *responder) Send404(rw responseWriter, err error, message any) {
	r.logError(err, status404, message)
	r.send(rw, status404, r.options.dataFormatter(
		r.options.errorFormatter(message)),
	)
}

func (r *responder) Send500(rw responseWriter, err error, message any) {
	r.logError(err, status500, message)
	r.send(rw, status500, r.options.dataFormatter(
		r.options.errorFormatter(message)),
	)
}
