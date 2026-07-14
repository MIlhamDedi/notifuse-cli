package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/milhamdedi/notifuse-cli/internal/apperr"
	"github.com/milhamdedi/notifuse-cli/internal/client"
	"github.com/milhamdedi/notifuse-cli/internal/config"
	"github.com/milhamdedi/notifuse-cli/internal/output"
	"github.com/milhamdedi/notifuse-cli/internal/secret"
	"github.com/spf13/cobra"
)

const version = "0.2.0"

type dependencies struct {
	stdin      io.Reader
	stdout     io.Writer
	stderr     io.Writer
	getenv     func(string) string
	readFile   func(string) ([]byte, error)
	httpClient client.HTTPDoer
}

type runtime struct {
	spec        []byte
	deps        dependencies
	pretty      bool
	profileName string
	configPath  string
}

func Execute(spec []byte, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	root := newRoot(spec, dependencies{
		stdin: stdin, stdout: stdout, stderr: stderr,
		getenv: os.Getenv, readFile: os.ReadFile,
	})
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		fmt.Fprintf(stderr, "error: %s\n", err)
		return apperr.Code(err)
	}
	return apperr.ExitOK
}

func newRoot(spec []byte, deps dependencies) *cobra.Command {
	app := &runtime{spec: spec, deps: deps}
	root := &cobra.Command{
		Use:           "notifuse",
		Short:         "Agent-safe CLI for Notifuse workspaces",
		Version:       version,
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.SetVersionTemplate("notifuse {{.Version}}\n")
	root.PersistentFlags().BoolVar(&app.pretty, "pretty", false, "Pretty-print JSON output")
	root.PersistentFlags().StringVar(&app.profileName, "profile", "", "Profile name from config")
	root.PersistentFlags().StringVar(&app.configPath, "config", "", "Config path; defaults to ~/.config/notifuse-cli/config.yaml")
	root.AddCommand(
		app.newProfilesCommand(),
		app.newAuthCommand(),
		app.newOpenAPICommand(),
		app.newAPICommand(),
		app.newContactsCommand(),
		app.newTemplatesCommand(),
		app.newBroadcastsCommand(),
		app.newTransactionalCommand(),
	)
	return root
}

func (r *runtime) configFile() (string, config.File, error) {
	path := strings.TrimSpace(r.configPath)
	if path == "" {
		var err error
		path, err = config.DefaultPath()
		if err != nil {
			return "", config.File{}, err
		}
	}
	file, err := config.Load(path)
	return path, file, err
}

func (r *runtime) profile() (string, config.Profile, error) {
	_, file, err := r.configFile()
	if err != nil {
		return "", config.Profile{}, err
	}
	return file.Resolve(r.profileName)
}

func (r *runtime) apiClient() (string, config.Profile, *client.Client, error) {
	name, profile, err := r.profile()
	if err != nil {
		return "", config.Profile{}, nil, err
	}
	apiKey, err := (secret.Resolver{Getenv: r.deps.getenv, ReadFile: r.deps.readFile}).Resolve(profile.APIKeyRef)
	if err != nil {
		return "", config.Profile{}, nil, err
	}
	apiClient, err := client.New(profile.Endpoint, apiKey, r.deps.httpClient)
	return name, profile, apiClient, err
}

func (r *runtime) perform(command *cobra.Command, request client.Request) error {
	_, _, apiClient, err := r.apiClient()
	if err != nil {
		return err
	}
	response, err := apiClient.Do(command.Context(), request)
	if err != nil {
		return err
	}
	if statusErr := apperr.FromStatus(response.Status); statusErr != nil {
		writeUpstreamError(r.deps.stderr, response.Body, r.pretty)
		return statusErr
	}
	return output.JSON(r.deps.stdout, response.Body, r.pretty)
}

func writeUpstreamError(writer io.Writer, body []byte, pretty bool) {
	if len(strings.TrimSpace(string(body))) == 0 {
		return
	}
	if err := output.JSON(writer, body, pretty); err == nil {
		return
	}
	const limit = 4096
	if len(body) > limit {
		body = append(body[:limit], []byte("...")...)
	}
	fmt.Fprintf(writer, "upstream response: %q\n", body)
}

func withWorkspaceQuery(profile config.Profile, queries []string) (url.Values, error) {
	values := url.Values{}
	values.Set("workspace_id", profile.WorkspaceID)
	for _, item := range queries {
		key, value, ok := strings.Cut(item, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, apperr.New(apperr.ExitUsage, "query must be KEY=VALUE")
		}
		values.Add(key, value)
	}
	return values, nil
}

func withWorkspaceBody(profile config.Profile, body []byte) ([]byte, error) {
	var object map[string]any
	if err := json.Unmarshal(body, &object); err != nil {
		return nil, err
	}
	if existing, ok := object["workspace_id"]; ok && fmt.Sprint(existing) != profile.WorkspaceID {
		return nil, apperr.New(apperr.ExitUsage, "body workspace_id %q does not match selected profile workspace_id %q", existing, profile.WorkspaceID)
	}
	object["workspace_id"] = profile.WorkspaceID
	return json.Marshal(object)
}

func dryRunResponse(writer io.Writer, profileName string, profile config.Profile, method string, path string, query url.Values, body []byte, pretty bool) error {
	response := map[string]any{
		"dry_run":    true,
		"profile":    profileName,
		"endpoint":   profile.Endpoint,
		"workspace":  profile.WorkspaceID,
		"method":     method,
		"path":       path,
		"query":      query,
		"body":       json.RawMessage(body),
		"would_send": false,
	}
	encoded, err := json.Marshal(response)
	if err != nil {
		return err
	}
	return output.JSON(writer, encoded, pretty)
}
