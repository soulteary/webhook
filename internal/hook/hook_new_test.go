package hook_test

import (
	"fmt"
	"testing"

	"github.com/adnanh/webhook/internal/hook"
)

func TestParameterNodeError_Error(t *testing.T) {
	// Test 1: Error message when e is not nil
	err := &hook.ParameterNodeError{Key: "missing_key"}
	expectedMessage := "parameter node not found: missing_key"
	if err.Error() != expectedMessage {
		t.Errorf("Expected message %q, got %q", expectedMessage, err.Error())
	}

	// Test 2: Error message when e is nil
	var nilErr *hook.ParameterNodeError
	expectedNilMessage := "<nil>"
	if nilErr.Error() != expectedNilMessage {
		t.Errorf("Expected message %q for nil error, got %q", expectedNilMessage, nilErr.Error())
	}
}

func TestIsParameterNodeError(t *testing.T) {
	// Test with ParameterNodeError type
	e := &hook.ParameterNodeError{}
	if !hook.IsParameterNodeError(e) {
		t.Error("Expected true, got false")
	}

	// Test with different error type
	notE := fmt.Errorf("some other error")
	if hook.IsParameterNodeError(notE) {
		t.Error("Expected false, got true")
	}

	// Test with nil
	if hook.IsParameterNodeError(nil) {
		t.Error("Expected false, got true")
	}
}

func TestSignatureError_Error(t *testing.T) {
	tests := []struct {
		sigError         *hook.SignatureError
		expectedErrorMsg string
	}{
		{
			sigError:         &hook.SignatureError{Signature: "signature1", EmptyPayload: false},
			expectedErrorMsg: "invalid payload signature signature1",
		},
		{
			sigError:         &hook.SignatureError{Signature: "signature2", EmptyPayload: true},
			expectedErrorMsg: "invalid payload signature signature2 on empty payload",
		},
		{
			sigError:         &hook.SignatureError{Signatures: []string{"sig1", "sig2"}, EmptyPayload: false},
			expectedErrorMsg: "invalid payload signatures [sig1 sig2]",
		},
		{
			sigError:         &hook.SignatureError{Signatures: []string{"sig3", "sig4"}, EmptyPayload: true},
			expectedErrorMsg: "invalid payload signatures [sig3 sig4] on empty payload",
		},
		{
			sigError:         nil,
			expectedErrorMsg: "<nil>",
		},
	}

	for _, tt := range tests {
		errorMsg := tt.sigError.Error()
		if errorMsg != tt.expectedErrorMsg {
			t.Errorf("Expected error message %q, got %q", tt.expectedErrorMsg, errorMsg)
		}
	}
}

func TestIsSignatureError(t *testing.T) {
	// Test with SignatureError type
	e := &hook.SignatureError{}
	if !hook.IsSignatureError(e) {
		t.Error("Expected true, got false")
	}

	// Test with different error type
	notE := fmt.Errorf("some other error")
	if hook.IsSignatureError(notE) {
		t.Error("Expected false, got true")
	}

	// Test with nil
	if hook.IsSignatureError(nil) {
		t.Error("Expected false, got true")
	}
}

func TestArgumentError_Error(t *testing.T) {
	argErr := &hook.ArgumentError{Argument: hook.Argument{Name: "arg_name"}}
	expectedMessage := "couldn't retrieve argument for {Source: Name:arg_name EnvName: Base64Decode:false}"
	if argErr.Error() != expectedMessage {
		t.Errorf("Expected message %q, got %q", expectedMessage, argErr.Error())
	}
}

func TestSourceError_Error(t *testing.T) {
	srcErr := &hook.SourceError{Argument: hook.Argument{Name: "src_name"}}
	expectedMessage := "invalid source for argument {Source: Name:src_name EnvName: Base64Decode:false}"
	if srcErr.Error() != expectedMessage {
		t.Errorf("Expected message %q, got %q", expectedMessage, srcErr.Error())
	}
}

func TestParseError_Error(t *testing.T) {
	parseErr := &hook.ParseError{Err: fmt.Errorf("specific parse error")}
	expectedMessage := "specific parse error"
	if parseErr.Error() != expectedMessage {
		t.Errorf("Expected message %q, got %q", expectedMessage, parseErr.Error())
	}
}
