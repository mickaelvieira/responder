package responder

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
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

		if !bytes.Equal(result, expected) {
			t.Errorf("expected %q, got %q", string(expected), string(result))
		}
	})

	t.Run("handles empty string content", func(t *testing.T) {
		input := ""
		result := contentFormatter(input)

		if len(result) != 0 {
			t.Errorf("expected empty string, got %q", string(result))
		}
	})

	t.Run("handles byte slice content", func(t *testing.T) {
		input := []byte("byte content")
		result := contentFormatter(input)

		if !bytes.Equal(result, input) {
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

		// Now it should successfully marshal via the default case
		var parsed CustomJSON
		if err := json.Unmarshal(result, &parsed); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}

		if parsed.Name != "test" || parsed.Value != 42 {
			t.Errorf("expected Name='test' and Value=42, got Name=%q Value=%d", parsed.Name, parsed.Value)
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

	t.Run("handles xml.Marshaler implementation", func(t *testing.T) {
		marshaler := customXMLMarshaler{Name: "test", Value: 42}
		result := contentFormatter(marshaler)

		// Verify it was marshaled as XML
		var parsed customXMLMarshaler
		if err := xml.Unmarshal(result, &parsed); err != nil {
			t.Fatalf("failed to unmarshal XML result: %v", err)
		}

		if parsed.Name != "test" || parsed.Value != 42 {
			t.Errorf("expected Name='test' and Value=42, got Name=%q Value=%d", parsed.Name, parsed.Value)
		}
	})

	t.Run("handles xml.Marshaler with error", func(t *testing.T) {
		marshaler := errorXMLMarshaler{}
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
		type SimpleStruct struct {
			Field string
		}

		input := SimpleStruct{Field: "value"}
		result := contentFormatter(input)

		// Should now successfully marshal via the default case
		var parsed SimpleStruct
		if err := json.Unmarshal(result, &parsed); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}

		if parsed.Field != "value" {
			t.Errorf("expected Field='value', got Field=%q", parsed.Field)
		}
	})

	t.Run("handles various types via json.Marshal", func(t *testing.T) {
		testCases := []struct {
			name     string
			input    any
			validate func(*testing.T, []byte)
		}{
			{
				name:  "int",
				input: 42,
				validate: func(t *testing.T, result []byte) {
					var num int
					if err := json.Unmarshal(result, &num); err != nil {
						t.Fatalf("failed to unmarshal: %v", err)
					}

					if num != 42 {
						t.Errorf("expected 42, got %d", num)
					}
				},
			},
			{
				name:  "float",
				input: 3.14,
				validate: func(t *testing.T, result []byte) {
					var f float64
					if err := json.Unmarshal(result, &f); err != nil {
						t.Fatalf("failed to unmarshal: %v", err)
					}

					if f != 3.14 {
						t.Errorf("expected 3.14, got %f", f)
					}
				},
			},
			{
				name:  "bool",
				input: true,
				validate: func(t *testing.T, result []byte) {
					var b bool
					if err := json.Unmarshal(result, &b); err != nil {
						t.Fatalf("failed to unmarshal: %v", err)
					}

					if !b {
						t.Errorf("expected true, got false")
					}
				},
			},
			{
				name:  "slice",
				input: []string{"a", "b", "c"},
				validate: func(t *testing.T, result []byte) {
					var slice []string
					if err := json.Unmarshal(result, &slice); err != nil {
						t.Fatalf("failed to unmarshal: %v", err)
					}

					if len(slice) != 3 || slice[0] != "a" || slice[1] != "b" || slice[2] != "c" {
						t.Errorf("unexpected slice content: %v", slice)
					}
				},
			},
			{
				name:  "map",
				input: map[string]int{"key": 1, "another": 2},
				validate: func(t *testing.T, result []byte) {
					var m map[string]int
					if err := json.Unmarshal(result, &m); err != nil {
						t.Fatalf("failed to unmarshal: %v", err)
					}

					if m["key"] != 1 || m["another"] != 2 {
						t.Errorf("unexpected map content: %v", m)
					}
				},
			},
			{
				name:  "struct",
				input: struct{ X int }{X: 10},
				validate: func(t *testing.T, result []byte) {
					var s struct{ X int }
					if err := json.Unmarshal(result, &s); err != nil {
						t.Fatalf("failed to unmarshal: %v", err)
					}

					if s.X != 10 {
						t.Errorf("expected X=10, got X=%d", s.X)
					}
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := contentFormatter(tc.input)
				tc.validate(t, result)
			})
		}
	})

	t.Run("handles unmarshalable types", func(t *testing.T) {
		// Channels, functions, and complex types can't be marshaled
		testCases := []struct {
			name  string
			input any
		}{
			{"channel", make(chan int)},
			{"function", func() {}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := contentFormatter(tc.input)

				expectedPrefix := "received invalid content"
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

type customXMLMarshaler struct {
	XMLName xml.Name `xml:"custom"`
	Name    string   `xml:"name"`
	Value   int      `xml:"value"`
}

func (c customXMLMarshaler) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	// Custom XML marshaling logic
	start.Name.Local = "custom"
	if err := e.EncodeToken(start); err != nil {
		return err
	}

	if err := e.EncodeElement(c.Name, xml.StartElement{Name: xml.Name{Local: "name"}}); err != nil {
		return err
	}

	if err := e.EncodeElement(c.Value, xml.StartElement{Name: xml.Name{Local: "value"}}); err != nil {
		return err
	}

	return e.EncodeToken(start.End())
}

type errorXMLMarshaler struct{}

func (e errorXMLMarshaler) MarshalXML(enc *xml.Encoder, start xml.StartElement) error {
	return fmt.Errorf("intentional XML marshal error")
}
