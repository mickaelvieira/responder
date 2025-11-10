# Responder

A simple, flexible HTTP response handler for Go that provides a clean interface for sending various types of HTTP responses.

## Features

- 🚀 Simple and intuitive API
- 📝 Multiple content type support (JSON, HTML, CSV, Plain Text)
- 🔧 Customizable error and content formatters

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

Customize how error messages are formatted:

```go
customFormatter := func(message string) any {
    return map[string]interface{}{
        "error": message,
        "timestamp": time.Now().Unix(),
        "code": "ERR_001",
    }
}

resp := responder.JSONResponder(responder.WithErrorFormatter(customFormatter))
```

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

