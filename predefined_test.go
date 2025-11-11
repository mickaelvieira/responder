package responder

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestJSONResponder(t *testing.T) {
	t.Run("returns a Responder interface", func(t *testing.T) {
		responder := JSONResponder()
		if responder == nil {
			t.Fatal("expected non-nil Responder")
		}
	})

	t.Run("sets correct content type", func(t *testing.T) {
		responder := JSONResponder()
		w := httptest.NewRecorder()
		responder.Send200(w, map[string]string{"message": "success"})

		contentType := w.Header().Get("Content-Type")
		expected := JSONContentType
		if contentType != expected {
			t.Errorf("expected Content-Type %q, got %q", expected, contentType)
		}
	})

	t.Run("formats error messages as JSON", func(t *testing.T) {
		// Create a custom content formatter that handles JSON marshaling
		jsonContentFormatter := func(content any) []byte {
			data, err := json.Marshal(content)
			if err != nil {
				return []byte(fmt.Sprintf("marshal error: %v", err))
			}
			return data
		}

		responder := JSONResponder(WithContentFormatter(jsonContentFormatter))
		w := httptest.NewRecorder()
		errorMessage := "validation failed"

		responder.Send400(w, errors.New("some error"), errorMessage)

		var result jsonError
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("failed to unmarshal response: %v (body: %s)", err, w.Body.String())
		}

		if result.Error != errorMessage {
			t.Errorf("expected error message %q, got %q", errorMessage, result.Error)
		}
	})

	t.Run("accepts additional options modifiers", func(t *testing.T) {
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
		responder := JSONResponder(WithLogger(logger))

		if responder == nil {
			t.Fatal("expected non-nil Responder with options")
		}
	})

	t.Run("default JSON error formatter is always applied", func(t *testing.T) {
		jsonContentFormatter := func(content any) []byte {
			data, _ := json.Marshal(content)
			return data
		}

		customFormatter := func(message any) any {
			return map[string]string{
				"custom_error": MessageToString(message),
				"formatted":    "true",
			}
		}

		// JSONResponder always applies jsonFormatter last, so it overrides custom formatters
		responder := JSONResponder(WithContentFormatter(jsonContentFormatter), WithErrorFormatter(customFormatter))
		w := httptest.NewRecorder()

		responder.Send400(w, errors.New("test"), "validation error")

		var result jsonError
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		// The default jsonError formatter is applied, not the custom one
		if result.Error != "validation error" {
			t.Errorf("expected error message %q, got %q", "validation error", result.Error)
		}
	})

	t.Run("works with all HTTP methods", func(t *testing.T) {
		testCases := []struct {
			name       string
			sendFunc   func(Responder, http.ResponseWriter)
			wantStatus int
		}{
			{
				name: "Send200",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send200(w, map[string]string{"status": "ok"})
				},
				wantStatus: http.StatusOK,
			},
			{
				name: "Send201",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send201(w, map[string]string{"status": "created"})
				},
				wantStatus: http.StatusCreated,
			},
			{
				name: "Send202",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send202(w, map[string]string{"status": "accepted"})
				},
				wantStatus: http.StatusAccepted,
			},
			{
				name: "Send204",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send204(w)
				},
				wantStatus: http.StatusNoContent,
			},
			{
				name: "Send400",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send400(w, errors.New("bad request"), "invalid input")
				},
				wantStatus: http.StatusBadRequest,
			},
			{
				name: "Send401",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send401(w, errors.New("unauthorized"), "authentication required")
				},
				wantStatus: http.StatusUnauthorized,
			},
			{
				name: "Send403",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send403(w, errors.New("forbidden"), "access denied")
				},
				wantStatus: http.StatusForbidden,
			},
			{
				name: "Send404",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send404(w, errors.New("not found"), "resource not found")
				},
				wantStatus: http.StatusNotFound,
			},
			{
				name: "Send500",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send500(w, errors.New("internal error"), "server error")
				},
				wantStatus: http.StatusInternalServerError,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				responder := JSONResponder()
				w := httptest.NewRecorder()

				tc.sendFunc(responder, w)

				if w.Code != tc.wantStatus {
					t.Errorf("expected status %d, got %d", tc.wantStatus, w.Code)
				}

				contentType := w.Header().Get("Content-Type")
				if contentType != JSONContentType {
					t.Errorf("expected Content-Type %q, got %q", JSONContentType, contentType)
				}
			})
		}
	})

	t.Run("marshals complex JSON structures correctly", func(t *testing.T) {
		jsonContentFormatter := func(content any) []byte {
			data, _ := json.Marshal(content)
			return data
		}

		type Response struct {
			Message string         `json:"message"`
			Data    map[string]int `json:"data"`
			Tags    []string       `json:"tags"`
		}

		responder := JSONResponder(WithContentFormatter(jsonContentFormatter))
		w := httptest.NewRecorder()

		expected := Response{
			Message: "success",
			Data:    map[string]int{"count": 42, "total": 100},
			Tags:    []string{"tag1", "tag2"},
		}

		responder.Send200(w, expected)

		var result Response
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("failed to unmarshal response: %v (body: %s)", err, w.Body.String())
		}

		if result.Message != expected.Message {
			t.Errorf("expected message %q, got %q", expected.Message, result.Message)
		}
		if result.Data["count"] != expected.Data["count"] {
			t.Errorf("expected count %d, got %d", expected.Data["count"], result.Data["count"])
		}
		if len(result.Tags) != len(expected.Tags) {
			t.Errorf("expected %d tags, got %d", len(expected.Tags), len(result.Tags))
		}
	})

	t.Run("multiple option modifiers applied correctly", func(t *testing.T) {
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
		customContentFormatter := func(content any) []byte {
			// Simple custom formatter that wraps content
			data, _ := json.Marshal(content)
			return data
		}

		responder := JSONResponder(
			WithLogger(logger),
			WithContentFormatter(customContentFormatter),
		)

		w := httptest.NewRecorder()
		responder.Send200(w, map[string]string{"test": "data"})

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var result map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if result["test"] != "data" {
			t.Errorf("expected data to be preserved with custom formatter")
		}
	})

	t.Run("accepts custom message types", func(t *testing.T) {
		jsonContentFormatter := func(content any) []byte {
			data, _ := json.Marshal(content)
			return data
		}

		type CustomErrorMessage struct {
			Code    string
			Message string
			Details string
		}

		// JSONResponder enforces jsonError format, but MessageToString handles custom types
		responder := JSONResponder(
			WithContentFormatter(jsonContentFormatter),
		)
		w := httptest.NewRecorder()

		// Pass custom struct - it will be converted to string by MessageToString
		// which returns GenericErrorMessage for non-string/non-error/non-Stringer types
		customMsg := CustomErrorMessage{
			Code:    "VALIDATION_ERROR",
			Message: "Invalid input provided",
			Details: "Field 'email' must be a valid email address",
		}

		responder.Send400(w, errors.New("validation failed"), customMsg)

		var result jsonError
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("failed to unmarshal response: %v (body: %s)", err, w.Body.String())
		}

		// Since CustomErrorMessage doesn't implement String() or Error(),
		// MessageToString returns GenericErrorMessage
		if result.Error != GenericErrorMessage {
			t.Errorf("expected error %q, got %q", GenericErrorMessage, result.Error)
		}
	})

	t.Run("handles error type as message", func(t *testing.T) {
		jsonContentFormatter := func(content any) []byte {
			data, _ := json.Marshal(content)
			return data
		}

		customFormatter := func(message any) any {
			// If the message is an error, extract it
			if err, ok := message.(error); ok {
				return jsonError{Error: err.Error()}
			}
			return jsonError{Error: MessageToString(message)}
		}

		responder := JSONResponder(
			WithContentFormatter(jsonContentFormatter),
			WithErrorFormatter(customFormatter),
		)
		w := httptest.NewRecorder()

		msgError := errors.New("database connection failed")
		responder.Send500(w, msgError, msgError)

		var result jsonError
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if result.Error != "database connection failed" {
			t.Errorf("expected error %q, got %q", "database connection failed", result.Error)
		}
	})

	t.Run("handles map type as message", func(t *testing.T) {
		jsonContentFormatter := func(content any) []byte {
			data, _ := json.Marshal(content)
			return data
		}

		responder := JSONResponder(
			WithContentFormatter(jsonContentFormatter),
		)
		w := httptest.NewRecorder()

		// Pass a map - it will be converted to string by MessageToString
		// which returns GenericErrorMessage for non-string/non-error/non-Stringer types
		msgMap := map[string]interface{}{
			"error":   "validation_failed",
			"field":   "email",
			"code":    400,
			"details": []string{"invalid format", "required field"},
		}

		responder.Send400(w, errors.New("validation error"), msgMap)

		var result jsonError
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		// JSONResponder enforces jsonError format
		// Map doesn't implement String() or Error(), so returns GenericErrorMessage
		if result.Error != GenericErrorMessage {
			t.Errorf("expected error %q, got %q", GenericErrorMessage, result.Error)
		}
	})
}

func TestTextResponder(t *testing.T) {
	t.Run("returns a Responder interface", func(t *testing.T) {
		responder := TextResponder()
		if responder == nil {
			t.Fatal("expected non-nil Responder")
		}
	})

	t.Run("sets correct content type", func(t *testing.T) {
		responder := TextResponder()
		w := httptest.NewRecorder()
		responder.Send200(w, "plain text response")

		contentType := w.Header().Get("Content-Type")
		expected := TextContentType
		if contentType != expected {
			t.Errorf("expected Content-Type %q, got %q", expected, contentType)
		}
	})

	t.Run("sends plain text content", func(t *testing.T) {
		responder := TextResponder()
		w := httptest.NewRecorder()
		message := "Hello, World!"

		responder.Send200(w, message)

		if w.Body.String() != message {
			t.Errorf("expected body %q, got %q", message, w.Body.String())
		}
	})

	t.Run("sends error messages as plain text", func(t *testing.T) {
		responder := TextResponder()
		w := httptest.NewRecorder()
		errorMessage := "validation failed"

		responder.Send400(w, errors.New("some error"), errorMessage)

		if w.Body.String() != errorMessage {
			t.Errorf("expected error message %q, got %q", errorMessage, w.Body.String())
		}
	})

	t.Run("accepts additional options modifiers", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		responder := TextResponder(WithLogger(logger))

		if responder == nil {
			t.Fatal("expected non-nil Responder with options")
		}
	})

	t.Run("applies custom error formatter", func(t *testing.T) {
		customFormatter := func(message any) any {
			return fmt.Sprintf("ERROR: %s", MessageToString(message))
		}

		responder := TextResponder(WithErrorFormatter(customFormatter))
		w := httptest.NewRecorder()

		responder.Send400(w, errors.New("test"), "bad request")

		expected := "ERROR: bad request"
		if w.Body.String() != expected {
			t.Errorf("expected body %q, got %q", expected, w.Body.String())
		}
	})

	t.Run("works with all HTTP methods", func(t *testing.T) {
		testCases := []struct {
			name       string
			sendFunc   func(Responder, http.ResponseWriter)
			wantStatus int
			wantBody   string
		}{
			{
				name: "Send200",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send200(w, "success")
				},
				wantStatus: http.StatusOK,
				wantBody:   "success",
			},
			{
				name: "Send201",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send201(w, "created")
				},
				wantStatus: http.StatusCreated,
				wantBody:   "created",
			},
			{
				name: "Send202",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send202(w, "accepted")
				},
				wantStatus: http.StatusAccepted,
				wantBody:   "accepted",
			},
			{
				name: "Send204",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send204(w)
				},
				wantStatus: http.StatusNoContent,
				wantBody:   "",
			},
			{
				name: "Send400",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send400(w, errors.New("bad request"), "invalid input")
				},
				wantStatus: http.StatusBadRequest,
				wantBody:   "invalid input",
			},
			{
				name: "Send401",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send401(w, errors.New("unauthorized"), "authentication required")
				},
				wantStatus: http.StatusUnauthorized,
				wantBody:   "authentication required",
			},
			{
				name: "Send403",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send403(w, errors.New("forbidden"), "access denied")
				},
				wantStatus: http.StatusForbidden,
				wantBody:   "access denied",
			},
			{
				name: "Send404",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send404(w, errors.New("not found"), "resource not found")
				},
				wantStatus: http.StatusNotFound,
				wantBody:   "resource not found",
			},
			{
				name: "Send500",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send500(w, errors.New("internal error"), "server error")
				},
				wantStatus: http.StatusInternalServerError,
				wantBody:   "server error",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				responder := TextResponder()
				w := httptest.NewRecorder()

				tc.sendFunc(responder, w)

				if w.Code != tc.wantStatus {
					t.Errorf("expected status %d, got %d", tc.wantStatus, w.Code)
				}

				contentType := w.Header().Get("Content-Type")
				if contentType != TextContentType {
					t.Errorf("expected Content-Type %q, got %q", TextContentType, contentType)
				}

				if w.Body.String() != tc.wantBody {
					t.Errorf("expected body %q, got %q", tc.wantBody, w.Body.String())
				}
			})
		}
	})

	t.Run("handles byte slice content", func(t *testing.T) {
		responder := TextResponder()
		w := httptest.NewRecorder()
		content := []byte("byte content")

		responder.Send200(w, content)

		if w.Body.String() != string(content) {
			t.Errorf("expected body %q, got %q", string(content), w.Body.String())
		}
	})

	t.Run("multiple option modifiers applied correctly", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		customFormatter := func(message any) any {
			return fmt.Sprintf("[ERROR] %s", message)
		}

		responder := TextResponder(
			WithLogger(logger),
			WithErrorFormatter(customFormatter),
		)

		w := httptest.NewRecorder()
		responder.Send500(w, errors.New("test"), "internal error")

		expected := "[ERROR] internal error"
		if w.Body.String() != expected {
			t.Errorf("expected body %q, got %q", expected, w.Body.String())
		}

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}
	})

	t.Run("handles empty string content", func(t *testing.T) {
		responder := TextResponder()
		w := httptest.NewRecorder()

		responder.Send200(w, "")

		if w.Body.String() != "" {
			t.Errorf("expected empty body, got %q", w.Body.String())
		}

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("handles nil content", func(t *testing.T) {
		responder := TextResponder()
		w := httptest.NewRecorder()

		responder.Send204(w)

		if w.Body.String() != "" {
			t.Errorf("expected empty body for 204, got %q", w.Body.String())
		}

		if w.Code != http.StatusNoContent {
			t.Errorf("expected status 204, got %d", w.Code)
		}
	})

	t.Run("accepts custom message types with ErrorFormatter", func(t *testing.T) {
		type CustomError struct {
			Code    int
			Message string
		}

		// Custom formatter that formats struct messages as text
		customFormatter := func(message any) any {
			switch v := message.(type) {
			case CustomError:
				return fmt.Sprintf("Error %d: %s", v.Code, v.Message)
			case error:
				return fmt.Sprintf("ERROR: %s", v.Error())
			default:
				return MessageToString(message)
			}
		}

		responder := TextResponder(WithErrorFormatter(customFormatter))
		w := httptest.NewRecorder()

		customMsg := CustomError{
			Code:    1001,
			Message: "Database connection timeout",
		}

		responder.Send500(w, errors.New("db error"), customMsg)

		expected := "Error 1001: Database connection timeout"
		if w.Body.String() != expected {
			t.Errorf("expected body %q, got %q", expected, w.Body.String())
		}
	})

	t.Run("handles error type as message", func(t *testing.T) {
		customFormatter := func(message any) any {
			if err, ok := message.(error); ok {
				return fmt.Sprintf("Error occurred: %s", err.Error())
			}
			return MessageToString(message)
		}

		responder := TextResponder(WithErrorFormatter(customFormatter))
		w := httptest.NewRecorder()

		msgError := errors.New("authentication failed")
		responder.Send401(w, msgError, msgError)

		expected := "Error occurred: authentication failed"
		if w.Body.String() != expected {
			t.Errorf("expected body %q, got %q", expected, w.Body.String())
		}
	})

	t.Run("handles fmt.Stringer as message", func(t *testing.T) {
		customFormatter := func(message any) any {
			if stringer, ok := message.(fmt.Stringer); ok {
				return stringer.String()
			}
			return MessageToString(message)
		}

		responder := TextResponder(WithErrorFormatter(customFormatter))
		w := httptest.NewRecorder()

		// Use an error (which implements fmt.Stringer) as the message
		msg := errors.New("VALIDATION: invalid email format")
		responder.Send400(w, msg, msg)

		expected := "VALIDATION: invalid email format"
		if w.Body.String() != expected {
			t.Errorf("expected body %q, got %q", expected, w.Body.String())
		}
	})
}

func TestHTMLResponder(t *testing.T) {
	t.Run("returns a Responder interface", func(t *testing.T) {
		responder := HTMLResponder()
		if responder == nil {
			t.Fatal("expected non-nil Responder")
		}
	})

	t.Run("sets correct content type", func(t *testing.T) {
		responder := HTMLResponder()
		w := httptest.NewRecorder()
		responder.Send200(w, "<html><body>Hello</body></html>")

		contentType := w.Header().Get("Content-Type")
		expected := HTMLContentType
		if contentType != expected {
			t.Errorf("expected Content-Type %q, got %q", expected, contentType)
		}
	})

	t.Run("sends HTML content", func(t *testing.T) {
		responder := HTMLResponder()
		w := httptest.NewRecorder()
		htmlContent := "<html><head><title>Test</title></head><body><h1>Hello, World!</h1></body></html>"

		responder.Send200(w, htmlContent)

		if w.Body.String() != htmlContent {
			t.Errorf("expected body %q, got %q", htmlContent, w.Body.String())
		}
	})

	t.Run("sends error messages as HTML", func(t *testing.T) {
		responder := HTMLResponder()
		w := httptest.NewRecorder()
		errorMessage := "<p>Validation failed</p>"

		responder.Send400(w, errors.New("some error"), errorMessage)

		if w.Body.String() != errorMessage {
			t.Errorf("expected error message %q, got %q", errorMessage, w.Body.String())
		}
	})

	t.Run("accepts additional options modifiers", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		responder := HTMLResponder(WithLogger(logger))

		if responder == nil {
			t.Fatal("expected non-nil Responder with options")
		}
	})

	t.Run("applies custom error formatter for HTML", func(t *testing.T) {
		customFormatter := func(message any) any {
			return fmt.Sprintf("<div class='error'>%s</div>", message)
		}

		responder := HTMLResponder(WithErrorFormatter(customFormatter))
		w := httptest.NewRecorder()

		responder.Send404(w, errors.New("test"), "Page not found")

		expected := "<div class='error'>Page not found</div>"
		if w.Body.String() != expected {
			t.Errorf("expected body %q, got %q", expected, w.Body.String())
		}
	})

	t.Run("works with all HTTP methods", func(t *testing.T) {
		testCases := []struct {
			name       string
			sendFunc   func(Responder, http.ResponseWriter)
			wantStatus int
			wantBody   string
		}{
			{
				name: "Send200",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send200(w, "<p>Success</p>")
				},
				wantStatus: http.StatusOK,
				wantBody:   "<p>Success</p>",
			},
			{
				name: "Send201",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send201(w, "<p>Created</p>")
				},
				wantStatus: http.StatusCreated,
				wantBody:   "<p>Created</p>",
			},
			{
				name: "Send202",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send202(w, "<p>Accepted</p>")
				},
				wantStatus: http.StatusAccepted,
				wantBody:   "<p>Accepted</p>",
			},
			{
				name: "Send204",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send204(w)
				},
				wantStatus: http.StatusNoContent,
				wantBody:   "",
			},
			{
				name: "Send400",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send400(w, errors.New("bad request"), "<p>Invalid input</p>")
				},
				wantStatus: http.StatusBadRequest,
				wantBody:   "<p>Invalid input</p>",
			},
			{
				name: "Send401",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send401(w, errors.New("unauthorized"), "<p>Authentication required</p>")
				},
				wantStatus: http.StatusUnauthorized,
				wantBody:   "<p>Authentication required</p>",
			},
			{
				name: "Send403",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send403(w, errors.New("forbidden"), "<p>Access denied</p>")
				},
				wantStatus: http.StatusForbidden,
				wantBody:   "<p>Access denied</p>",
			},
			{
				name: "Send404",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send404(w, errors.New("not found"), "<p>Resource not found</p>")
				},
				wantStatus: http.StatusNotFound,
				wantBody:   "<p>Resource not found</p>",
			},
			{
				name: "Send500",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send500(w, errors.New("internal error"), "<p>Server error</p>")
				},
				wantStatus: http.StatusInternalServerError,
				wantBody:   "<p>Server error</p>",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				responder := HTMLResponder()
				w := httptest.NewRecorder()

				tc.sendFunc(responder, w)

				if w.Code != tc.wantStatus {
					t.Errorf("expected status %d, got %d", tc.wantStatus, w.Code)
				}

				contentType := w.Header().Get("Content-Type")
				if contentType != HTMLContentType {
					t.Errorf("expected Content-Type %q, got %q", HTMLContentType, contentType)
				}

				if w.Body.String() != tc.wantBody {
					t.Errorf("expected body %q, got %q", tc.wantBody, w.Body.String())
				}
			})
		}
	})

	t.Run("handles complex HTML structures", func(t *testing.T) {
		responder := HTMLResponder()
		w := httptest.NewRecorder()
		htmlContent := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Test Page</title>
    <style>
        body { font-family: Arial, sans-serif; }
        .container { max-width: 800px; margin: 0 auto; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Welcome</h1>
        <p>This is a test page.</p>
        <ul>
            <li>Item 1</li>
            <li>Item 2</li>
            <li>Item 3</li>
        </ul>
    </div>
    <script>
        console.log('Page loaded');
    </script>
</body>
</html>`

		responder.Send200(w, htmlContent)

		if w.Body.String() != htmlContent {
			t.Errorf("HTML content mismatch")
		}

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("handles byte slice HTML content", func(t *testing.T) {
		responder := HTMLResponder()
		w := httptest.NewRecorder()
		content := []byte("<html><body>Byte content</body></html>")

		responder.Send200(w, content)

		if w.Body.String() != string(content) {
			t.Errorf("expected body %q, got %q", string(content), w.Body.String())
		}
	})

	t.Run("multiple option modifiers applied correctly", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		customFormatter := func(message any) any {
			return fmt.Sprintf(`<div class="alert alert-error">
				<h2>Error</h2>
				<p>%s</p>
			</div>`, message)
		}

		responder := HTMLResponder(
			WithLogger(logger),
			WithErrorFormatter(customFormatter),
		)

		w := httptest.NewRecorder()
		responder.Send500(w, errors.New("test"), "Database connection failed")

		expectedBody := `<div class="alert alert-error">
				<h2>Error</h2>
				<p>Database connection failed</p>
			</div>`
		if w.Body.String() != expectedBody {
			t.Errorf("expected body %q, got %q", expectedBody, w.Body.String())
		}

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}
	})

	t.Run("handles empty HTML content", func(t *testing.T) {
		responder := HTMLResponder()
		w := httptest.NewRecorder()

		responder.Send200(w, "")

		if w.Body.String() != "" {
			t.Errorf("expected empty body, got %q", w.Body.String())
		}

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("handles HTML fragments", func(t *testing.T) {
		responder := HTMLResponder()
		w := httptest.NewRecorder()
		fragment := `<div id="content">
			<h2>Section Title</h2>
			<p>Some paragraph text.</p>
		</div>`

		responder.Send200(w, fragment)

		if w.Body.String() != fragment {
			t.Errorf("expected body %q, got %q", fragment, w.Body.String())
		}
	})

	t.Run("handles HTML with special characters", func(t *testing.T) {
		responder := HTMLResponder()
		w := httptest.NewRecorder()
		htmlWithSpecialChars := `<p>Special characters: &lt; &gt; &amp; &quot; &#39;</p>`

		responder.Send200(w, htmlWithSpecialChars)

		if w.Body.String() != htmlWithSpecialChars {
			t.Errorf("expected body %q, got %q", htmlWithSpecialChars, w.Body.String())
		}
	})

	t.Run("accepts custom message types with ErrorFormatter", func(t *testing.T) {
		type HTMLError struct {
			Title   string
			Message string
			Code    int
		}

		// Custom formatter that formats struct messages as HTML
		customFormatter := func(message any) any {
			switch v := message.(type) {
			case HTMLError:
				return fmt.Sprintf(`<div class="error">
					<h3>%s</h3>
					<p>%s</p>
					<small>Error Code: %d</small>
				</div>`, v.Title, v.Message, v.Code)
			case error:
				return fmt.Sprintf(`<div class="error"><p>%s</p></div>`, v.Error())
			default:
				return fmt.Sprintf(`<p>%s</p>`, MessageToString(message))
			}
		}

		responder := HTMLResponder(WithErrorFormatter(customFormatter))
		w := httptest.NewRecorder()

		customMsg := HTMLError{
			Title:   "Validation Error",
			Message: "The email field is required",
			Code:    4001,
		}

		responder.Send400(w, errors.New("validation"), customMsg)

		expected := `<div class="error">
					<h3>Validation Error</h3>
					<p>The email field is required</p>
					<small>Error Code: 4001</small>
				</div>`
		if w.Body.String() != expected {
			t.Errorf("expected body %q, got %q", expected, w.Body.String())
		}
	})

	t.Run("handles error type as HTML message", func(t *testing.T) {
		customFormatter := func(message any) any {
			if err, ok := message.(error); ok {
				return fmt.Sprintf(`<div class="alert alert-danger">%s</div>`, err.Error())
			}
			return MessageToString(message)
		}

		responder := HTMLResponder(WithErrorFormatter(customFormatter))
		w := httptest.NewRecorder()

		msgError := errors.New("resource not found")
		responder.Send404(w, msgError, msgError)

		expected := `<div class="alert alert-danger">resource not found</div>`
		if w.Body.String() != expected {
			t.Errorf("expected body %q, got %q", expected, w.Body.String())
		}
	})
}

func TestCSVResponder(t *testing.T) {
	t.Run("returns a Responder interface", func(t *testing.T) {
		responder := CSVResponder()
		if responder == nil {
			t.Fatal("expected non-nil Responder")
		}
	})

	t.Run("sets correct content type", func(t *testing.T) {
		responder := CSVResponder()
		w := httptest.NewRecorder()
		responder.Send200(w, "name,age\nJohn,30\nJane,25")

		contentType := w.Header().Get("Content-Type")
		expected := CSVContentType
		if contentType != expected {
			t.Errorf("expected Content-Type %q, got %q", expected, contentType)
		}
	})

	t.Run("sends CSV content", func(t *testing.T) {
		responder := CSVResponder()
		w := httptest.NewRecorder()
		csvContent := "id,name,email\n1,John Doe,john@example.com\n2,Jane Smith,jane@example.com"

		responder.Send200(w, csvContent)

		if w.Body.String() != csvContent {
			t.Errorf("expected body %q, got %q", csvContent, w.Body.String())
		}
	})

	t.Run("sends error messages as CSV text", func(t *testing.T) {
		responder := CSVResponder()
		w := httptest.NewRecorder()
		errorMessage := "Invalid CSV format"

		responder.Send400(w, errors.New("parse error"), errorMessage)

		if w.Body.String() != errorMessage {
			t.Errorf("expected error message %q, got %q", errorMessage, w.Body.String())
		}
	})

	t.Run("accepts additional options modifiers", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		responder := CSVResponder(WithLogger(logger))

		if responder == nil {
			t.Fatal("expected non-nil Responder with options")
		}
	})

	t.Run("applies custom error formatter", func(t *testing.T) {
		customFormatter := func(message any) any {
			return fmt.Sprintf("error\n%s", message)
		}

		responder := CSVResponder(WithErrorFormatter(customFormatter))
		w := httptest.NewRecorder()

		responder.Send400(w, errors.New("test"), "bad data")

		expected := "error\nbad data"
		if w.Body.String() != expected {
			t.Errorf("expected body %q, got %q", expected, w.Body.String())
		}
	})

	t.Run("works with all HTTP methods", func(t *testing.T) {
		testCases := []struct {
			name       string
			sendFunc   func(Responder, http.ResponseWriter)
			wantStatus int
			wantBody   string
		}{
			{
				name: "Send200",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send200(w, "status\nok")
				},
				wantStatus: http.StatusOK,
				wantBody:   "status\nok",
			},
			{
				name: "Send201",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send201(w, "status\ncreated")
				},
				wantStatus: http.StatusCreated,
				wantBody:   "status\ncreated",
			},
			{
				name: "Send202",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send202(w, "status\naccepted")
				},
				wantStatus: http.StatusAccepted,
				wantBody:   "status\naccepted",
			},
			{
				name: "Send204",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send204(w)
				},
				wantStatus: http.StatusNoContent,
				wantBody:   "",
			},
			{
				name: "Send400",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send400(w, errors.New("bad request"), "Invalid CSV")
				},
				wantStatus: http.StatusBadRequest,
				wantBody:   "Invalid CSV",
			},
			{
				name: "Send401",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send401(w, errors.New("unauthorized"), "Authentication required")
				},
				wantStatus: http.StatusUnauthorized,
				wantBody:   "Authentication required",
			},
			{
				name: "Send403",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send403(w, errors.New("forbidden"), "Access denied")
				},
				wantStatus: http.StatusForbidden,
				wantBody:   "Access denied",
			},
			{
				name: "Send404",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send404(w, errors.New("not found"), "Resource not found")
				},
				wantStatus: http.StatusNotFound,
				wantBody:   "Resource not found",
			},
			{
				name: "Send500",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send500(w, errors.New("internal error"), "Server error")
				},
				wantStatus: http.StatusInternalServerError,
				wantBody:   "Server error",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				responder := CSVResponder()
				w := httptest.NewRecorder()

				tc.sendFunc(responder, w)

				if w.Code != tc.wantStatus {
					t.Errorf("expected status %d, got %d", tc.wantStatus, w.Code)
				}

				contentType := w.Header().Get("Content-Type")
				if contentType != CSVContentType {
					t.Errorf("expected Content-Type %q, got %q", CSVContentType, contentType)
				}

				if w.Body.String() != tc.wantBody {
					t.Errorf("expected body %q, got %q", tc.wantBody, w.Body.String())
				}
			})
		}
	})

	t.Run("handles CSV with headers and multiple rows", func(t *testing.T) {
		responder := CSVResponder()
		w := httptest.NewRecorder()
		csvContent := `name,age,city,country
John Doe,30,New York,USA
Jane Smith,25,London,UK
Bob Johnson,35,Toronto,Canada
Alice Brown,28,Sydney,Australia`

		responder.Send200(w, csvContent)

		if w.Body.String() != csvContent {
			t.Errorf("CSV content mismatch")
		}

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("handles CSV with quoted fields", func(t *testing.T) {
		responder := CSVResponder()
		w := httptest.NewRecorder()
		csvContent := `name,description,price
"Widget A","A simple, useful widget",19.99
"Widget B","Complex ""super"" widget",29.99
"Widget, C","Widget with comma",15.50`

		responder.Send200(w, csvContent)

		if w.Body.String() != csvContent {
			t.Errorf("expected body %q, got %q", csvContent, w.Body.String())
		}
	})

	t.Run("handles byte slice CSV content", func(t *testing.T) {
		responder := CSVResponder()
		w := httptest.NewRecorder()
		content := []byte("id,value\n1,test\n2,data")

		responder.Send200(w, content)

		if w.Body.String() != string(content) {
			t.Errorf("expected body %q, got %q", string(content), w.Body.String())
		}
	})

	t.Run("multiple option modifiers applied correctly", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		customFormatter := func(message any) any {
			return fmt.Sprintf("error_type,message\ndata_error,%s", message)
		}

		responder := CSVResponder(
			WithLogger(logger),
			WithErrorFormatter(customFormatter),
		)

		w := httptest.NewRecorder()
		responder.Send500(w, errors.New("test"), "Database query failed")

		expected := "error_type,message\ndata_error,Database query failed"
		if w.Body.String() != expected {
			t.Errorf("expected body %q, got %q", expected, w.Body.String())
		}

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}
	})

	t.Run("handles empty CSV content", func(t *testing.T) {
		responder := CSVResponder()
		w := httptest.NewRecorder()

		responder.Send200(w, "")

		if w.Body.String() != "" {
			t.Errorf("expected empty body, got %q", w.Body.String())
		}

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("handles CSV with only headers", func(t *testing.T) {
		responder := CSVResponder()
		w := httptest.NewRecorder()
		csvContent := "id,name,email"

		responder.Send200(w, csvContent)

		if w.Body.String() != csvContent {
			t.Errorf("expected body %q, got %q", csvContent, w.Body.String())
		}
	})

	t.Run("handles CSV with different delimiters in content", func(t *testing.T) {
		responder := CSVResponder()
		w := httptest.NewRecorder()
		csvContent := `product,tags
laptop,"electronics,computers,portable"
phone,"electronics,mobile,smart"
desk,"furniture,office"`

		responder.Send200(w, csvContent)

		if w.Body.String() != csvContent {
			t.Errorf("expected body %q, got %q", csvContent, w.Body.String())
		}
	})

	t.Run("handles large CSV dataset", func(t *testing.T) {
		responder := CSVResponder()
		w := httptest.NewRecorder()

		// Build a large CSV
		var csvBuilder strings.Builder
		csvBuilder.WriteString("id,timestamp,value,status\n")
		for i := 1; i <= 1000; i++ {
			csvBuilder.WriteString(fmt.Sprintf("%d,2025-11-10T12:00:%02d,%.2f,active\n", i, i%60, float64(i)*1.5))
		}
		csvContent := csvBuilder.String()

		responder.Send200(w, csvContent)

		if w.Body.String() != csvContent {
			t.Errorf("CSV content length mismatch: expected %d bytes, got %d bytes",
				len(csvContent), len(w.Body.String()))
		}

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("handles CSV with special characters and escaping", func(t *testing.T) {
		responder := CSVResponder()
		w := httptest.NewRecorder()
		csvContent := `name,note
John,"Contains ""quotes"" and commas, see?"
Jane,"Line breaks:
are handled"
Bob,"Special chars: @#$%^&*()"`

		responder.Send200(w, csvContent)

		if w.Body.String() != csvContent {
			t.Errorf("expected body %q, got %q", csvContent, w.Body.String())
		}
	})

	t.Run("accepts custom message types with ErrorFormatter", func(t *testing.T) {
		type CSVError struct {
			ErrorType string
			Row       int
			Column    string
		}

		// Custom formatter that formats struct messages as CSV
		customFormatter := func(message any) any {
			switch v := message.(type) {
			case CSVError:
				return fmt.Sprintf("error_type,row,column\n%s,%d,%s", v.ErrorType, v.Row, v.Column)
			case error:
				return fmt.Sprintf("error\n%s", v.Error())
			default:
				return MessageToString(message)
			}
		}

		responder := CSVResponder(WithErrorFormatter(customFormatter))
		w := httptest.NewRecorder()

		customMsg := CSVError{
			ErrorType: "INVALID_FORMAT",
			Row:       42,
			Column:    "email",
		}

		responder.Send400(w, errors.New("csv error"), customMsg)

		expected := "error_type,row,column\nINVALID_FORMAT,42,email"
		if w.Body.String() != expected {
			t.Errorf("expected body %q, got %q", expected, w.Body.String())
		}
	})

	t.Run("handles error type as CSV message", func(t *testing.T) {
		customFormatter := func(message any) any {
			if err, ok := message.(error); ok {
				return fmt.Sprintf("status,message\nerror,%s", err.Error())
			}
			return MessageToString(message)
		}

		responder := CSVResponder(WithErrorFormatter(customFormatter))
		w := httptest.NewRecorder()

		msgError := errors.New("data processing failed")
		responder.Send500(w, msgError, msgError)

		expected := "status,message\nerror,data processing failed"
		if w.Body.String() != expected {
			t.Errorf("expected body %q, got %q", expected, w.Body.String())
		}
	})
}

func TestXMLResponder(t *testing.T) {
	t.Run("returns a Responder interface", func(t *testing.T) {
		responder := XMLResponder()
		if responder == nil {
			t.Fatal("expected non-nil Responder")
		}
	})

	t.Run("sets correct content type", func(t *testing.T) {
		responder := XMLResponder()
		w := httptest.NewRecorder()
		responder.Send200(w, "<root><message>success</message></root>")

		contentType := w.Header().Get("Content-Type")
		expected := XMLContentType
		if contentType != expected {
			t.Errorf("expected Content-Type %q, got %q", expected, contentType)
		}
	})

	t.Run("sends XML content", func(t *testing.T) {
		responder := XMLResponder()
		w := httptest.NewRecorder()
		xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<users>
	<user>
		<id>1</id>
		<name>John Doe</name>
		<email>john@example.com</email>
	</user>
	<user>
		<id>2</id>
		<name>Jane Smith</name>
		<email>jane@example.com</email>
	</user>
</users>`

		responder.Send200(w, xmlContent)

		if w.Body.String() != xmlContent {
			t.Errorf("expected body %q, got %q", xmlContent, w.Body.String())
		}
	})

	t.Run("sends error messages as XML text", func(t *testing.T) {
		responder := XMLResponder()
		w := httptest.NewRecorder()
		errorMessage := "<error>Invalid XML format</error>"

		responder.Send400(w, errors.New("parse error"), errorMessage)

		if w.Body.String() != errorMessage {
			t.Errorf("expected error message %q, got %q", errorMessage, w.Body.String())
		}
	})

	t.Run("accepts additional options modifiers", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		responder := XMLResponder(WithLogger(logger))

		if responder == nil {
			t.Fatal("expected non-nil Responder with options")
		}
	})

	t.Run("applies custom error formatter", func(t *testing.T) {
		customFormatter := func(message any) any {
			return fmt.Sprintf("<error><message>%s</message></error>", message)
		}

		responder := XMLResponder(WithErrorFormatter(customFormatter))
		w := httptest.NewRecorder()

		responder.Send400(w, errors.New("test"), "bad data")

		expected := "<error><message>bad data</message></error>"
		if w.Body.String() != expected {
			t.Errorf("expected body %q, got %q", expected, w.Body.String())
		}
	})

	t.Run("works with all HTTP methods", func(t *testing.T) {
		testCases := []struct {
			name       string
			sendFunc   func(Responder, http.ResponseWriter)
			wantStatus int
			wantBody   string
		}{
			{
				name: "Send200",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send200(w, "<status>ok</status>")
				},
				wantStatus: http.StatusOK,
				wantBody:   "<status>ok</status>",
			},
			{
				name: "Send201",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send201(w, "<status>created</status>")
				},
				wantStatus: http.StatusCreated,
				wantBody:   "<status>created</status>",
			},
			{
				name: "Send202",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send202(w, "<status>accepted</status>")
				},
				wantStatus: http.StatusAccepted,
				wantBody:   "<status>accepted</status>",
			},
			{
				name: "Send204",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send204(w)
				},
				wantStatus: http.StatusNoContent,
				wantBody:   "",
			},
			{
				name: "Send400",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send400(w, errors.New("bad request"), "<error>Invalid XML</error>")
				},
				wantStatus: http.StatusBadRequest,
				wantBody:   "<error>Invalid XML</error>",
			},
			{
				name: "Send401",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send401(w, errors.New("unauthorized"), "<error>Authentication required</error>")
				},
				wantStatus: http.StatusUnauthorized,
				wantBody:   "<error>Authentication required</error>",
			},
			{
				name: "Send403",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send403(w, errors.New("forbidden"), "<error>Access denied</error>")
				},
				wantStatus: http.StatusForbidden,
				wantBody:   "<error>Access denied</error>",
			},
			{
				name: "Send404",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send404(w, errors.New("not found"), "<error>Resource not found</error>")
				},
				wantStatus: http.StatusNotFound,
				wantBody:   "<error>Resource not found</error>",
			},
			{
				name: "Send500",
				sendFunc: func(r Responder, w http.ResponseWriter) {
					r.Send500(w, errors.New("internal error"), "<error>Server error</error>")
				},
				wantStatus: http.StatusInternalServerError,
				wantBody:   "<error>Server error</error>",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				responder := XMLResponder()
				w := httptest.NewRecorder()

				tc.sendFunc(responder, w)

				if w.Code != tc.wantStatus {
					t.Errorf("expected status %d, got %d", tc.wantStatus, w.Code)
				}

				contentType := w.Header().Get("Content-Type")
				if contentType != XMLContentType {
					t.Errorf("expected Content-Type %q, got %q", XMLContentType, contentType)
				}

				if w.Body.String() != tc.wantBody {
					t.Errorf("expected body %q, got %q", tc.wantBody, w.Body.String())
				}
			})
		}
	})

	t.Run("handles complex XML structure", func(t *testing.T) {
		responder := XMLResponder()
		w := httptest.NewRecorder()
		xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<catalog>
	<book id="bk101">
		<author>Gambardella, Matthew</author>
		<title>XML Developer's Guide</title>
		<genre>Computer</genre>
		<price>44.95</price>
		<publish_date>2000-10-01</publish_date>
		<description>An in-depth look at creating applications with XML.</description>
	</book>
	<book id="bk102">
		<author>Ralls, Kim</author>
		<title>Midnight Rain</title>
		<genre>Fantasy</genre>
		<price>5.95</price>
		<publish_date>2000-12-16</publish_date>
		<description>A former architect battles corporate zombies.</description>
	</book>
</catalog>`

		responder.Send200(w, xmlContent)

		if w.Body.String() != xmlContent {
			t.Errorf("XML content mismatch")
		}

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("handles byte slice XML content", func(t *testing.T) {
		responder := XMLResponder()
		w := httptest.NewRecorder()
		content := []byte("<data><value>test</value></data>")

		responder.Send200(w, content)

		if w.Body.String() != string(content) {
			t.Errorf("expected body %q, got %q", string(content), w.Body.String())
		}
	})

	t.Run("multiple option modifiers applied correctly", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		customFormatter := func(message any) any {
			return fmt.Sprintf(`<?xml version="1.0"?>
<error>
	<type>system</type>
	<message>%s</message>
</error>`, message)
		}

		responder := XMLResponder(
			WithLogger(logger),
			WithErrorFormatter(customFormatter),
		)

		w := httptest.NewRecorder()
		responder.Send500(w, errors.New("test"), "Database connection failed")

		expected := `<?xml version="1.0"?>
<error>
	<type>system</type>
	<message>Database connection failed</message>
</error>`
		if w.Body.String() != expected {
			t.Errorf("expected body %q, got %q", expected, w.Body.String())
		}

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}
	})

	t.Run("handles empty XML content", func(t *testing.T) {
		responder := XMLResponder()
		w := httptest.NewRecorder()

		responder.Send200(w, "")

		if w.Body.String() != "" {
			t.Errorf("expected empty body, got %q", w.Body.String())
		}

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("handles XML with CDATA sections", func(t *testing.T) {
		responder := XMLResponder()
		w := httptest.NewRecorder()
		xmlContent := `<message><![CDATA[This is <markup> & special characters]]></message>`

		responder.Send200(w, xmlContent)

		if w.Body.String() != xmlContent {
			t.Errorf("expected body %q, got %q", xmlContent, w.Body.String())
		}
	})

	t.Run("handles XML with namespaces", func(t *testing.T) {
		responder := XMLResponder()
		w := httptest.NewRecorder()
		xmlContent := `<?xml version="1.0"?>
<root xmlns:h="http://www.w3.org/TR/html4/" xmlns:f="https://www.example.com/furniture">
	<h:table>
		<h:tr>
			<h:td>Apples</h:td>
			<h:td>Bananas</h:td>
		</h:tr>
	</h:table>
	<f:table>
		<f:name>Coffee Table</f:name>
		<f:width>80</f:width>
	</f:table>
</root>`

		responder.Send200(w, xmlContent)

		if w.Body.String() != xmlContent {
			t.Errorf("expected body %q, got %q", xmlContent, w.Body.String())
		}
	})

	t.Run("handles XML with attributes and special characters", func(t *testing.T) {
		responder := XMLResponder()
		w := httptest.NewRecorder()
		xmlContent := `<product id="123" available="true">
	<name lang="en">Coffee &amp; Tea</name>
	<description>Great for &lt;morning&gt; &quot;refreshment&quot;</description>
</product>`

		responder.Send200(w, xmlContent)

		if w.Body.String() != xmlContent {
			t.Errorf("expected body %q, got %q", xmlContent, w.Body.String())
		}
	})

	t.Run("accepts custom message types with ErrorFormatter", func(t *testing.T) {
		type XMLError struct {
			Code    string
			Message string
			Details string
		}

		// Custom formatter that formats struct messages as XML
		customFormatter := func(message any) any {
			switch v := message.(type) {
			case XMLError:
				return fmt.Sprintf(`<error>
	<code>%s</code>
	<message>%s</message>
	<details>%s</details>
</error>`, v.Code, v.Message, v.Details)
			case error:
				return fmt.Sprintf(`<error><message>%s</message></error>`, v.Error())
			default:
				return fmt.Sprintf(`<message>%s</message>`, MessageToString(message))
			}
		}

		responder := XMLResponder(WithErrorFormatter(customFormatter))
		w := httptest.NewRecorder()

		customMsg := XMLError{
			Code:    "AUTH_FAILED",
			Message: "Authentication required",
			Details: "Token has expired",
		}

		responder.Send401(w, errors.New("auth error"), customMsg)

		expected := `<error>
	<code>AUTH_FAILED</code>
	<message>Authentication required</message>
	<details>Token has expired</details>
</error>`
		if w.Body.String() != expected {
			t.Errorf("expected body %q, got %q", expected, w.Body.String())
		}
	})

	t.Run("handles error type as XML message", func(t *testing.T) {
		customFormatter := func(message any) any {
			if err, ok := message.(error); ok {
				return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<error>
	<message>%s</message>
	<timestamp>2025-11-10T12:00:00Z</timestamp>
</error>`, err.Error())
			}
			return MessageToString(message)
		}

		responder := XMLResponder(WithErrorFormatter(customFormatter))
		w := httptest.NewRecorder()

		msgError := errors.New("service unavailable")
		responder.Send500(w, msgError, msgError)

		expected := `<?xml version="1.0" encoding="UTF-8"?>
<error>
	<message>service unavailable</message>
	<timestamp>2025-11-10T12:00:00Z</timestamp>
</error>`
		if w.Body.String() != expected {
			t.Errorf("expected body %q, got %q", expected, w.Body.String())
		}
	})
}
