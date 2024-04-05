package server

import (
	"bytes"
	"log"
	"os"
	"regexp"
	"runtime"
	"testing"

	"github.com/soulteary/webhook/internal/hook"
)

func TestStaticParams(t *testing.T) {
	// FIXME(moorereason): incorporate this test into TestWebhook.
	//   Need to be able to execute a binary with a space in the filename.
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	spHeaders := make(map[string]interface{})
	spHeaders["User-Agent"] = "curl/7.54.0"
	spHeaders["Accept"] = "*/*"

	// case 2: binary with spaces in its name
	d1 := []byte("#!/bin/sh\n/bin/echo\n")
	err := os.WriteFile("/tmp/with space", d1, 0755)
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer os.Remove("/tmp/with space")

	spHook := &hook.Hook{
		ID:                      "static-params-name-space",
		ExecuteCommand:          "/tmp/with space",
		CommandWorkingDirectory: "/tmp",
		ResponseMessage:         "success",
		CaptureCommandOutput:    true,
		PassArgumentsToCommand: []hook.Argument{
			{Source: "string", Name: "passed"},
		},
	}

	b := &bytes.Buffer{}
	log.SetOutput(b)

	r := &hook.Request{
		ID:      "test",
		Headers: spHeaders,
	}
	_, err = handleHook(spHook, r, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v\n", err)
	}
	matched, _ := regexp.MatchString("(?s)command output: .*static-params-name-space", b.String())
	if !matched {
		t.Fatalf("Unexpected log output:\n%sn", b)
	}
}
