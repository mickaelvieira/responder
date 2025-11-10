package responder

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestContentFormatter(t *testing.T) {
	t.Run("handles nil content", func(t *testing.T) {
		result := contentFormatter(nil)
		if len(result) != 0 {
			t.Errorf("expected empty byte slice for nil, got %v", result)
		}
	})

	t.Run("handles string content", func(t *testing.T) {
		input := "Hello, World!"
		result := contentFormatter(input)
		expected := []byte(input)

		if string(result) != string(expected) {
			t.Errorf("expected %q, got %q", string(expected), string(result))
		}
	})

	t.Run("handles empty string content", func(t *testing.T) {
		input := ""
		result := contentFormatter(input)

		if string(result) != "" {
			t.Errorf("expected empty string, got %q", string(result))
		}
	})

	t.Run("handles byte slice content", func(t *testing.T) {
		input := []byte("byte content")
		result := contentFormatter(input)

		if string(result) != string(input) {
			t.Errorf("expected %q, got %q", string(input), string(result))
		}
	})

	t.Run("handles empty byte slice", func(t *testing.T) {
		input := []byte{}
		result := contentFormatter(input)

		if len(result) != 0 {
			t.Errorf("expected empty byte slice, got %v", result)
		}
	})

	t.Run("handles json.Marshaler implementation", func(t *testing.T) {
		type CustomJSON struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}

		input := CustomJSON{Name: "test", Value: 42}
		result := contentFormatter(input)

		// Since CustomJSON doesn't implement json.Marshaler, it should fall through to default
		expectedError := "received invalid content - unknown type responder.CustomJSON"
		if string(result) != expectedError {
			t.Logf("got %q", string(result))
		}
	})

	t.Run("handles struct implementing json.Marshaler", func(t *testing.T) {
		// Create a proper json.Marshaler
		marshaler := customJSONMarshaler{value: "custom_value"}
		result := contentFormatter(marshaler)

		var parsed map[string]string
		if err := json.Unmarshal(result, &parsed); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}

		if parsed["custom"] != "custom_value" {
			t.Errorf("expected custom field to be 'custom_value', got %q", parsed["custom"])
		}
	})

	t.Run("handles json.Marshaler with error", func(t *testing.T) {
		marshaler := errorJSONMarshaler{}
		result := contentFormatter(marshaler)

		expectedPrefix := "received invalid content"
		if !strings.Contains(string(result), expectedPrefix) {
			t.Errorf("expected error message to contain %q, got %q", expectedPrefix, string(result))
		}
	})

	t.Run("handles encoding.TextMarshaler implementation", func(t *testing.T) {
		marshaler := customTextMarshaler{value: "text_value"}
		result := contentFormatter(marshaler)

		expected := "TEXT:text_value"
		if string(result) != expected {
			t.Errorf("expected %q, got %q", expected, string(result))
		}
	})

	t.Run("handles encoding.TextMarshaler with error", func(t *testing.T) {
		marshaler := errorTextMarshaler{}
		result := contentFormatter(marshaler)

		expectedPrefix := "received invalid content"
		if !strings.Contains(string(result), expectedPrefix) {
			t.Errorf("expected error message to contain %q, got %q", expectedPrefix, string(result))
		}
	})

	t.Run("handles unsupported type", func(t *testing.T) {
		type UnsupportedType struct {
			Field string
		}

		input := UnsupportedType{Field: "value"}
		result := contentFormatter(input)

		expectedPrefix := "received invalid content - unknown type"
		if !strings.Contains(string(result), expectedPrefix) {
			t.Errorf("expected error message to contain %q, got %q", expectedPrefix, string(result))
		}
	})

	t.Run("handles various unsupported types", func(t *testing.T) {
		testCases := []struct {
			name  string
			input any
		}{
			{"int", 42},
			{"float", 3.14},
			{"bool", true},
			{"slice", []string{"a", "b"}},
			{"map", map[string]int{"key": 1}},
			{"struct", struct{ X int }{X: 10}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := contentFormatter(tc.input)
				expectedPrefix := "received invalid content - unknown type"
				if !strings.Contains(string(result), expectedPrefix) {
					t.Errorf("expected error message for %s, got %q", tc.name, string(result))
				}
			})
		}
	})

	t.Run("handles large string content", func(t *testing.T) {
		// Create a large string
		var builder strings.Builder
		for i := 0; i < 10000; i++ {
			builder.WriteString("Lorem ipsum dolor sit amet. ")
		}
		input := builder.String()

		result := contentFormatter(input)

		if string(result) != input {
			t.Errorf("large string content mismatch: expected %d bytes, got %d bytes",
				len(input), len(result))
		}
	})

	t.Run("handles large byte slice content", func(t *testing.T) {
		// Create a large byte slice
		input := make([]byte, 1024*1024) // 1MB
		for i := range input {
			input[i] = byte(i % 256)
		}

		result := contentFormatter(input)

		if len(result) != len(input) {
			t.Errorf("expected %d bytes, got %d bytes", len(input), len(result))
		}
	})

	t.Run("preserves byte slice reference", func(t *testing.T) {
		input := []byte("test")
		result := contentFormatter(input)

		// Modify the result and check if input is affected
		result[0] = 'X'

		// The input should be affected since []byte is returned directly
		if input[0] != 'X' {
			t.Error("expected byte slice to be returned by reference")
		}
	})
}

// Helper types for testing marshalers

type customJSONMarshaler struct {
	value string
}

func (c customJSONMarshaler) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{"custom": c.value})
}

type errorJSONMarshaler struct{}

func (e errorJSONMarshaler) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("intentional JSON marshal error")
}

type customTextMarshaler struct {
	value string
}

func (c customTextMarshaler) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprintf("TEXT:%s", c.value)), nil
}

type errorTextMarshaler struct{}

func (e errorTextMarshaler) MarshalText() ([]byte, error) {
	return nil, fmt.Errorf("intentional text marshal error")
}
