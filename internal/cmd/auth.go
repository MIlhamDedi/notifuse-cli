package cmd

import (
	"fmt"
	"strings"

	"github.com/milhamdedi/notifuse-cli/internal/apperr"
	"github.com/milhamdedi/notifuse-cli/internal/secret"
	"github.com/spf13/cobra"
)

func (r *runtime) newAuthCommand() *cobra.Command {
	command := &cobra.Command{Use: "auth", Short: "Manage profile API-key storage"}
	command.AddCommand(r.newAuthLoginCommand())
	return command
}

func (r *runtime) newAuthLoginCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "login PROFILE",
		Short: "Prompt for an API key and store it when the profile uses keychain:",
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
			if !strings.HasPrefix(profile.APIKeyRef, "keychain:") {
				return apperr.New(apperr.ExitUsage, "auth login only supports keychain: refs; profile uses %q", profile.APIKeyRef)
			}
			apiKey, err := secret.ReadAPIKeyFromTerminal("Notifuse API key: ")
			if err != nil {
				return err
			}
			if err := secret.StoreKeychain(strings.TrimPrefix(profile.APIKeyRef, "keychain:"), apiKey); err != nil {
				return err
			}
			fmt.Fprintf(r.deps.stderr, "stored API key for profile %q\n", args[0])
			return nil
		},
	}
}
