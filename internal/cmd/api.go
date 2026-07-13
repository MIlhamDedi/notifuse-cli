package cmd

import (
	"net/http"
	"strings"

	"github.com/milhamdedi/notifuse-cli/internal/apperr"
	"github.com/milhamdedi/notifuse-cli/internal/client"
	"github.com/milhamdedi/notifuse-cli/internal/input"
	"github.com/spf13/cobra"
)

var allowedAPIPaths = map[string]bool{
	"/api/contacts.list":     true,
	"/api/contacts.count":    true,
	"/api/contacts.upsert":   true,
	"/api/templates.list":    true,
	"/api/templates.get":     true,
	"/api/templates.create":  true,
	"/api/templates.update":  true,
	"/api/templates.compile": true,
	"/api/broadcasts.list":   true,
	"/api/broadcasts.get":    true,
	"/api/broadcasts.create": true,
}

func (r *runtime) newAPICommand() *cobra.Command {
	command := &cobra.Command{Use: "api", Short: "Call allowlisted Notifuse API endpoints"}
	command.AddCommand(r.newAPIGetCommand(), r.newAPIPostCommand())
	return command
}

func (r *runtime) newAPIGetCommand() *cobra.Command {
	var queries []string
	command := &cobra.Command{
		Use:   "get PATH",
		Short: "GET an allowlisted endpoint with workspace_id injected",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			path := normalizeAPIPath(args[0])
			if !allowedAPIPaths[path] {
				return apperr.New(apperr.ExitUsage, "endpoint %s is not allowlisted", path)
			}
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

func (r *runtime) newAPIPostCommand() *cobra.Command {
	var bodyJSON, bodyFile string
	var dryRun bool
	command := &cobra.Command{
		Use:   "post PATH",
		Short: "POST JSON to an allowlisted endpoint with workspace_id injected",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			path := normalizeAPIPath(args[0])
			if !allowedAPIPaths[path] {
				return apperr.New(apperr.ExitUsage, "endpoint %s is not allowlisted", path)
			}
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
				return dryRunResponse(r.deps.stdout, profileName, profile, http.MethodPost, path, nil, body, r.pretty)
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

func normalizeAPIPath(path string) string {
	path = strings.TrimSpace(path)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}
