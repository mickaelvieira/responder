package responder

import (
	"encoding"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

const GenericErrorMessage = "an error occurred"

const (
	TextContentType = "text/plain; charset=utf-8"
	CSVContentType  = "text/csv; charset=utf-8"
	HTMLContentType = "text/html; charset=utf-8"
	JSONContentType = "application/json; charset=utf-8"
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

func contentFormatter(c any) []byte {
	if c == nil {
		return []byte{}
	}

	switch v := c.(type) {
	case string:
		return []byte(v)
	case []byte:
		return v
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
		return fmt.Appendf(nil, "received invalid content - %s", fmt.Errorf("unknown type %T", c))
	}
}

// ErrorFormatter defines a function type for formatting error messages
// before sending them in the response.
// It receives the original error message as a string and returns
// the formatted message as an any type. The returned value
// should be a string, a []byte, or a struct that can be marshaled to JSON.
type ErrorFormatter func(string) any

// ContentFormatter defines a function type for formatting
// the content before sending it in the response.
// It receives the original content as an any type and returns
// the formatted content as a []byte.
type ContentFormatter func(any) []byte

var noopFormatter = func(message string) any {
	return message
}

// OptionsModifier defines a function type for modifying Responder options.
type OptionsModifier func(*Options)

// WithLogger sets a logger for the responder
func WithLogger(l *slog.Logger) OptionsModifier {
	return func(o *Options) {
		o.logger = l
	}
}

// WithContentFormatter sets a custom content formatter
func WithContentFormatter(f ContentFormatter) OptionsModifier {
	return func(o *Options) {
		o.contentFormatter = f
	}
}

// WithErrorFormatter sets a custom error message formatter
func WithErrorFormatter(f ErrorFormatter) OptionsModifier {
	return func(o *Options) {
		o.errorFormatter = f
	}
}

type Options struct {
	logger           *slog.Logger
	errorFormatter   ErrorFormatter
	contentFormatter ContentFormatter
}

// Responder defines the interface for sending HTTP responses.
type Responder interface {
	// Send200 sends a 200 OK response. It takes as second argument the data
	// to be sent to the client.
	Send200(http.ResponseWriter, any)

	// Send201 sends a 201 Created response. It takes as second argument the data
	// to be sent to the client.
	Send201(http.ResponseWriter, any)

	// Send202 sends a 202 Accepted response. It takes as second argument the data
	// to be sent to the client.
	Send202(http.ResponseWriter, any)

	// Send204 sends a 204 No Content response.
	Send204(http.ResponseWriter)

	// Redirect301 sends a 301 Moved Permanently response to the given URL.
	Redirect301(http.ResponseWriter, *http.Request, string)

	// Redirect302 sends a 302 Found response to the given URL.
	Redirect302(http.ResponseWriter, *http.Request, string)

	// Redirect303 sends a 303 See Other response to the given URL.
	Redirect303(http.ResponseWriter, *http.Request, string)

	// Redirect307 sends a 307 Temporary Redirect response to the given URL.
	Redirect307(http.ResponseWriter, *http.Request, string)

	// Send400 sends a 400 Bad Request response. It takes as second argument
	// the error that caused the bad request, and as third argument a message
	// to be sent to the client. The error will be logged if a logger was provided.
	Send400(http.ResponseWriter, error, string)

	// Send401 sends a 401 Unauthorized response. It takes as second argument
	// the error that caused the unauthorized response, and as third argument
	// a message to be sent to the client. The error will be logged if a logger was provided.
	Send401(http.ResponseWriter, error, string)

	// Send403 sends a 403 Forbidden response. It takes as second argument
	// the error that caused the forbidden response, and as third argument
	// a message to be sent to the client. The error will be logged if a logger was provided.
	Send403(http.ResponseWriter, error, string)

	// Send404 sends a 404 Not Found response. It takes as second argument
	// the error that caused the not found response, and as third argument
	// a message to be sent to the client. The error will be logged if a logger was provided.
	Send404(http.ResponseWriter, error, string)

	// Send500 sends a 500 Internal Server Error response. It takes as second argument
	// the error that caused the internal server error, and as third argument
	// a message to be sent to the client. The error will be logged if a logger was provided.
	Send500(http.ResponseWriter, error, string)
}

func New(contentType string, optionsModifiers ...OptionsModifier) Responder {
	options := &Options{
		errorFormatter:   noopFormatter,
		contentFormatter: contentFormatter,
	}

	for _, modify := range optionsModifiers {
		modify(options)
	}

	return &responder{
		contentType: contentType,
		options:     options,
	}
}

type responder struct {
	contentType string
	options     *Options
}

func (r responder) send(w http.ResponseWriter, code int, content []byte) {
	w.Header().Set("Content-Type", r.contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
	w.WriteHeader(code)

	_, err := w.Write(content)
	if err != nil && r.options.logger != nil {
		r.options.logger.Error("failed to write response", "status", code, "error", err)
	}
}

func (h *responder) logError(err error, code int, message any) {
	if err == nil || h.options.logger == nil {
		return
	}

	var m string
	switch v := message.(type) {
	case string:
		m = v
	case jsonError:
		m = v.Error
	default:
		m = GenericErrorMessage
	}

	h.options.logger.Error(m, "status", code, "error", err)
}

func (h *responder) Send200(w http.ResponseWriter, content any) {
	h.send(w, status200, h.options.contentFormatter(content))
}

func (h *responder) Send201(w http.ResponseWriter, content any) {
	h.send(w, status201, h.options.contentFormatter(content))
}

func (h *responder) Send202(w http.ResponseWriter, content any) {
	h.send(w, status202, h.options.contentFormatter(content))
}

func (h *responder) Send204(w http.ResponseWriter) {
	h.send(w, status204, h.options.contentFormatter(nil))
}

func (h *responder) Redirect301(w http.ResponseWriter, r *http.Request, location string) {
	http.Redirect(w, r, location, status301)
}

func (h *responder) Redirect302(w http.ResponseWriter, r *http.Request, location string) {
	http.Redirect(w, r, location, status302)
}

func (h *responder) Redirect303(w http.ResponseWriter, r *http.Request, location string) {
	http.Redirect(w, r, location, status303)
}

func (h *responder) Redirect307(w http.ResponseWriter, r *http.Request, location string) {
	http.Redirect(w, r, location, status307)
}

func (h *responder) Send400(w http.ResponseWriter, err error, message string) {
	h.logError(err, status400, message)
	h.send(w, status400, h.options.contentFormatter(
		h.options.errorFormatter(message)),
	)
}

func (h *responder) Send401(w http.ResponseWriter, err error, message string) {
	h.logError(err, status401, message)
	h.send(w, status401, h.options.contentFormatter(
		h.options.errorFormatter(message)),
	)
}

func (h *responder) Send403(w http.ResponseWriter, err error, message string) {
	h.logError(err, status403, message)
	h.send(w, status403, h.options.contentFormatter(
		h.options.errorFormatter(message)),
	)
}

func (h *responder) Send404(w http.ResponseWriter, err error, message string) {
	h.logError(err, status404, message)
	h.send(w, status404, h.options.contentFormatter(
		h.options.errorFormatter(message)),
	)
}

func (h *responder) Send500(w http.ResponseWriter, err error, message string) {
	h.logError(err, status500, message)
	h.send(w, status500, h.options.contentFormatter(
		h.options.errorFormatter(message)),
	)
}
