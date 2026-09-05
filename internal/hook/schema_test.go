package hook

import (
	"encoding/json"
	"os"
	"reflect"
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
