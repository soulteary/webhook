package hook

import (
	"encoding/json"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestSchemaCoversHookFields(t *testing.T) {
	data, err := os.ReadFile("../../schema/hooks.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Defs map[string]struct {
			Properties map[string]json.RawMessage `json:"properties"`
		} `json:"$defs"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	properties := schema.Defs["hook"].Properties
	typeOfHook := reflect.TypeOf(Hook{})
	for i := 0; i < typeOfHook.NumField(); i++ {
		name := strings.Split(typeOfHook.Field(i).Tag.Get("json"), ",")[0]
		if name == "" || name == "-" {
			continue
		}
		if _, ok := properties[name]; !ok {
			t.Errorf("schema is missing Hook field %q", name)
		}
	}
}

func TestSchemaRejectsUnreachableHookIDs(t *testing.T) {
	data, err := os.ReadFile("../../schema/hooks.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Defs map[string]struct {
			Properties map[string]struct {
				Pattern string `json:"pattern"`
			} `json:"properties"`
		} `json:"$defs"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	pattern := schema.Defs["hook"].Properties["id"].Pattern
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		t.Fatalf("invalid hook ID pattern: %v", err)
	}
	for _, id := range []string{"hook", "hook with spaces", "hook-编号"} {
		if !compiled.MatchString(id) {
			t.Errorf("schema rejected reachable hook ID %q", id)
		}
	}
	for _, id := range []string{"", " hook", "hook ", "foo\rbar", "foo\nbar", "foo\tbar"} {
		if compiled.MatchString(id) {
			t.Errorf("schema accepted unreachable hook ID %q", id)
		}
	}
}

func TestSchemaMatchesRuntimeArgumentNameRules(t *testing.T) {
	data, err := os.ReadFile("../../schema/hooks.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Defs map[string]struct {
			AllOf []struct {
				If struct {
					Properties map[string]struct {
						Const any `json:"const"`
					} `json:"properties"`
				} `json:"if"`
				Then struct {
					Properties map[string]struct {
						Pattern string `json:"pattern"`
					} `json:"properties"`
				} `json:"then"`
			} `json:"allOf"`
		} `json:"$defs"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}

	patterns := make(map[string]string)
	for _, condition := range schema.Defs["argument"].AllOf {
		if source, ok := condition.If.Properties["source"].Const.(string); ok {
			patterns[source] = condition.Then.Properties["name"].Pattern
		}
	}
	requestPattern, err := regexp.Compile(patterns[SourceRequest])
	if err != nil {
		t.Fatalf("invalid request-key pattern: %v", err)
	}
	for _, name := range []string{"method", "Method", "METHOD", "remote-addr", "REMOTE-ADDR"} {
		if !requestPattern.MatchString(name) {
			t.Errorf("schema rejected supported request key %q", name)
		}
	}
	for _, name := range []string{"method-name", "remote_addr", "unsupported"} {
		if requestPattern.MatchString(name) {
			t.Errorf("schema accepted unsupported request key %q", name)
		}
	}

	literalPattern, err := regexp.Compile(patterns[SourceString])
	if err != nil {
		t.Fatalf("invalid literal-argument pattern: %v", err)
	}
	if !literalPattern.MatchString("valid argument") {
		t.Error("schema rejected a valid literal argument")
	}
	if literalPattern.MatchString("bad\x00argument") {
		t.Error("schema accepted a literal argument containing NUL")
	}
}

func TestSchemaConstrainsMSTeamsSecretsToStandardBase64(t *testing.T) {
	data, err := os.ReadFile("../../schema/hooks.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Defs map[string]struct {
			AllOf []struct {
				If struct {
					Properties map[string]struct {
						Const any `json:"const"`
					} `json:"properties"`
				} `json:"if"`
				Then struct {
					Properties map[string]struct {
						Pattern string `json:"pattern"`
					} `json:"properties"`
				} `json:"then"`
			} `json:"allOf"`
		} `json:"$defs"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}

	pattern := ""
	for _, condition := range schema.Defs["match"].AllOf {
		if condition.If.Properties["type"].Const == MSTeamsSignature {
			pattern = condition.Then.Properties["secret"].Pattern
			break
		}
	}
	if pattern == "" {
		t.Fatal("schema is missing the msteams-signature secret constraint")
	}
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		t.Fatalf("invalid msteams-signature secret pattern: %v", err)
	}
	for _, value := range []string{"aGVsbG8=", "YWJjZA==", "YWJj"} {
		if !compiled.MatchString(value) {
			t.Errorf("schema rejected valid standard Base64 %q", value)
		}
	}
	for _, value := range []string{"not_base64", "abc", "===="} {
		if compiled.MatchString(value) {
			t.Errorf("schema accepted invalid standard Base64 %q", value)
		}
	}
}

func TestSchemaConstrainsLiteralFileArgumentsToStandardBase64(t *testing.T) {
	data, err := os.ReadFile("../../schema/hooks.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Defs map[string]struct {
			AllOf []struct {
				If struct {
					Properties map[string]struct {
						Const any `json:"const"`
					} `json:"properties"`
				} `json:"if"`
				Then struct {
					Properties map[string]struct {
						Pattern string `json:"pattern"`
					} `json:"properties"`
				} `json:"then"`
			} `json:"allOf"`
		} `json:"$defs"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}

	pattern := ""
	for _, condition := range schema.Defs["fileArgument"].AllOf {
		properties := condition.If.Properties
		if properties["source"].Const == SourceString && properties["base64decode"].Const == true {
			pattern = condition.Then.Properties["name"].Pattern
			break
		}
	}
	if pattern == "" {
		t.Fatal("schema is missing the literal base64 file-argument constraint")
	}
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		t.Fatalf("invalid literal file-argument Base64 pattern: %v", err)
	}
	for _, value := range []string{"aGVsbG8=", "YWJjZA==", "YWJj"} {
		if !compiled.MatchString(value) {
			t.Errorf("schema rejected valid standard Base64 %q", value)
		}
	}
	for _, value := range []string{"not_base64", "abc", "===="} {
		if compiled.MatchString(value) {
			t.Errorf("schema accepted invalid standard Base64 %q", value)
		}
	}
}

func TestSchemaConstrainsResponseHeaderNames(t *testing.T) {
	data, err := os.ReadFile("../../schema/hooks.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Defs map[string]struct {
			Properties map[string]struct {
				Pattern string `json:"pattern"`
			} `json:"properties"`
		} `json:"$defs"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	pattern := schema.Defs["header"].Properties["name"].Pattern
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		t.Fatalf("invalid response-header name pattern: %v", err)
	}
	for _, name := range []string{"Content-Type", "X_Custom", "!#$%&'*+.^_`|~"} {
		if !compiled.MatchString(name) {
			t.Errorf("schema rejected valid response-header name %q", name)
		}
	}
	for _, name := range []string{"", "Bad Header", "Bad:Header", "Bad\r\nHeader"} {
		if compiled.MatchString(name) {
			t.Errorf("schema accepted invalid response-header name %q", name)
		}
	}
}

func TestSchemaConstrainsResponseHeaderValues(t *testing.T) {
	data, err := os.ReadFile("../../schema/hooks.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Defs map[string]struct {
			Properties map[string]struct {
				Pattern string `json:"pattern"`
			} `json:"properties"`
		} `json:"$defs"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	pattern := schema.Defs["header"].Properties["value"].Pattern
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		t.Fatalf("invalid response-header value pattern: %v", err)
	}
	for _, value := range []string{"", "text/plain", "valid\tvalue", "ümlaut"} {
		if !compiled.MatchString(value) {
			t.Errorf("schema rejected valid response-header value %q", value)
		}
	}
	for _, value := range []string{"bad\rvalue", "bad\nvalue", "bad\x00value", "bad\x1fvalue", "bad\x7fvalue"} {
		if compiled.MatchString(value) {
			t.Errorf("schema accepted invalid response-header value %q", value)
		}
	}
}

func TestSchemaConstrainsTemporaryFilePatterns(t *testing.T) {
	data, err := os.ReadFile("../../schema/hooks.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Defs map[string]struct {
			AllOf []struct {
				Properties map[string]struct {
					Pattern string `json:"pattern"`
				} `json:"properties"`
				Then struct {
					Properties map[string]struct {
						Pattern string `json:"pattern"`
					} `json:"properties"`
				} `json:"then"`
			} `json:"allOf"`
		} `json:"$defs"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}

	explicitPattern := ""
	for _, condition := range schema.Defs["fileArgument"].AllOf {
		if condition.Properties["envname"].Pattern != "" {
			explicitPattern = condition.Properties["envname"].Pattern
			break
		}
	}
	if explicitPattern == "" {
		t.Fatal("schema is missing the explicit temporary-file pattern constraint")
	}
	derivedPattern := ""
	for _, condition := range schema.Defs["fileArgument"].AllOf {
		if condition.Then.Properties["name"].Pattern == explicitPattern {
			derivedPattern = condition.Then.Properties["name"].Pattern
			break
		}
	}
	if derivedPattern == "" {
		t.Fatal("schema is missing the derived temporary-file pattern constraint")
	}
	patterns := []string{explicitPattern, derivedPattern}
	for _, pattern := range patterns {
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			t.Fatalf("invalid temporary-file pattern: %v", err)
		}
		for _, value := range []string{"", "WEBHOOK_FILE", "payload-*.json"} {
			if !compiled.MatchString(value) {
				t.Errorf("schema rejected valid temporary-file pattern %q", value)
			}
		}
		for _, value := range []string{`bad?name`, `bad:name`, `bad\"name`, `bad<name`, `bad>name`, `bad|name`, "bad\x00name", "bad\x1fname", "two**stars"} {
			if compiled.MatchString(value) {
				t.Errorf("schema accepted invalid temporary-file pattern %q", value)
			}
		}
	}
}
