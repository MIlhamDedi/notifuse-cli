package cmd

import (
	"bytes"
	"os"
	"path/filepath"
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

func TestTemplateCompileFromFileDryRun(t *testing.T) {
	configPath := writeTestConfig(t)
	templatePath := filepath.Join(t.TempDir(), "template.json")
	template := `{
		"id": "video_ready",
		"name": "Video Ready",
		"channel": "email",
		"category": "transactional",
		"email": {
			"subject": "Your analysis is ready",
			"compiled_preview": "<p>Ready</p>",
			"visual_editor_tree": {
				"id":"mjml-1",
				"type":"mjml",
				"children":[
					{"id":"body-1","type":"mj-body","children":[
						{"id":"section-1","type":"mj-section","children":[
							{"id":"column-1","type":"mj-column","children":[
								{"id":"text-1","type":"mj-text","content":"hello"}
							]}
						]}
					]}
				]
			}
		},
		"test_data": {"first_name":"Ilham"}
	}`
	if err := os.WriteFile(templatePath, []byte(template), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Execute(nil, []string{"--config", configPath, "--profile", "courtpro", "--pretty", "templates", "compile-from-file", "--file", templatePath, "--dry-run"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	for _, expected := range []string{`"path": "/api/templates.compile"`, `"workspace_id": "courtpro"`, `"visual_editor_tree"`} {
		if !strings.Contains(stdout.String(), expected) {
			t.Fatalf("missing %s in stdout: %s", expected, stdout.String())
		}
	}
}

func TestTransactionalTestSendRequiresAllowedRecipient(t *testing.T) {
	configPath := writeTestConfig(t)
	body := `{"notification":{"id":"video_ready","contact":{"email":"external@example.com"},"channels":["email"]}}`
	var stdout, stderr bytes.Buffer
	code := Execute(nil, []string{"--config", configPath, "--profile", "courtpro", "transactional", "test-send", "--body-json", body, "--dry-run"}, strings.NewReader(""), &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected failure, stdout=%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "allowed_test_recipients") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestTransactionalTestSendDryRun(t *testing.T) {
	configPath := writeTestConfig(t)
	body := `{"notification":{"id":"video_ready","contact":{"email":"ilham@alif.ventures"},"channels":["email"]}}`
	var stdout, stderr bytes.Buffer
	code := Execute(nil, []string{"--config", configPath, "--profile", "courtpro", "--pretty", "transactional", "test-send", "--body-json", body, "--dry-run"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	for _, expected := range []string{`"path": "/api/transactional.send"`, `"workspace_id": "courtpro"`, `"id": "video_ready"`} {
		if !strings.Contains(stdout.String(), expected) {
			t.Fatalf("missing %s in stdout: %s", expected, stdout.String())
		}
	}
}

func writeTestConfig(t *testing.T) string {
	t.Helper()
	t.Setenv("NOTIFUSE_API_KEY_TEST", "test-key")
	path := filepath.Join(t.TempDir(), "config.yaml")
	body := `profiles:
  courtpro:
    endpoint: https://notifuse.example.test
    workspace_id: courtpro
    api_key_ref: env:NOTIFUSE_API_KEY_TEST
    allowed_test_recipients:
      - ilham@alif.ventures
`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
