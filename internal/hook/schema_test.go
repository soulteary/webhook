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
