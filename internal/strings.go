// Package internal contains utility functions
// used across the responder package that we do not want to expose publicly.
package internal

import "fmt"

// GenericErrorMessage is the default message used when an error message
const GenericErrorMessage = "an error occurred"

// MessageToString converts an error message of any type to a string.
// If the message is a string, it is returned as is.
// If the message implements fmt.Stringer, its String() method is called.
// If the message is an error, its Error() method is called.
// For any other type, a generic error message is returned.
func MessageToString(message any) string {
	switch v := message.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case error:
		return v.Error()
	default:
		return GenericErrorMessage
	}
}
