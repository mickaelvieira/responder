package responder

type jsonError struct {
	Error string `json:"error"`
}

func jsonFormatter(message string) any {
	return jsonError{
		Error: message,
	}
}

// JSONResponder creates a new JSON response handler.
// The Content-Type will be set to application/json with UTF-8 charset
// and the message will be formatted as a JSON error object { "error": string }.
func JSONResponder(options ...OptionsModifier) Responder {
	options = append(options, WithErrorFormatter(jsonFormatter))
	return New(JSONContentType, options...)
}

// TextResponder creates a new text responder.
func TextResponder(options ...OptionsModifier) Responder {
	return New(TextContentType, options...)
}

// HTMLResponder creates a new HTML responder.
func HTMLResponder(options ...OptionsModifier) Responder {
	return New(HTMLContentType, options...)
}

// CSVResponder creates a new CSV responder.
func CSVResponder(options ...OptionsModifier) Responder {
	return New(CSVContentType, options...)
}
