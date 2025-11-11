# Responder

A simple, flexible HTTP response handler for Go that provides a clean interface for sending various types of HTTP responses.

## Features

- Simple and intuitive API
- Multiple content type support (JSON, HTML, CSV, Plain Text, XML)
- Customizable error and content formatters

## Installation

```bash
go get github.com/mickaelvieira/responder
```

## Quick Start

```go
package main

import (
    "net/http"
    "github.com/mickaelvieira/responder"
)

func main() {
    // Create a JSON responder
    jsonResp := responder.JSONResponder()

    http.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
        // Send a successful response
        jsonResp.Send200(w, map[string]string{
            "name": "John Doe",
            "email": "john@example.com",
        })
    })

    http.HandleFunc("/api/error", func(w http.ResponseWriter, r *http.Request) {
        // Send an error response
        err := someFunction()
        if err != nil {
            jsonResp.Send400(w, err, "Invalid request")
            return
        }
    })

    http.ListenAndServe(":8080", nil)
}

func someFunction() error {
    return nil
}
```

## Available Responders

### JSON Responder

Sends responses with `application/json; charset=utf-8` content type. Error messages are automatically formatted as JSON objects.

```go
resp := responder.JSONResponder()

// Success response
resp.Send200(w, map[string]interface{}{
    "message": "Success",
    "data": []string{"item1", "item2"},
})

// Error response (automatically formatted as {"error": "Resource not found"})
resp.Send404(w, err, "Resource not found")
```

### Text Responder

Sends plain text responses with `text/plain; charset=utf-8` content type.

```go
resp := responder.TextResponder()

resp.Send200(w, "Hello, World!")
resp.Send500(w, err, "Internal server error")
```

### HTML Responder

Sends HTML responses with `text/html; charset=utf-8` content type.

```go
resp := responder.HTMLResponder()

resp.Send200(w, "<html><body><h1>Welcome</h1></body></html>")
resp.Send404(w, err, "<p>Page not found</p>")
```

### CSV Responder

Sends CSV responses with `text/csv; charset=utf-8` content type.

```go
resp := responder.CSVResponder()

csvData := "name,age,city\nJohn,30,New York\nJane,25,London"
resp.Send200(w, csvData)
```

### XML Responder

Sends XML responses with `application/xml; charset=utf-8` content type.

```go
resp := responder.XMLResponder()

xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<user>
    <name>John Doe</name>
    <email>john@example.com</email>
</user>`
resp.Send200(w, xmlData)
```

## Message Types

The error message parameter accepts `any` type, allowing you to pass various message formats:

### String Messages

```go
resp.Send400(w, err, "Invalid email format")
```

### Error Messages

```go
resp.Send500(w, err, err) // Pass the error itself as the message
```

### Custom Struct Messages

With a custom `ErrorFormatter`, you can pass structured error messages:

```go
type ErrorDetails struct {
    Code    string
    Message string
    Field   string
}

customFormatter := func(message any) any {
    switch v := message.(type) {
    case ErrorDetails:
        return fmt.Sprintf("Error %s: %s (field: %s)", v.Code, v.Message, v.Field)
    case error:
        return v.Error()
    default:
        return fmt.Sprint(message)
    }
}

resp := responder.TextResponder(responder.WithErrorFormatter(customFormatter))

// Pass a custom struct as the message
resp.Send400(w, err, ErrorDetails{
    Code:    "VALIDATION_001",
    Message: "Invalid input",
    Field:   "email",
})
```

### JSON Error Messages

For JSON responses with custom error structures:

```go
type JSONErrorResponse struct {
    Code      string   `json:"code"`
    Message   string   `json:"message"`
    Details   []string `json:"details"`
    Timestamp int64    `json:"timestamp"`
}

jsonContentFormatter := func(content any) []byte {
    data, _ := json.Marshal(content)
    return data
}

customFormatter := func(message any) any {
    switch v := message.(type) {
    case JSONErrorResponse:
        return v
    case error:
        return JSONErrorResponse{
            Code:      "ERROR",
            Message:   v.Error(),
            Timestamp: time.Now().Unix(),
        }
    default:
        return JSONErrorResponse{
            Code:      "ERROR",
            Message:   fmt.Sprint(message),
            Timestamp: time.Now().Unix(),
        }
    }
}

// Note: JSONResponder enforces {"error": "..."} format by default
// For custom JSON error structures, use New() with JSONContentType
resp := responder.New(responder.JSONContentType,
    responder.WithContentFormatter(jsonContentFormatter),
    responder.WithErrorFormatter(customFormatter),
)

resp.Send400(w, err, JSONErrorResponse{
    Code:    "VALIDATION_ERROR",
    Message: "Invalid request payload",
    Details: []string{"email is required", "password too short"},
    Timestamp: time.Now().Unix(),
})
```

## Customization

### With Logger

Add structured logging to your responder:

```go
import "log/slog"

logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
resp := responder.JSONResponder(responder.WithLogger(logger))

// Errors will be automatically logged
resp.Send500(w, err, "Database connection failed")
```

### Custom Error Formatter

Customize how error messages are formatted. The formatter receives `any` type and returns `any` type:

```go
customFormatter := func(message any) any {
    // Handle different message types
    switch v := message.(type) {
    case string:
        return map[string]interface{}{
            "error":     v,
            "timestamp": time.Now().Unix(),
            "code":      "ERR_001",
        }
    case error:
        return map[string]interface{}{
            "error":     v.Error(),
            "timestamp": time.Now().Unix(),
            "code":      "ERR_002",
        }
    default:
        return map[string]interface{}{
            "error":     fmt.Sprint(v),
            "timestamp": time.Now().Unix(),
        }
    }
}

resp := responder.JSONResponder(responder.WithErrorFormatter(customFormatter))
```

**Note:** `JSONResponder()` applies a default formatter that wraps messages in `{"error": "..."}` format. To use fully custom JSON error structures, create a responder with `responder.New()` instead.

### Custom Content Formatter

Customize how content is serialized:

```go
customContentFormatter := func(content any) []byte {
    data, _ := json.MarshalIndent(content, "", "  ")
    return data
}

resp := responder.JSONResponder(responder.WithContentFormatter(customContentFormatter))
```

### Combining Options

You can combine multiple options:

```go
resp := responder.JSONResponder(
    responder.WithLogger(logger),
    responder.WithErrorFormatter(customErrorFormatter),
    responder.WithContentFormatter(customContentFormatter),
)
```

## Advanced Usage

### Creating Custom Responders

```go
// Create a custom responder with a specific content type
customResp := responder.New("application/xml; charset=utf-8",
    responder.WithLogger(logger),
)
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

