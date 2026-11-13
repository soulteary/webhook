package hook

import (
	"sync"
	"testing"
)

func TestCheckIPWhitelist_MixedFamilies(t *testing.T) {
	const list = "10.0.0.1 2001:db8::1"
	cases := []struct {
		remote string
		want   bool
	}{
		{"10.0.0.1:1234", true},
		{"[2001:db8::1]:1234", true},
		{"10.0.0.2:1234", false},
		{"[2001:db8::2]:1234", false},
	}
	for _, c := range cases {
		got, err := CheckIPWhitelist(c.remote, list)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", c.remote, err)
		}
		if got != c.want {
			t.Errorf("%s: got %v, want %v", c.remote, got, c.want)
		}
	}
}

// Header values are normally strings, but the signature helpers must not
// panic if a hook rewrote one (parse-parameters-as-json) into an object.
func TestSignatureChecks_NonStringHeaders(t *testing.T) {
	obj := map[string]interface{}{"a": "b"}
	r := &Request{Headers: map[string]interface{}{"Authorization": obj, "X-Signature": obj, "Date": obj}}
	if ok, _ := CheckMSTeamsSignature(r, "c2VjcmV0"); ok {
		t.Error("MS Teams signature accepted a non-string header")
	}
	if ok, _ := CheckScalrSignature(r, "secret", false); ok {
		t.Error("Scalr signature accepted a non-string header")
	}
}

func TestCheckScalrSignature_DoesNotMutateBody(t *testing.T) {
	backing := []byte("payload-----------------")
	body := backing[:7]
	r := &Request{Body: body, Headers: map[string]interface{}{"X-Signature": "bad", "Date": "Fri 08 Sep 2017 11:24:32 UTC"}}
	_, _ = CheckScalrSignature(r, "secret", false)
	if string(backing) != "payload-----------------" {
		t.Fatalf("request body backing array was overwritten: %q", backing)
	}
}

func TestReplaceParameter_NestedAndMismatchedTypes(t *testing.T) {
	params := map[string]interface{}{
		"a":    map[string]interface{}{"b": "{}", "list": []interface{}{"{}"}},
		"text": "plain",
	}
	if !ReplaceParameter("a.b", &params, "new") {
		t.Fatal("nested key was not replaced")
	}
	if got := params["a"].(map[string]interface{})["b"]; got != "new" {
		t.Fatalf("nested value = %v", got)
	}
	if !ReplaceParameter("a.list.0", &params, "item") {
		t.Fatal("slice element was not replaced")
	}
	// Traversing through a scalar used to panic on a failed type assertion.
	if ReplaceParameter("text.child", &params, "x") {
		t.Fatal("replacement through a scalar reported success")
	}
	if ReplaceParameter("missing", &params, "x") {
		t.Fatal("missing key reported success")
	}
}

// Run with -race: resolving the default env name must not write to the shared Hook.
func TestExtractCommandArgumentsForFile_NoSharedWrite(t *testing.T) {
	h := &Hook{ID: "h", PassFileToCommand: []Argument{{Source: SourcePayload, Name: "file"}}}
	r := &Request{Payload: map[string]interface{}{"file": "data"}}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			files, errs := h.ExtractCommandArgumentsForFile(r)
			if len(errs) != 0 || len(files) != 1 || files[0].EnvName != EnvNamespace+"FILE" {
				t.Errorf("unexpected result: %+v %v", files, errs)
			}
		}()
	}
	wg.Wait()
	if h.PassFileToCommand[0].EnvName != "" {
		t.Fatal("shared hook definition was modified")
	}
}
