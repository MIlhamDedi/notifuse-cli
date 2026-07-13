package cmd

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/milhamdedi/notifuse-cli/internal/apperr"
	"github.com/milhamdedi/notifuse-cli/internal/client"
	"github.com/milhamdedi/notifuse-cli/internal/input"
	"github.com/milhamdedi/notifuse-cli/internal/output"
	"github.com/spf13/cobra"
)

func (r *runtime) newContactsCommand() *cobra.Command {
	command := &cobra.Command{Use: "contacts", Short: "Manage contacts safely"}
	command.AddCommand(
		r.newListLikeCommand("list", "List contacts", "/api/contacts.list"),
		r.newListLikeCommand("count", "Count contacts", "/api/contacts.count"),
		r.newPostBodyCommand("upsert", "Upsert one contact", "/api/contacts.upsert"),
	)
	return command
}

func (r *runtime) newTemplatesCommand() *cobra.Command {
	command := &cobra.Command{Use: "templates", Short: "Manage and compile templates"}
	command.AddCommand(
		r.newListLikeCommand("list", "List templates", "/api/templates.list"),
		r.newListLikeCommand("get", "Get a template", "/api/templates.get"),
		r.newPostBodyCommand("create", "Create a template", "/api/templates.create"),
		r.newPostBodyCommand("update", "Update a template", "/api/templates.update"),
		r.newPostBodyCommand("compile", "Compile a template", "/api/templates.compile"),
	)
	return command
}

func (r *runtime) newBroadcastsCommand() *cobra.Command {
	command := &cobra.Command{Use: "broadcasts", Short: "Create drafts and inspect broadcasts"}
	command.AddCommand(
		r.newListLikeCommand("list", "List broadcasts", "/api/broadcasts.list"),
		r.newListLikeCommand("get", "Get a broadcast", "/api/broadcasts.get"),
		r.newPostBodyCommand("create-draft", "Create a broadcast draft", "/api/broadcasts.create"),
		r.newBroadcastTestSendCommand(),
		r.blockedCommand("schedule", "Broadcast scheduling is intentionally blocked by this CLI"),
		r.blockedCommand("resume", "Broadcast resume is intentionally blocked by this CLI"),
		r.blockedCommand("send", "Production broadcast sending is intentionally blocked by this CLI"),
	)
	return command
}

func (r *runtime) newTransactionalCommand() *cobra.Command {
	command := &cobra.Command{Use: "transactional", Short: "Transactional email helpers"}
	command.AddCommand(r.blockedCommand("send", "Transactional sending is blocked until an explicit policy is configured"))
	return command
}

func (r *runtime) newListLikeCommand(use, short, path string) *cobra.Command {
	var queries []string
	command := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			_, profile, apiClient, err := r.apiClient()
			if err != nil {
				return err
			}
			query, err := withWorkspaceQuery(profile, queries)
			if err != nil {
				return err
			}
			response, err := apiClient.Do(command.Context(), client.Request{Method: http.MethodGet, Path: path, Query: query})
			if err != nil {
				return err
			}
			if statusErr := apperr.FromStatus(response.Status); statusErr != nil {
				writeUpstreamError(r.deps.stderr, response.Body, r.pretty)
				return statusErr
			}
			return outputJSON(r, response.Body)
		},
	}
	command.Flags().StringArrayVar(&queries, "query", nil, "Query KEY=VALUE; repeatable")
	return command
}

func (r *runtime) newPostBodyCommand(use, short, path string) *cobra.Command {
	var bodyJSON, bodyFile string
	var dryRun bool
	command := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			profileName, profile, apiClient, err := r.apiClient()
			if err != nil {
				return err
			}
			body, err := input.JSON(bodyJSON, bodyFile, r.deps.stdin)
			if err != nil {
				return err
			}
			body, err = withWorkspaceBody(profile, body)
			if err != nil {
				return err
			}
			if dryRun {
				return dryRunResponse(r.deps.stdout, profileName, profile, http.MethodPost, path, url.Values{}, body, r.pretty)
			}
			response, err := apiClient.Do(command.Context(), client.Request{Method: http.MethodPost, Path: path, Body: body, ContentType: "application/json"})
			if err != nil {
				return err
			}
			if statusErr := apperr.FromStatus(response.Status); statusErr != nil {
				writeUpstreamError(r.deps.stderr, response.Body, r.pretty)
				return statusErr
			}
			return outputJSON(r, response.Body)
		},
	}
	addBodyFlags(command, &bodyJSON, &bodyFile)
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Print request without sending")
	return command
}

func (r *runtime) newBroadcastTestSendCommand() *cobra.Command {
	var bodyJSON, bodyFile string
	var dryRun bool
	command := &cobra.Command{
		Use:   "test-send",
		Short: "Send one broadcast test to an allowlisted recipient",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			profileName, profile, apiClient, err := r.apiClient()
			if err != nil {
				return err
			}
			body, err := input.JSON(bodyJSON, bodyFile, r.deps.stdin)
			if err != nil {
				return err
			}
			object, err := input.Object(body)
			if err != nil {
				return err
			}
			email := firstString(object, "email", "recipient_email", "to")
			if email == "" {
				return apperr.New(apperr.ExitUsage, "test-send body must include email, recipient_email, or to")
			}
			if !profile.AllowsRecipient(email) {
				return apperr.New(apperr.ExitUsage, "recipient %s is not in allowed_test_recipients for profile %s", email, profileName)
			}
			body, err = withWorkspaceBody(profile, body)
			if err != nil {
				return err
			}
			const path = "/api/broadcasts.sendToIndividual"
			if dryRun {
				return dryRunResponse(r.deps.stdout, profileName, profile, http.MethodPost, path, url.Values{}, body, r.pretty)
			}
			response, err := apiClient.Do(command.Context(), client.Request{Method: http.MethodPost, Path: path, Body: body, ContentType: "application/json"})
			if err != nil {
				return err
			}
			if statusErr := apperr.FromStatus(response.Status); statusErr != nil {
				writeUpstreamError(r.deps.stderr, response.Body, r.pretty)
				return statusErr
			}
			return outputJSON(r, response.Body)
		},
	}
	addBodyFlags(command, &bodyJSON, &bodyFile)
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Print request without sending")
	return command
}

func (r *runtime) blockedCommand(use, message string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: message,
		RunE: func(_ *cobra.Command, _ []string) error {
			return apperr.New(apperr.ExitUsage, "%s", message)
		},
	}
}

func firstString(object map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := object[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func outputJSON(r *runtime, body []byte) error {
	return output.JSON(r.deps.stdout, body, r.pretty)
}

func addBodyFlags(command *cobra.Command, bodyJSON, bodyFile *string) {
	command.Flags().StringVar(bodyJSON, "body-json", "", "Inline JSON request body")
	command.Flags().StringVar(bodyFile, "body-file", "", "JSON request body file, or - for stdin")
	command.MarkFlagsMutuallyExclusive("body-json", "body-file")
}
