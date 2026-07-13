package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func run(args ...string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := Execute([]byte(`{"openapi":"3.0.3","info":{"title":"Test","version":"1"},"paths":{"/api/contacts.list":{"get":{"operationId":"listContacts","summary":"List contacts"}}}}`), args, strings.NewReader(""), &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestOpenAPIList(t *testing.T) {
	code, stdout, stderr := run("openapi", "list", "--filter", "contacts")
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "/api/contacts.list") {
		t.Fatalf("missing operation: %s", stdout)
	}
}

func TestBlockedBroadcastSend(t *testing.T) {
	code, _, stderr := run("broadcasts", "send")
	if code == 0 {
		t.Fatal("expected blocked command to fail")
	}
	if !strings.Contains(stderr, "blocked") {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}
