package hook

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestGetParameter(t *testing.T) {
	tests := []struct {
		name   string
		key    string
		val    interface{}
		expect interface{}
		ok     bool
	}{
		// Success cases
		{"simple map key", "a", map[string]interface{}{"a": "1"}, "1", true},
		{"dotted key in map", "a.b", map[string]interface{}{"a.b": "1"}, "1", true},
		{"nested map", "a.c", map[string]interface{}{"a": map[string]interface{}{"c": 2}}, 2, true},
		{"numeric key in map", "a.1", map[string]interface{}{"a": map[string]interface{}{"1": 3}}, 3, true},
		{"slice index", "a.1", map[string]interface{}{"a": []interface{}{"a", "b"}}, "b", true},
		{"direct slice index", "0", []interface{}{"a", "b"}, "a", true},

		// Failure cases
		{"missing key", "z", map[string]interface{}{"a": "1"}, nil, false},
		{"missing nested key", "a.z", map[string]interface{}{"a": map[string]interface{}{"b": 2}}, nil, false},
		{"missing parent key", "z.b", map[string]interface{}{"a": map[string]interface{}{"z": 2}}, nil, false},
		{"out of range slice index", "a.2", map[string]interface{}{"a": []interface{}{"a", "b"}}, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := GetParameter(tt.key, tt.val)
			if (err == nil) != tt.ok {
				t.Errorf("unexpected result: got error=%v, want ok=%v", err, tt.ok)
			}

			if !reflect.DeepEqual(res, tt.expect) {
				t.Errorf("unexpected value: got %#v, want %#v", res, tt.expect)
			}
		})
	}
}

var checkPayloadSignatureTests = []struct {
	payload   []byte
	secret    string
	signature string
	mac       string
	ok        bool
}{
	{[]byte(`{"a": "z"}`), "secret", "b17e04cbb22afa8ffbff8796fc1894ed27badd9e", "b17e04cbb22afa8ffbff8796fc1894ed27badd9e", true},
	{[]byte(`{"a": "z"}`), "secret", "sha1=b17e04cbb22afa8ffbff8796fc1894ed27badd9e", "b17e04cbb22afa8ffbff8796fc1894ed27badd9e", true},
	{[]byte(`{"a": "z"}`), "secret", "sha1=XXXe04cbb22afa8ffbff8796fc1894ed27badd9e,sha1=b17e04cbb22afa8ffbff8796fc1894ed27badd9e", "b17e04cbb22afa8ffbff8796fc1894ed27badd9e", true},
	{[]byte(``), "secret", "25af6174a0fcecc4d346680a72b7ce644b9a88e8", "25af6174a0fcecc4d346680a72b7ce644b9a88e8", true},
	// failures
	{[]byte(`{"a": "z"}`), "secret", "XXXe04cbb22afa8ffbff8796fc1894ed27badd9e", "b17e04cbb22afa8ffbff8796fc1894ed27badd9e", false},
	{[]byte(`{"a": "z"}`), "secret", "sha1=XXXe04cbb22afa8ffbff8796fc1894ed27badd9e", "b17e04cbb22afa8ffbff8796fc1894ed27badd9e", false},
	{[]byte(`{"a": "z"}`), "secret", "sha1=XXXe04cbb22afa8ffbff8796fc1894ed27badd9e,sha1=XXXe04cbb22afa8ffbff8796fc1894ed27badd9e", "b17e04cbb22afa8ffbff8796fc1894ed27badd9e", false},
	{[]byte(`{"a": "z"}`), "secreX", "b17e04cbb22afa8ffbff8796fc1894ed27badd9e", "900225703e9342328db7307692736e2f7cc7b36e", false},
	{[]byte(`{"a": "z"}`), "", "b17e04cbb22afa8ffbff8796fc1894ed27badd9e", "", false},
	{[]byte(``), "secret", "XXXf6174a0fcecc4d346680a72b7ce644b9a88e8", "25af6174a0fcecc4d346680a72b7ce644b9a88e8", false},
}

func TestCheckPayloadSignature(t *testing.T) {
	for i, tt := range checkPayloadSignatureTests {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			mac, err := CheckPayloadSignature(tt.payload, tt.secret, tt.signature)
			if (err == nil) != tt.ok || mac != tt.mac {
				t.Errorf("failed to check payload signature:\nexpected {mac:%#v, ok:%#v},\ngot {mac:%#v, ok:%#v}", tt.mac, tt.ok, mac, (err == nil))
			}

			if err != nil && tt.mac != "" && strings.Contains(err.Error(), tt.mac) {
				t.Errorf("error message should not disclose expected mac: %s", err)
			}
		})
	}
}

var checkPayloadSignature256Tests = []struct {
	payload   []byte
	secret    string
	signature string
	mac       string
	ok        bool
}{
	{[]byte(`{"a": "z"}`), "secret", "f417af3a21bd70379b5796d5f013915e7029f62c580fb0f500f59a35a6f04c89", "f417af3a21bd70379b5796d5f013915e7029f62c580fb0f500f59a35a6f04c89", true},
	{[]byte(`{"a": "z"}`), "secret", "sha256=f417af3a21bd70379b5796d5f013915e7029f62c580fb0f500f59a35a6f04c89", "f417af3a21bd70379b5796d5f013915e7029f62c580fb0f500f59a35a6f04c89", true},
	{[]byte(`{"a": "z"}`), "secret", "sha256=XXX7af3a21bd70379b5796d5f013915e7029f62c580fb0f500f59a35a6f04c89,sha256=f417af3a21bd70379b5796d5f013915e7029f62c580fb0f500f59a35a6f04c89", "f417af3a21bd70379b5796d5f013915e7029f62c580fb0f500f59a35a6f04c89", true},
	{[]byte(``), "secret", "f9e66e179b6747ae54108f82f8ade8b3c25d76fd30afde6c395822c530196169", "f9e66e179b6747ae54108f82f8ade8b3c25d76fd30afde6c395822c530196169", true},
	// failures
	{[]byte(`{"a": "z"}`), "secret", "XXX7af3a21bd70379b5796d5f013915e7029f62c580fb0f500f59a35a6f04c89", "f417af3a21bd70379b5796d5f013915e7029f62c580fb0f500f59a35a6f04c89", false},
	{[]byte(`{"a": "z"}`), "secret", "sha256=XXX7af3a21bd70379b5796d5f013915e7029f62c580fb0f500f59a35a6f04c89", "f417af3a21bd70379b5796d5f013915e7029f62c580fb0f500f59a35a6f04c89", false},
	{[]byte(`{"a": "z"}`), "secret", "sha256=XXX7af3a21bd70379b5796d5f013915e7029f62c580fb0f500f59a35a6f04c89,sha256=XXX7af3a21bd70379b5796d5f013915e7029f62c580fb0f500f59a35a6f04c89", "f417af3a21bd70379b5796d5f013915e7029f62c580fb0f500f59a35a6f04c89", false},
	{[]byte(`{"a": "z"}`), "", "XXX7af3a21bd70379b5796d5f013915e7029f62c580fb0f500f59a35a6f04c89", "", false},
	{[]byte(``), "secret", "XXX66e179b6747ae54108f82f8ade8b3c25d76fd30afde6c395822c530196169", "f9e66e179b6747ae54108f82f8ade8b3c25d76fd30afde6c395822c530196169", false},
}

func TestCheckPayloadSignature256(t *testing.T) {
	for i, tt := range checkPayloadSignature256Tests {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			mac, err := CheckPayloadSignature256(tt.payload, tt.secret, tt.signature)
			if (err == nil) != tt.ok || mac != tt.mac {
				t.Errorf("failed to check payload signature:\nexpected {mac:%#v, ok:%#v},\ngot {mac:%#v, ok:%#v}", tt.mac, tt.ok, mac, (err == nil))
			}

			if err != nil && tt.mac != "" && strings.Contains(err.Error(), tt.mac) {
				t.Errorf("error message should not disclose expected mac: %s", err)
			}
		})
	}
}

var checkPayloadSignature512Tests = []struct {
	payload   []byte
	secret    string
	signature string
	mac       string
	ok        bool
}{
	{[]byte(`{"a": "z"}`), "secret", "4ab17cc8ec668ead8bf498f87f8f32848c04d5ca3c9bcfcd3db9363f0deb44e580b329502a7fdff633d4d8fca301cc5c94a55a2fec458c675fb0ff2655898324", "4ab17cc8ec668ead8bf498f87f8f32848c04d5ca3c9bcfcd3db9363f0deb44e580b329502a7fdff633d4d8fca301cc5c94a55a2fec458c675fb0ff2655898324", true},
	{[]byte(`{"a": "z"}`), "secret", "sha512=4ab17cc8ec668ead8bf498f87f8f32848c04d5ca3c9bcfcd3db9363f0deb44e580b329502a7fdff633d4d8fca301cc5c94a55a2fec458c675fb0ff2655898324", "4ab17cc8ec668ead8bf498f87f8f32848c04d5ca3c9bcfcd3db9363f0deb44e580b329502a7fdff633d4d8fca301cc5c94a55a2fec458c675fb0ff2655898324", true},
	{[]byte(``), "secret", "b0e9650c5faf9cd8ae02276671545424104589b3656731ec193b25d01b07561c27637c2d4d68389d6cf5007a8632c26ec89ba80a01c77a6cdd389ec28db43901", "b0e9650c5faf9cd8ae02276671545424104589b3656731ec193b25d01b07561c27637c2d4d68389d6cf5007a8632c26ec89ba80a01c77a6cdd389ec28db43901", true},
	// failures
	{[]byte(`{"a": "z"}`), "secret", "74a0081f5b5988f4f3e8b8dd34dadc6291611f2e6260635a7e1535f8e95edb97ff520ba8b152e8ca5760ac42639854f3242e29efc81be73a8bf52d474d31ffea", "4ab17cc8ec668ead8bf498f87f8f32848c04d5ca3c9bcfcd3db9363f0deb44e580b329502a7fdff633d4d8fca301cc5c94a55a2fec458c675fb0ff2655898324", false},
	{[]byte(`{"a": "z"}`), "", "74a0081f5b5988f4f3e8b8dd34dadc6291611f2e6260635a7e1535f8e95edb97ff520ba8b152e8ca5760ac42639854f3242e29efc81be73a8bf52d474d31ffea", "", false},
	{[]byte(``), "secret", "XXX9650c5faf9cd8ae02276671545424104589b3656731ec193b25d01b07561c27637c2d4d68389d6cf5007a8632c26ec89ba80a01c77a6cdd389ec28db43901", "b0e9650c5faf9cd8ae02276671545424104589b3656731ec193b25d01b07561c27637c2d4d68389d6cf5007a8632c26ec89ba80a01c77a6cdd389ec28db43901", false},
}

func TestCheckPayloadSignature512(t *testing.T) {
	for i, tt := range checkPayloadSignature512Tests {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			mac, err := CheckPayloadSignature512(tt.payload, tt.secret, tt.signature)
			if (err == nil) != tt.ok || mac != tt.mac {
				t.Errorf("failed to check payload signature:\nexpected {mac:%#v, ok:%#v},\ngot {mac:%#v, ok:%#v}", tt.mac, tt.ok, mac, (err == nil))
			}

			if err != nil && tt.mac != "" && strings.Contains(err.Error(), tt.mac) {
				t.Errorf("error message should not disclose expected mac: %s", err)
			}
		})
	}
}

var checkScalrSignatureTests = []struct {
	description       string
	headers           map[string]interface{}
	body              []byte
	secret            string
	expectedSignature string
	ok                bool
}{
	{
		"Valid signature",
		map[string]interface{}{"Date": "Thu 07 Sep 2017 06:30:04 UTC", "X-Signature": "48e395e38ac48988929167df531eb2da00063a7d"},
		[]byte(`{"a": "b"}`), "bilFGi4ZVZUdG+C6r0NIM9tuRq6PaG33R3eBUVhLwMAErGBaazvXe4Gq2DcJs5q+",
		"48e395e38ac48988929167df531eb2da00063a7d", true,
	},
	{
		"Wrong signature",
		map[string]interface{}{"Date": "Thu 07 Sep 2017 06:30:04 UTC", "X-Signature": "999395e38ac48988929167df531eb2da00063a7d"},
		[]byte(`{"a": "b"}`), "bilFGi4ZVZUdG+C6r0NIM9tuRq6PaG33R3eBUVhLwMAErGBaazvXe4Gq2DcJs5q+",
		"48e395e38ac48988929167df531eb2da00063a7d", false,
	},
	{
		"Missing Date header",
		map[string]interface{}{"X-Signature": "999395e38ac48988929167df531eb2da00063a7d"},
		[]byte(`{"a": "b"}`), "bilFGi4ZVZUdG+C6r0NIM9tuRq6PaG33R3eBUVhLwMAErGBaazvXe4Gq2DcJs5q+",
		"48e395e38ac48988929167df531eb2da00063a7d", false,
	},
	{
		"Missing X-Signature header",
		map[string]interface{}{"Date": "Thu 07 Sep 2017 06:30:04 UTC"},
		[]byte(`{"a": "b"}`), "bilFGi4ZVZUdG+C6r0NIM9tuRq6PaG33R3eBUVhLwMAErGBaazvXe4Gq2DcJs5q+",
		"48e395e38ac48988929167df531eb2da00063a7d", false,
	},
	{
		"Missing signing key",
		map[string]interface{}{"Date": "Thu 07 Sep 2017 06:30:04 UTC", "X-Signature": "48e395e38ac48988929167df531eb2da00063a7d"},
		[]byte(`{"a": "b"}`), "",
		"48e395e38ac48988929167df531eb2da00063a7d", false,
	},
}

func TestCheckScalrSignature(t *testing.T) {
	for _, testCase := range checkScalrSignatureTests {
		t.Run(testCase.description, func(t *testing.T) {
			r := &Request{
				Headers: testCase.headers,
				Body:    testCase.body,
			}
			valid, err := CheckScalrSignature(r, testCase.secret, false)
			if valid != testCase.ok {
				t.Errorf("unexpected result: got ok=%v, want ok=%v", valid, testCase.ok)
			}

			if err != nil && testCase.secret != "" && strings.Contains(err.Error(), testCase.expectedSignature) {
				t.Errorf("error message should not disclose expected mac: %s", err)
			}
		})
	}
}

var checkMSTeamsSignatureTests = []struct {
	description       string
	headers           map[string]interface{}
	body              []byte
	secret            string
	expectedSignature string
	ok                bool
}{
	{
		"Valid signature",
		map[string]interface{}{"Authorization": "HMAC gpjdTlOlaReTBLRFdwqdXhLqG7hFXVYTBorGDpaW5UE="},
		[]byte(`{"a": "b"}`), "bmV2ZXJnb25uYWdpdmV5b3V1cA==",
		"gpjdTlOlaReTBLRFdwqdXhLqG7hFXVYTBorGDpaW5UE=", true,
	},
	{
		"Wrong signature",
		map[string]interface{}{"Authorization": "HMAC 1337TlOlaReTBLRFdwqdXhLqG7hFXVYTBorGDpaW5UE="},
		[]byte(`{"a": "b"}`), "bmV2ZXJnb25uYWdpdmV5b3V1cA==",
		"gpjdTlOlaReTBLRFdwqdXhLqG7hFXVYTBorGDpaW5UE=", false,
	},
	{
		"Missing Authorization header",
		map[string]interface{}{"Different-Header": "HMAC wrong"},
		[]byte(`{"a": "b"}`), "bmV2ZXJnb25uYWdpdmV5b3V1cA==",
		"gpjdTlOlaReTBLRFdwqdXhLqG7hFXVYTBorGDpaW5UE=", false,
	},
	{
		"Malformed Authorization header",
		map[string]interface{}{"Authorization": "HMAC 123---"},
		[]byte(`{"a": "b"}`), "bmV2ZXJnb25uYWdpdmV5b3V1cA==",
		"gpjdTlOlaReTBLRFdwqdXhLqG7hFXVYTBorGDpaW5UE=", false,
	},
	{
		"Missing signing key",
		map[string]interface{}{"Authorization": "HMAC gpjdTlOlaReTBLRFdwqdXhLqG7hFXVYTBorGDpaW5UE="},
		[]byte(`{"a": "b"}`), "",
		"gpjdTlOlaReTBLRFdwqdXhLqG7hFXVYTBorGDpaW5UE=", false,
	},
	{
		"Malformed signing key",
		map[string]interface{}{"Authorization": "HMAC gpjdTlOlaReTBLRFdwqdXhLqG7hFXVYTBorGDpaW5UE="},
		[]byte(`{"a": "b"}`), "---2ZXJnb25uYWdpdmV5b3V1cA==",
		"gpjdTlOlaReTBLRFdwqdXhLqG7hFXVYTBorGDpaW5UE=", false,
	},
}

func TestCheckMSTeamsSignature(t *testing.T) {
	for _, testCase := range checkMSTeamsSignatureTests {
		t.Run(testCase.description, func(t *testing.T) {
			r := &Request{
				Headers: testCase.headers,
				Body:    testCase.body,
			}
			valid, err := CheckMSTeamsSignature(r, testCase.secret)
			if valid != testCase.ok {
				t.Errorf("unexpected result: got ok=%v, want ok=%v", valid, testCase.ok)
			}

			if err != nil && testCase.secret != "" && strings.Contains(err.Error(), testCase.expectedSignature) {
				t.Errorf("error message should not disclose expected mac: %s", err)
			}
		})
	}
}

var checkIPWhitelistTests = []struct {
	addr    string
	ipRange string
	expect  bool
	ok      bool
}{
	{"[ 10.0.0.1:1234 ] ", "  10.0.0.1 ", true, true},
	{"[ 10.0.0.1:1234 ] ", "  10.0.0.0 ", false, true},
	{"[ 10.0.0.1:1234 ] ", "  10.0.0.1 10.0.0.1 ", true, true},
	{"[ 10.0.0.1:1234 ] ", "  10.0.0.0/31 ", true, true},
	{" [2001:db8:1:2::1:1234] ", "  2001:db8:1::/48 ", true, true},
	{" [2001:db8:1:2::1:1234] ", "  2001:db8:1::/48 2001:db8:1::/64", true, true},
	{" [2001:db8:1:2::1:1234] ", "  2001:db8:1::/64 ", false, true},
}

func TestCheckIPWhitelist(t *testing.T) {
	for i, tt := range checkIPWhitelistTests {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			result, err := CheckIPWhitelist(tt.addr, tt.ipRange)
			if (err == nil) != tt.ok || result != tt.expect {
				t.Errorf("unexpected result:\nwant {expect:%#v, ok:%#v},\ngot {result:%#v, ok:%#v}", tt.expect, tt.ok, result, err == nil)
			}
		})
	}
}

var extractParameterTests = []struct {
	s      string
	params interface{}
	value  string
	ok     bool
}{
	{"a", map[string]interface{}{"a": "z"}, "z", true},
	{"a.b", map[string]interface{}{"a": map[string]interface{}{"b": "z"}}, "z", true},
	{"a.b.c", map[string]interface{}{"a": map[string]interface{}{"b": map[string]interface{}{"c": "z"}}}, "z", true},
	{"a.b.0", map[string]interface{}{"a": map[string]interface{}{"b": []interface{}{"x", "y", "z"}}}, "x", true},
	{"a.1.b", map[string]interface{}{"a": []interface{}{map[string]interface{}{"b": "y"}, map[string]interface{}{"b": "z"}}}, "z", true},
	{"a.1.b.c", map[string]interface{}{"a": []interface{}{map[string]interface{}{"b": map[string]interface{}{"c": "y"}}, map[string]interface{}{"b": map[string]interface{}{"c": "z"}}}}, "z", true},
	{"b", map[string]interface{}{"b": map[string]interface{}{"z": 1}}, `{"z":1}`, true},
	{"c", map[string]interface{}{"c": []interface{}{"y", "z"}}, `["y","z"]`, true},
	{"d", map[string]interface{}{"d": [2]interface{}{"y", "z"}}, `["y","z"]`, true},
	{"a.b.c.1", map[string]interface{}{"a": map[string]interface{}{"b": map[string]interface{}{"c.1": "z"}}}, "z", true},
	{"a.b.1.c", map[string]interface{}{"a": map[string]interface{}{"b.1": map[string]interface{}{"c": "z"}}}, "z", true},
	// failures
	{"check_nil", nil, "", false},
	{"a.X", map[string]interface{}{"a": map[string]interface{}{"b": "z"}}, "", false},                                                      // non-existent parameter reference
	{"a.X.c", map[string]interface{}{"a": []interface{}{map[string]interface{}{"b": "y"}, map[string]interface{}{"b": "z"}}}, "", false},   // non-integer slice index
	{"a.-1.b", map[string]interface{}{"a": []interface{}{map[string]interface{}{"b": "y"}, map[string]interface{}{"b": "z"}}}, "", false},  // negative slice index
	{"a.500.b", map[string]interface{}{"a": map[string]interface{}{"b": "z"}}, "", false},                                                  // non-existent slice
	{"a.501.b", map[string]interface{}{"a": []interface{}{map[string]interface{}{"b": "y"}, map[string]interface{}{"b": "z"}}}, "", false}, // non-existent slice index
	{"a.502.b", map[string]interface{}{"a": []interface{}{}}, "", false},                                                                   // non-existent slice index
	{"a.b.503", map[string]interface{}{"a": map[string]interface{}{"b": []interface{}{"x", "y", "z"}}}, "", false},                         // trailing, non-existent slice index
	{"a.b", interface{}("a"), "", false}, // non-map, non-slice input
}

func TestExtractParameter(t *testing.T) {
	for _, tt := range extractParameterTests {
		value, err := ExtractParameterAsString(tt.s, tt.params)
		if (err == nil) != tt.ok || value != tt.value {
			t.Errorf("failed to extract parameter %q:\nexpected {value:%#v, ok:%#v},\ngot {value:%#v, err:%v}", tt.s, tt.value, tt.ok, value, err)
		}
	}
}

var argumentGetTests = []struct {
	source, name            string
	headers, query, payload map[string]interface{}
	request                 *http.Request
	value                   string
	ok                      bool
}{
	{"header", "a", map[string]interface{}{"A": "z"}, nil, nil, nil, "z", true},
	{"url", "a", nil, map[string]interface{}{"a": "z"}, nil, nil, "z", true},
	{"payload", "a", nil, nil, map[string]interface{}{"a": "z"}, nil, "z", true},
	{"request", "METHOD", nil, nil, map[string]interface{}{"a": "z"}, &http.Request{Method: "POST", RemoteAddr: "127.0.0.1:1234"}, "POST", true},
	{"request", "remote-addr", nil, nil, map[string]interface{}{"a": "z"}, &http.Request{Method: "POST", RemoteAddr: "127.0.0.1:1234"}, "127.0.0.1:1234", true},
	{"string", "a", nil, nil, map[string]interface{}{"a": "z"}, nil, "a", true},
	// failures
	{"header", "a", nil, map[string]interface{}{"a": "z"}, map[string]interface{}{"a": "z"}, nil, "", false},  // nil headers
	{"url", "a", map[string]interface{}{"A": "z"}, nil, map[string]interface{}{"a": "z"}, nil, "", false},     // nil query
	{"payload", "a", map[string]interface{}{"A": "z"}, map[string]interface{}{"a": "z"}, nil, nil, "", false}, // nil payload
	{"foo", "a", map[string]interface{}{"A": "z"}, nil, nil, nil, "", false},                                  // invalid source
}

func TestArgumentGet(t *testing.T) {
	for _, tt := range argumentGetTests {
		a := Argument{tt.source, tt.name, "", false}
		r := &Request{
			Headers:    tt.headers,
			Query:      tt.query,
			Payload:    tt.payload,
			RawRequest: tt.request,
		}
		value, err := a.Get(r)
		if (err == nil) != tt.ok || value != tt.value {
			t.Errorf("failed to get {%q, %q}:\nexpected {value:%#v, ok:%#v},\ngot {value:%#v, err:%v}", tt.source, tt.name, tt.value, tt.ok, value, err)
		}
	}
}

var hookParseJSONParametersTests = []struct {
	params                     []Argument
	headers, query, payload    map[string]interface{}
	rheaders, rquery, rpayload map[string]interface{}
	ok                         bool
}{
	{[]Argument{{"header", "a", "", false}}, map[string]interface{}{"A": `{"b": "y"}`}, nil, nil, map[string]interface{}{"A": map[string]interface{}{"b": "y"}}, nil, nil, true},
	{[]Argument{{"url", "a", "", false}}, nil, map[string]interface{}{"a": `{"b": "y"}`}, nil, nil, map[string]interface{}{"a": map[string]interface{}{"b": "y"}}, nil, true},
	{[]Argument{{"payload", "a", "", false}}, nil, nil, map[string]interface{}{"a": `{"b": "y"}`}, nil, nil, map[string]interface{}{"a": map[string]interface{}{"b": "y"}}, true},
	{[]Argument{{"header", "z", "", false}}, map[string]interface{}{"Z": `{}`}, nil, nil, map[string]interface{}{"Z": map[string]interface{}{}}, nil, nil, true},
	// failures
	{[]Argument{{"header", "z", "", false}}, map[string]interface{}{"Z": ``}, nil, nil, map[string]interface{}{"Z": ``}, nil, nil, false},     // empty string
	{[]Argument{{"header", "y", "", false}}, map[string]interface{}{"X": `{}`}, nil, nil, map[string]interface{}{"X": `{}`}, nil, nil, false}, // missing parameter
	{[]Argument{{"string", "z", "", false}}, map[string]interface{}{"Z": ``}, nil, nil, map[string]interface{}{"Z": ``}, nil, nil, false},     // invalid argument source
}

func TestHookParseJSONParameters(t *testing.T) {
	for _, tt := range hookParseJSONParametersTests {
		h := &Hook{JSONStringParameters: tt.params}
		r := &Request{
			Headers: tt.headers,
			Query:   tt.query,
			Payload: tt.payload,
		}
		err := h.ParseJSONParameters(r)
		if (err == nil) != tt.ok || !reflect.DeepEqual(tt.headers, tt.rheaders) {
			t.Errorf("failed to parse %v:\nexpected %#v, ok: %v\ngot %#v, ok: %v", tt.params, tt.rheaders, tt.ok, tt.headers, (err == nil))
		}
	}
}

var hookExtractCommandArgumentsTests = []struct {
	exec                    string
	args                    []Argument
	headers, query, payload map[string]interface{}
	value                   []string
	ok                      bool
}{
	{"test", []Argument{{"header", "a", "", false}}, map[string]interface{}{"A": "z"}, nil, nil, []string{"test", "z"}, true},
	// failures
	{"fail", []Argument{{"payload", "a", "", false}}, map[string]interface{}{"A": "z"}, nil, nil, []string{"fail", ""}, false},
}

func TestHookExtractCommandArguments(t *testing.T) {
	for _, tt := range hookExtractCommandArgumentsTests {
		h := &Hook{ExecuteCommand: tt.exec, PassArgumentsToCommand: tt.args}
		r := &Request{
			Headers: tt.headers,
			Query:   tt.query,
			Payload: tt.payload,
		}
		value, err := h.ExtractCommandArguments(r)
		if (err == nil) != tt.ok || !reflect.DeepEqual(value, tt.value) {
			t.Errorf("failed to extract args {cmd=%q, args=%v}:\nexpected %#v, ok: %v\ngot %#v, ok: %v", tt.exec, tt.args, tt.value, tt.ok, value, (err == nil))
		}
	}
}

// Here we test the extraction of env variables when the user defined a hook
// with the "pass-environment-to-command" directive
// we test both cases where the name of the data is used as the name of the
// env key & the case where the hook definition sets the env var name to a
// fixed value using the envname construct like so::
//
//	[
//	  {
//	    "id": "push",
//	    "execute-command": "bb2mm",
//	    "command-working-directory": "/tmp",
//	    "pass-environment-to-command":
//	    [
//	      {
//	        "source": "entire-payload",
//	        "envname": "PAYLOAD"
//	      },
//	    ]
//	  }
//	]
var hookExtractCommandArgumentsForEnvTests = []struct {
	exec                    string
	args                    []Argument
	headers, query, payload map[string]interface{}
	value                   []string
	ok                      bool
}{
	// successes
	{
		"test",
		[]Argument{{"header", "a", "", false}},
		map[string]interface{}{"A": "z"},
		nil, nil,
		[]string{"HOOK_a=z"},
		true,
	},
	{
		"test",
		[]Argument{{"header", "a", "MYKEY", false}},
		map[string]interface{}{"A": "z"},
		nil, nil,
		[]string{"MYKEY=z"},
		true,
	},
	// failures
	{
		"fail",
		[]Argument{{"payload", "a", "", false}},
		map[string]interface{}{"A": "z"},
		nil, nil,
		[]string{},
		false,
	},
}

func TestHookExtractCommandArgumentsForEnv(t *testing.T) {
	for _, tt := range hookExtractCommandArgumentsForEnvTests {
		h := &Hook{ExecuteCommand: tt.exec, PassEnvironmentToCommand: tt.args}
		r := &Request{
			Headers: tt.headers,
			Query:   tt.query,
			Payload: tt.payload,
		}
		value, err := h.ExtractCommandArgumentsForEnv(r)
		if (err == nil) != tt.ok || !reflect.DeepEqual(value, tt.value) {
			t.Errorf("failed to extract args for env {cmd=%q, args=%v}:\nexpected %#v, ok: %v\ngot %#v, ok: %v", tt.exec, tt.args, tt.value, tt.ok, value, (err == nil))
		}
	}
}

var hooksLoadFromFileTests = []struct {
	path       string
	asTemplate bool
	ok         bool
}{
	{"../../example/configs/hooks.json", false, true},
	{"../../example/configs/hooks.yaml", false, true},
	{"../../example/configs/hooks.json.tmpl", true, true},
	{"../../example/configs/hooks.yaml.tmpl", true, true},
	{"", false, true},
	// failures
	{"missing.json", false, false},
}

func TestHooksLoadFromFile(t *testing.T) {
	secret := `foo"123`
	os.Setenv("XXXTEST_SECRET", secret)

	for _, tt := range hooksLoadFromFileTests {
		h := &Hooks{}
		err := h.LoadFromFile(tt.path, tt.asTemplate)
		if (err == nil) != tt.ok {
			t.Errorf("%v", err)
		}
	}
}

func TestHooksTemplateLoadFromFile(t *testing.T) {
	secret := `foo"123`
	os.Setenv("XXXTEST_SECRET", secret)

	for _, tt := range hooksLoadFromFileTests {
		if !tt.asTemplate {
			continue
		}

		h := &Hooks{}
		err := h.LoadFromFile(tt.path, tt.asTemplate)
		if (err == nil) != tt.ok {
			t.Errorf("%v", err)
			continue
		}

		s := (*h.Match("webhook").TriggerRule.And)[0].Match.Secret
		if s != secret {
			t.Errorf("Expected secret of %q, got %q", secret, s)
		}
	}
}

var hooksMatchTests = []struct {
	id    string
	hooks Hooks
	value *Hook
}{
	{"a", Hooks{Hook{ID: "a"}}, &Hook{ID: "a"}},
	{"X", Hooks{Hook{ID: "a"}}, new(Hook)},
}

func TestHooksMatch(t *testing.T) {
	for _, tt := range hooksMatchTests {
		value := tt.hooks.Match(tt.id)
		if reflect.DeepEqual(reflect.ValueOf(value), reflect.ValueOf(tt.value)) {
			t.Errorf("failed to match %q:\nexpected %#v,\ngot %#v", tt.id, tt.value, value)
		}
	}
}

var matchRuleTests = []struct {
	typ, regex, secret, value, ipRange string
	param                              Argument
	headers, query, payload            map[string]interface{}
	body                               []byte
	remoteAddr                         string
	ok                                 bool
	err                                bool
}{
	{"value", "", "", "z", "", Argument{"header", "a", "", false}, map[string]interface{}{"A": "z"}, nil, nil, []byte{}, "", true, false},
	{"regex", "^z", "", "z", "", Argument{"header", "a", "", false}, map[string]interface{}{"A": "z"}, nil, nil, []byte{}, "", true, false},
	{"payload-hmac-sha1", "", "secret", "", "", Argument{"header", "a", "", false}, map[string]interface{}{"A": "b17e04cbb22afa8ffbff8796fc1894ed27badd9e"}, nil, nil, []byte(`{"a": "z"}`), "", true, false},
	{"payload-hash-sha1", "", "secret", "", "", Argument{"header", "a", "", false}, map[string]interface{}{"A": "b17e04cbb22afa8ffbff8796fc1894ed27badd9e"}, nil, nil, []byte(`{"a": "z"}`), "", true, false},
	{"payload-hmac-sha256", "", "secret", "", "", Argument{"header", "a", "", false}, map[string]interface{}{"A": "f417af3a21bd70379b5796d5f013915e7029f62c580fb0f500f59a35a6f04c89"}, nil, nil, []byte(`{"a": "z"}`), "", true, false},
	{"payload-hash-sha256", "", "secret", "", "", Argument{"header", "a", "", false}, map[string]interface{}{"A": "f417af3a21bd70379b5796d5f013915e7029f62c580fb0f500f59a35a6f04c89"}, nil, nil, []byte(`{"a": "z"}`), "", true, false},
	// failures
	{"value", "", "", "X", "", Argument{"header", "a", "", false}, map[string]interface{}{"A": "z"}, nil, nil, []byte{}, "", false, false},
	{"regex", "^X", "", "", "", Argument{"header", "a", "", false}, map[string]interface{}{"A": "z"}, nil, nil, []byte{}, "", false, false},
	{"value", "", "2", "X", "", Argument{"header", "a", "", false}, map[string]interface{}{"Y": "z"}, nil, nil, []byte{}, "", false, true}, // reference invalid header
	// errors
	{"regex", "*", "", "", "", Argument{"header", "a", "", false}, map[string]interface{}{"A": "z"}, nil, nil, []byte{}, "", false, true},                   // invalid regex
	{"payload-hmac-sha1", "", "secret", "", "", Argument{"header", "a", "", false}, map[string]interface{}{"A": ""}, nil, nil, []byte{}, "", false, true},   // invalid hmac
	{"payload-hash-sha1", "", "secret", "", "", Argument{"header", "a", "", false}, map[string]interface{}{"A": ""}, nil, nil, []byte{}, "", false, true},   // invalid hmac
	{"payload-hmac-sha256", "", "secret", "", "", Argument{"header", "a", "", false}, map[string]interface{}{"A": ""}, nil, nil, []byte{}, "", false, true}, // invalid hmac
	{"payload-hash-sha256", "", "secret", "", "", Argument{"header", "a", "", false}, map[string]interface{}{"A": ""}, nil, nil, []byte{}, "", false, true}, // invalid hmac
	{"payload-hmac-sha512", "", "secret", "", "", Argument{"header", "a", "", false}, map[string]interface{}{"A": ""}, nil, nil, []byte{}, "", false, true}, // invalid hmac
	{"payload-hash-sha512", "", "secret", "", "", Argument{"header", "a", "", false}, map[string]interface{}{"A": ""}, nil, nil, []byte{}, "", false, true}, // invalid hmac
	// IP whitelisting, valid cases
	{"ip-whitelist", "", "", "", "192.168.0.1/24", Argument{}, nil, nil, nil, []byte{}, "192.168.0.2:9000", true, false}, // valid IPv4, with range
	{"ip-whitelist", "", "", "", "192.168.0.1/24", Argument{}, nil, nil, nil, []byte{}, "192.168.0.2:9000", true, false}, // valid IPv4, with range
	{"ip-whitelist", "", "", "", "192.168.0.1", Argument{}, nil, nil, nil, []byte{}, "192.168.0.1:9000", true, false},    // valid IPv4, no range
	{"ip-whitelist", "", "", "", "::1/24", Argument{}, nil, nil, nil, []byte{}, "[::1]:9000", true, false},               // valid IPv6, with range
	{"ip-whitelist", "", "", "", "::1", Argument{}, nil, nil, nil, []byte{}, "[::1]:9000", true, false},                  // valid IPv6, no range
	// IP whitelisting, invalid cases
	{"ip-whitelist", "", "", "", "192.168.0.1/a", Argument{}, nil, nil, nil, []byte{}, "192.168.0.2:9000", false, true},  // invalid IPv4, with range
	{"ip-whitelist", "", "", "", "192.168.0.a", Argument{}, nil, nil, nil, []byte{}, "192.168.0.2:9000", false, true},    // invalid IPv4, no range
	{"ip-whitelist", "", "", "", "192.168.0.1/24", Argument{}, nil, nil, nil, []byte{}, "192.168.0.a:9000", false, true}, // invalid IPv4 address
	{"ip-whitelist", "", "", "", "::1/a", Argument{}, nil, nil, nil, []byte{}, "[::1]:9000", false, true},                // invalid IPv6, with range
	{"ip-whitelist", "", "", "", "::z", Argument{}, nil, nil, nil, []byte{}, "[::1]:9000", false, true},                  // invalid IPv6, no range
	{"ip-whitelist", "", "", "", "::1/24", Argument{}, nil, nil, nil, []byte{}, "[::z]:9000", false, true},               // invalid IPv6 address
}

func TestMatchRule(t *testing.T) {
	for i, tt := range matchRuleTests {
		r := MatchRule{tt.typ, tt.regex, tt.secret, tt.value, tt.param, tt.ipRange}
		req := &Request{
			Headers: tt.headers,
			Query:   tt.query,
			Payload: tt.payload,
			Body:    tt.body,
			RawRequest: &http.Request{
				RemoteAddr: tt.remoteAddr,
			},
		}
		ok, err := r.Evaluate(req)
		if ok != tt.ok || (err != nil) != tt.err {
			t.Errorf("%d failed to match %#v:\nexpected ok: %#v, err: %v\ngot ok: %#v, err: %v", i, r, tt.ok, tt.err, ok, err)
		}
	}
}

var andRuleTests = []struct {
	desc                    string // description of the test case
	rule                    AndRule
	headers, query, payload map[string]interface{}
	body                    []byte
	ok                      bool
	err                     bool
}{
	{
		"(a=z, b=y): a=z && b=y",
		AndRule{
			{Match: &MatchRule{"value", "", "", "z", Argument{"header", "a", "", false}, ""}},
			{Match: &MatchRule{"value", "", "", "y", Argument{"header", "b", "", false}, ""}},
		},
		map[string]interface{}{"A": "z", "B": "y"},
		nil, nil,
		[]byte{},
		true, false,
	},
	{
		"(a=z, b=Y): a=z && b=y",
		AndRule{
			{Match: &MatchRule{"value", "", "", "z", Argument{"header", "a", "", false}, ""}},
			{Match: &MatchRule{"value", "", "", "y", Argument{"header", "b", "", false}, ""}},
		},
		map[string]interface{}{"A": "z", "B": "Y"},
		nil, nil,
		[]byte{},
		false, false,
	},
	// Complex test to cover Rules.Evaluate
	{
		"(a=z, b=y, c=x, d=w=, e=X, f=X): a=z && (b=y && c=x) && (d=w || e=v) && !f=u",
		AndRule{
			{Match: &MatchRule{"value", "", "", "z", Argument{"header", "a", "", false}, ""}},
			{
				And: &AndRule{
					{Match: &MatchRule{"value", "", "", "y", Argument{"header", "b", "", false}, ""}},
					{Match: &MatchRule{"value", "", "", "x", Argument{"header", "c", "", false}, ""}},
				},
			},
			{
				Or: &OrRule{
					{Match: &MatchRule{"value", "", "", "w", Argument{"header", "d", "", false}, ""}},
					{Match: &MatchRule{"value", "", "", "v", Argument{"header", "e", "", false}, ""}},
				},
			},
			{
				Not: &NotRule{
					Match: &MatchRule{"value", "", "", "u", Argument{"header", "f", "", false}, ""},
				},
			},
		},
		map[string]interface{}{"A": "z", "B": "y", "C": "x", "D": "w", "E": "X", "F": "X"},
		nil, nil,
		[]byte{},
		true, false,
	},
	{"empty rule", AndRule{{}}, nil, nil, nil, nil, false, false},
	// failures
	{
		"invalid rule",
		AndRule{{Match: &MatchRule{"value", "", "", "X", Argument{"header", "a", "", false}, ""}}},
		map[string]interface{}{"Y": "z"},
		nil, nil, nil,
		false, true,
	},
}

func TestAndRule(t *testing.T) {
	for _, tt := range andRuleTests {
		r := &Request{
			Headers: tt.headers,
			Query:   tt.query,
			Payload: tt.payload,
			Body:    tt.body,
		}
		ok, err := tt.rule.Evaluate(r)
		if ok != tt.ok || (err != nil) != tt.err {
			t.Errorf("failed to match %#v:\nexpected ok: %#v, err: %v\ngot ok: %#v, err: %v", tt.desc, tt.ok, tt.err, ok, err)
		}
	}
}

var orRuleTests = []struct {
	desc                    string // description of the test case
	rule                    OrRule
	headers, query, payload map[string]interface{}
	body                    []byte
	ok                      bool
	err                     bool
}{
	{
		"(a=z, b=X): a=z || b=y",
		OrRule{
			{Match: &MatchRule{"value", "", "", "z", Argument{"header", "a", "", false}, ""}},
			{Match: &MatchRule{"value", "", "", "y", Argument{"header", "b", "", false}, ""}},
		},
		map[string]interface{}{"A": "z", "B": "X"},
		nil, nil,
		[]byte{},
		true, false,
	},
	{
		"(a=X, b=y): a=z || b=y",
		OrRule{
			{Match: &MatchRule{"value", "", "", "z", Argument{"header", "a", "", false}, ""}},
			{Match: &MatchRule{"value", "", "", "y", Argument{"header", "b", "", false}, ""}},
		},
		map[string]interface{}{"A": "X", "B": "y"},
		nil, nil,
		[]byte{},
		true, false,
	},
	{
		"(a=Z, b=Y): a=z || b=y",
		OrRule{
			{Match: &MatchRule{"value", "", "", "z", Argument{"header", "a", "", false}, ""}},
			{Match: &MatchRule{"value", "", "", "y", Argument{"header", "b", "", false}, ""}},
		},
		map[string]interface{}{"A": "Z", "B": "Y"},
		nil, nil,
		[]byte{},
		false, false,
	},
	// failures
	{
		"missing parameter node",
		OrRule{
			{Match: &MatchRule{"value", "", "", "z", Argument{"header", "a", "", false}, ""}},
		},
		map[string]interface{}{"Y": "Z"},
		nil, nil,
		[]byte{},
		false, false,
	},
}

func TestOrRule(t *testing.T) {
	for _, tt := range orRuleTests {
		r := &Request{
			Headers: tt.headers,
			Query:   tt.query,
			Payload: tt.payload,
			Body:    tt.body,
		}
		ok, err := tt.rule.Evaluate(r)
		if ok != tt.ok || (err != nil) != tt.err {
			t.Errorf("%#v:\nexpected ok: %#v, err: %v\ngot ok: %#v err: %v", tt.desc, tt.ok, tt.err, ok, err)
		}
	}
}

var notRuleTests = []struct {
	desc                    string // description of the test case
	rule                    NotRule
	headers, query, payload map[string]interface{}
	body                    []byte
	ok                      bool
	err                     bool
}{
	{"(a=z): !a=X", NotRule{Match: &MatchRule{"value", "", "", "X", Argument{"header", "a", "", false}, ""}}, map[string]interface{}{"A": "z"}, nil, nil, []byte{}, true, false},
	{"(a=z): !a=z", NotRule{Match: &MatchRule{"value", "", "", "z", Argument{"header", "a", "", false}, ""}}, map[string]interface{}{"A": "z"}, nil, nil, []byte{}, false, false},
}

func TestNotRule(t *testing.T) {
	for _, tt := range notRuleTests {
		r := &Request{
			Headers: tt.headers,
			Query:   tt.query,
			Payload: tt.payload,
			Body:    tt.body,
		}
		ok, err := tt.rule.Evaluate(r)
		if ok != tt.ok || (err != nil) != tt.err {
			t.Errorf("failed to match %#v:\nexpected ok: %#v, err: %v\ngot ok: %#v, err: %v", tt.rule, tt.ok, tt.err, ok, err)
		}
	}
}

func TestCompare(t *testing.T) {
	for _, tt := range []struct {
		a, b string
		ok   bool
	}{
		{"abcd", "abcd", true},
		{"zyxw", "abcd", false},
	} {
		if ok := compare(tt.a, tt.b); ok != tt.ok {
			t.Errorf("compare failed for %q and %q: got %v\n", tt.a, tt.b, ok)
		}
	}
}

func TestResponseHeaders_String(t *testing.T) {
	// Test with empty headers
	var headers ResponseHeaders
	result := headers.String()
	if result != "name=value" {
		t.Errorf("expected 'name=value', got %q", result)
	}

	// Test with headers
	headers = ResponseHeaders{
		{Name: "Content-Type", Value: "application/json"},
		{Name: "X-Custom", Value: "test"},
	}
	result = headers.String()
	expected := "Content-Type=application/json, X-Custom=test"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestResponseHeaders_Set(t *testing.T) {
	var headers ResponseHeaders

	// Test valid format
	err := headers.Set("Content-Type=application/json")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(headers) != 1 {
		t.Errorf("expected 1 header, got %d", len(headers))
	}
	if headers[0].Name != "Content-Type" || headers[0].Value != "application/json" {
		t.Errorf("unexpected header: %+v", headers[0])
	}

	// Test invalid format
	err = headers.Set("invalid")
	if err == nil {
		t.Error("expected error for invalid format")
	}
}

func TestHooksFiles_String(t *testing.T) {
	// Test with empty files
	var files HooksFiles
	result := files.String()
	if result != "hooks.json" {
		t.Errorf("expected 'hooks.json', got %q", result)
	}

	// Test with files
	files = HooksFiles{"hooks1.json", "hooks2.json"}
	result = files.String()
	expected := "hooks1.json, hooks2.json"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestHooksFiles_Set(t *testing.T) {
	var files HooksFiles

	// Test setting a file
	err := files.Set("hooks.json")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 file, got %d", len(files))
	}
	if files[0] != "hooks.json" {
		t.Errorf("expected 'hooks.json', got %q", files[0])
	}

	// Test setting multiple files
	err = files.Set("hooks2.json")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d", len(files))
	}
}

func TestHooks_Append(t *testing.T) {
	hooks := Hooks{
		{ID: "hook1"},
		{ID: "hook2"},
	}

	// Test appending new hooks
	other := &Hooks{
		{ID: "hook3"},
		{ID: "hook4"},
	}

	err := hooks.Append(other)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(hooks) != 4 {
		t.Errorf("expected 4 hooks, got %d", len(hooks))
	}

	// Test appending duplicate ID
	other = &Hooks{
		{ID: "hook1"},
	}
	err = hooks.Append(other)
	if err == nil {
		t.Error("expected error for duplicate ID")
	}
}

func TestHook_ExtractCommandArgumentsForFile(t *testing.T) {
	hook := Hook{
		PassFileToCommand: []Argument{
			{
				Name:   "file1",
				Source: SourcePayload,
			},
			{
				Name:         "file2",
				Source:       SourcePayload,
				Base64Decode: true,
			},
		},
	}

	// Create a request with payload
	payload := map[string]interface{}{
		"file1": "test content",
		"file2": base64.StdEncoding.EncodeToString([]byte("base64 content")),
	}
	req := &Request{
		Payload: payload,
		Body:    []byte(`{"file1":"test content","file2":"YmFzZTY0IGNvbnRlbnQ="}`),
	}

	args, errs := hook.ExtractCommandArgumentsForFile(req)
	if len(errs) > 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	if len(args) != 2 {
		t.Errorf("expected 2 args, got %d", len(args))
	}
	if string(args[0].Data) != "test content" {
		t.Errorf("expected 'test content', got %q", string(args[0].Data))
	}
	if string(args[1].Data) != "base64 content" {
		t.Errorf("expected 'base64 content', got %q", string(args[1].Data))
	}
}

func TestHook_SanitizeHTTPMethods(t *testing.T) {
	tests := []struct {
		name            string
		methods         []string
		expectedLength  int
		expectedMethods []string
	}{
		{
			name:            "Valid methods",
			methods:         []string{"GET", "POST", "PUT"},
			expectedLength:  3,
			expectedMethods: []string{"GET", "POST", "PUT"},
		},
		{
			name:            "Lowercase methods",
			methods:         []string{"get", "post"},
			expectedLength:  2,
			expectedMethods: []string{"GET", "POST"},
		},
		{
			name:            "Methods with whitespace",
			methods:         []string{" GET ", " POST "},
			expectedLength:  2,
			expectedMethods: []string{"GET", "POST"},
		},
		{
			name:            "Invalid methods",
			methods:         []string{"INVALID", "GET"},
			expectedLength:  1,
			expectedMethods: []string{"GET"},
		},
		{
			name:            "Duplicate methods",
			methods:         []string{"GET", "POST", "GET"},
			expectedLength:  2,
			expectedMethods: []string{"GET", "POST"},
		},
		{
			name:            "Empty methods",
			methods:         []string{},
			expectedLength:  0,
			expectedMethods: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := &Hook{
				ID:         "test-hook",
				HTTPMethods: tt.methods,
			}
			hook.SanitizeHTTPMethods()

			if len(hook.HTTPMethods) != tt.expectedLength {
				t.Errorf("SanitizeHTTPMethods() length = %d, want %d", len(hook.HTTPMethods), tt.expectedLength)
			}

			if tt.expectedMethods != nil {
				for i, expected := range tt.expectedMethods {
					if hook.HTTPMethods[i] != expected {
						t.Errorf("SanitizeHTTPMethods()[%d] = %s, want %s", i, hook.HTTPMethods[i], expected)
					}
				}
			}
		})
	}
}

func TestGetCompiledRegex(t *testing.T) {
	// Test valid regex
	re, err := getCompiledRegex("^test$")
	if err != nil {
		t.Errorf("getCompiledRegex() error = %v", err)
	}
	if re == nil {
		t.Error("getCompiledRegex() returned nil regex")
	}

	// Test invalid regex
	_, err = getCompiledRegex("[invalid")
	if err == nil {
		t.Error("getCompiledRegex() should return error for invalid regex")
	}

	// Test empty pattern
	_, err = getCompiledRegex("")
	if err == nil {
		t.Error("getCompiledRegex() should return error for empty pattern")
	}

	// Test caching (same pattern should return same instance)
	re1, _ := getCompiledRegex("^cached$")
	re2, _ := getCompiledRegex("^cached$")
	if re1 != re2 {
		t.Error("getCompiledRegex() should return cached regex for same pattern")
	}
}

func TestCheckScalrSignature_WithDateCheck(t *testing.T) {
	// Test with date check enabled
	headers := map[string]interface{}{
		"Date":        time.Now().Format("Mon 02 Jan 2006 15:04:05 MST"),
		"X-Signature": "test-signature",
	}
	req := &Request{
		Headers: headers,
		Body:    []byte(`{"test": "data"}`),
	}

	valid, err := CheckScalrSignature(req, "secret", true)
	// May be false due to signature mismatch, but should not error on date parsing
	if err != nil && !strings.Contains(err.Error(), "signature") {
		t.Errorf("CheckScalrSignature() unexpected error: %v", err)
	}
	_ = valid
}

func TestCheckScalrSignature_InvalidDate(t *testing.T) {
	headers := map[string]interface{}{
		"Date":        "Invalid Date Format",
		"X-Signature": "test-signature",
	}
	req := &Request{
		Headers: headers,
		Body:    []byte(`{"test": "data"}`),
	}

	valid, err := CheckScalrSignature(req, "secret", true)
	if err == nil {
		t.Error("CheckScalrSignature() should return error for invalid date format")
	}
	if valid {
		t.Error("CheckScalrSignature() should return false for invalid date")
	}
}

func TestCheckScalrSignature_OutdatedDate(t *testing.T) {
	// Test with date more than 5 minutes old
	oldDate := time.Now().Add(-10 * time.Minute).Format("Mon 02 Jan 2006 15:04:05 MST")
	headers := map[string]interface{}{
		"Date":        oldDate,
		"X-Signature": "test-signature",
	}
	req := &Request{
		Headers: headers,
		Body:    []byte(`{"test": "data"}`),
	}

	valid, err := CheckScalrSignature(req, "secret", true)
	if err == nil {
		t.Error("CheckScalrSignature() should return error for outdated date")
	}
	if valid {
		t.Error("CheckScalrSignature() should return false for outdated date")
	}
}

func TestArgument_Get_EntireSources(t *testing.T) {
	req := &Request{
		Headers: map[string]interface{}{"Header1": "value1"},
		Query:   map[string]interface{}{"param1": "value1"},
		Payload: map[string]interface{}{"key1": "value1"},
	}

	// Test entire-payload
	arg := Argument{Source: SourceEntirePayload}
	value, err := arg.Get(req)
	if err != nil {
		t.Errorf("Argument.Get() error = %v", err)
	}
	if !strings.Contains(value, "key1") {
		t.Error("Argument.Get() should return entire payload as JSON")
	}

	// Test entire-query
	arg = Argument{Source: SourceEntireQuery}
	value, err = arg.Get(req)
	if err != nil {
		t.Errorf("Argument.Get() error = %v", err)
	}
	if !strings.Contains(value, "param1") {
		t.Error("Argument.Get() should return entire query as JSON")
	}

	// Test entire-headers
	arg = Argument{Source: SourceEntireHeaders}
	value, err = arg.Get(req)
	if err != nil {
		t.Errorf("Argument.Get() error = %v", err)
	}
	if !strings.Contains(value, "Header1") {
		t.Error("Argument.Get() should return entire headers as JSON")
	}
}

func TestArgument_Get_RequestSource(t *testing.T) {
	req := &Request{
		RawRequest: &http.Request{
			Method:     "POST",
			RemoteAddr: "192.168.1.1:8080",
		},
	}

	// Test method
	arg := Argument{Source: SourceRequest, Name: "method"}
	value, err := arg.Get(req)
	if err != nil {
		t.Errorf("Argument.Get() error = %v", err)
	}
	if value != "POST" {
		t.Errorf("Argument.Get() = %s, want POST", value)
	}

	// Test remote-addr
	arg = Argument{Source: SourceRequest, Name: "remote-addr"}
	value, err = arg.Get(req)
	if err != nil {
		t.Errorf("Argument.Get() error = %v", err)
	}
	if value != "192.168.1.1:8080" {
		t.Errorf("Argument.Get() = %s, want 192.168.1.1:8080", value)
	}

	// Test unsupported key
	arg = Argument{Source: SourceRequest, Name: "unsupported"}
	_, err = arg.Get(req)
	if err == nil {
		t.Error("Argument.Get() should return error for unsupported request key")
	}
}

func TestArgument_Get_NilRequest(t *testing.T) {
	req := &Request{
		RawRequest: nil,
	}

	arg := Argument{Source: SourceRequest, Name: "method"}
	_, err := arg.Get(req)
	if err == nil {
		t.Error("Argument.Get() should return error for nil request")
	}
}
