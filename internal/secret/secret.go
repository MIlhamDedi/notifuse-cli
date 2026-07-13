package secret

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"golang.org/x/term"
)

type Resolver struct {
	Getenv   func(string) string
	ReadFile func(string) ([]byte, error)
}

func (r Resolver) Resolve(ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", fmt.Errorf("empty api key reference")
	}
	if strings.HasPrefix(ref, "env:") {
		name := strings.TrimPrefix(ref, "env:")
		value := strings.TrimSpace(r.getenv(name))
		if value == "" {
			return "", fmt.Errorf("environment variable %s is empty", name)
		}
		return value, nil
	}
	if strings.HasPrefix(ref, "file:") {
		path := strings.TrimPrefix(ref, "file:")
		body, err := r.readFile(path)
		if err != nil {
			return "", err
		}
		value := strings.TrimSpace(string(body))
		if value == "" {
			return "", fmt.Errorf("secret file %s is empty", path)
		}
		return value, nil
	}
	if strings.HasPrefix(ref, "keychain:") {
		return resolveKeychain(strings.TrimPrefix(ref, "keychain:"))
	}
	return "", fmt.Errorf("unsupported api_key_ref %q; use env:, file:, or keychain:", ref)
}

func StoreKeychain(ref string, value string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("keychain storage is only supported on macOS")
	}
	service, account, err := splitKeychain(ref)
	if err != nil {
		return err
	}
	cmd := exec.Command("security", "add-generic-password", "-U", "-s", service, "-a", account, "-w", value)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("security add-generic-password failed: %s", strings.TrimSpace(stderr.String()))
	}
	return nil
}

func ReadAPIKeyFromTerminal(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	bytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	value := strings.TrimSpace(string(bytes))
	if value == "" {
		return "", fmt.Errorf("empty API key")
	}
	return value, nil
}

func (r Resolver) getenv(name string) string {
	if r.Getenv != nil {
		return r.Getenv(name)
	}
	return os.Getenv(name)
}

func (r Resolver) readFile(path string) ([]byte, error) {
	if r.ReadFile != nil {
		return r.ReadFile(path)
	}
	return os.ReadFile(path)
}

func resolveKeychain(ref string) (string, error) {
	if runtime.GOOS != "darwin" {
		return "", fmt.Errorf("keychain lookup is only supported on macOS")
	}
	service, account, err := splitKeychain(ref)
	if err != nil {
		return "", err
	}
	cmd := exec.Command("security", "find-generic-password", "-s", service, "-a", account, "-w")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("security find-generic-password failed for %s/%s: %s", service, account, strings.TrimSpace(stderr.String()))
	}
	value := strings.TrimSpace(string(output))
	if value == "" {
		return "", fmt.Errorf("keychain item %s/%s is empty", service, account)
	}
	return value, nil
}

func splitKeychain(ref string) (string, string, error) {
	parts := strings.SplitN(ref, "/", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf("keychain ref must be keychain:<service>/<account>")
	}
	return parts[0], parts[1], nil
}
