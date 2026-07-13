package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/milhamdedi/notifuse-cli/internal/apperr"
	"github.com/milhamdedi/notifuse-cli/internal/config"
	"github.com/milhamdedi/notifuse-cli/internal/output"
	"github.com/spf13/cobra"
)

func (r *runtime) newProfilesCommand() *cobra.Command {
	command := &cobra.Command{Use: "profiles", Short: "Manage non-secret workspace profiles"}
	command.AddCommand(
		r.newProfilesListCommand(),
		r.newProfilesShowCommand(),
		r.newProfilesAddCommand(),
		r.newProfilesDefaultCommand(),
	)
	return command
}

func (r *runtime) newProfilesListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured profiles without secrets",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			_, file, err := r.configFile()
			if err != nil {
				return err
			}
			type row struct {
				Name        string `json:"name"`
				Default     bool   `json:"default"`
				Endpoint    string `json:"endpoint"`
				WorkspaceID string `json:"workspace_id"`
				APIKeyRef   string `json:"api_key_ref"`
			}
			var rows []row
			for name, profile := range file.Profiles {
				rows = append(rows, row{
					Name: name, Default: name == file.DefaultProfile,
					Endpoint: profile.Endpoint, WorkspaceID: profile.WorkspaceID, APIKeyRef: profile.APIKeyRef,
				})
			}
			sort.Slice(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })
			body, err := json.Marshal(map[string]any{"profiles": rows})
			if err != nil {
				return err
			}
			return output.JSON(r.deps.stdout, body, r.pretty)
		},
	}
}

func (r *runtime) newProfilesShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show NAME",
		Short: "Show one profile without resolving its secret",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			_, file, err := r.configFile()
			if err != nil {
				return err
			}
			profile, ok := file.Profiles[args[0]]
			if !ok {
				return apperr.New(apperr.ExitUsage, "unknown profile %q", args[0])
			}
			body, err := json.Marshal(map[string]any{"name": args[0], "default": file.DefaultProfile == args[0], "profile": profile})
			if err != nil {
				return err
			}
			return output.JSON(r.deps.stdout, body, r.pretty)
		},
	}
}

func (r *runtime) newProfilesAddCommand() *cobra.Command {
	var endpoint, workspaceID, apiKeyRef string
	var allowed []string
	var maxRecipients int
	var setDefault bool
	command := &cobra.Command{
		Use:   "add NAME",
		Short: "Add or update a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			path, file, err := r.configFile()
			if err != nil {
				return err
			}
			name := strings.TrimSpace(args[0])
			if name == "" {
				return apperr.New(apperr.ExitUsage, "profile name is required")
			}
			file.Profiles[name] = config.Profile{
				Endpoint: endpoint, WorkspaceID: workspaceID, APIKeyRef: apiKeyRef,
				MaxRecipients: maxRecipients, AllowedTestRecipients: allowed,
			}
			if setDefault || file.DefaultProfile == "" {
				file.DefaultProfile = name
			}
			if err := config.Save(path, file); err != nil {
				return err
			}
			fmt.Fprintf(r.deps.stderr, "saved profile %q to %s\n", name, path)
			return nil
		},
	}
	command.Flags().StringVar(&endpoint, "endpoint", "", "Notifuse base URL")
	command.Flags().StringVar(&workspaceID, "workspace-id", "", "Notifuse workspace_id")
	command.Flags().StringVar(&apiKeyRef, "api-key-ref", "", "Secret reference: env:NAME, file:PATH, or keychain:SERVICE/ACCOUNT")
	command.Flags().StringArrayVar(&allowed, "allowed-test-recipient", nil, "Allowed internal test recipient; repeatable")
	command.Flags().IntVar(&maxRecipients, "max-recipients", 100, "Maximum recipients allowed by policy")
	command.Flags().BoolVar(&setDefault, "default", false, "Set as default profile")
	_ = command.MarkFlagRequired("endpoint")
	_ = command.MarkFlagRequired("workspace-id")
	_ = command.MarkFlagRequired("api-key-ref")
	return command
}

func (r *runtime) newProfilesDefaultCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "default NAME",
		Short: "Set the default profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			path, file, err := r.configFile()
			if err != nil {
				return err
			}
			if _, ok := file.Profiles[args[0]]; !ok {
				return apperr.New(apperr.ExitUsage, "unknown profile %q", args[0])
			}
			file.DefaultProfile = args[0]
			return config.Save(path, file)
		},
	}
}
