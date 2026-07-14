package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/milhamdedi/notifuse-cli/internal/apperr"
	"github.com/milhamdedi/notifuse-cli/internal/client"
	"github.com/milhamdedi/notifuse-cli/internal/config"
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
		r.newTemplateFileCommand("create-from-file", "Create a template from a JSON file", "/api/templates.create"),
		r.newPostBodyCommand("update", "Update a template", "/api/templates.update"),
		r.newTemplateFileCommand("update-from-file", "Update a template from a JSON file", "/api/templates.update"),
		r.newPostBodyCommand("compile", "Compile a template", "/api/templates.compile"),
		r.newTemplateCompileFromFileCommand(),
		r.newTemplateValidateFileCommand(),
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
	command.AddCommand(
		r.newTransactionalTestSendCommand(),
		r.blockedCommand("send", "Production transactional sending is intentionally blocked by this CLI"),
	)
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

func (r *runtime) newTemplateFileCommand(use, short, path string) *cobra.Command {
	var file string
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
			body, err := input.JSON("", file, r.deps.stdin)
			if err != nil {
				return err
			}
			if err := validateTemplateBody(body); err != nil {
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
	command.Flags().StringVar(&file, "file", "", "Template JSON file, or - for stdin")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Print request without sending")
	command.MarkFlagRequired("file")
	return command
}

func (r *runtime) newTemplateCompileFromFileCommand() *cobra.Command {
	var file, dataJSON, dataFile, messageID string
	var dryRun bool
	command := &cobra.Command{
		Use:   "compile-from-file",
		Short: "Compile a template JSON file by extracting email.visual_editor_tree",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			profileName, profile, apiClient, err := r.apiClient()
			if err != nil {
				return err
			}
			body, err := input.JSON("", file, r.deps.stdin)
			if err != nil {
				return err
			}
			compileBody, err := compileBodyFromTemplate(body, messageID, dataJSON, dataFile, r.deps.stdin)
			if err != nil {
				return err
			}
			compileBody, err = withWorkspaceBody(profile, compileBody)
			if err != nil {
				return err
			}
			const path = "/api/templates.compile"
			if dryRun {
				return dryRunResponse(r.deps.stdout, profileName, profile, http.MethodPost, path, url.Values{}, compileBody, r.pretty)
			}
			response, err := apiClient.Do(command.Context(), client.Request{Method: http.MethodPost, Path: path, Body: compileBody, ContentType: "application/json"})
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
	command.Flags().StringVar(&file, "file", "", "Template JSON file, or - for stdin")
	command.Flags().StringVar(&messageID, "message-id", "", "Message id for compile request; defaults to template id or preview")
	command.Flags().StringVar(&dataJSON, "data-json", "", "Override template test_data with inline JSON")
	command.Flags().StringVar(&dataFile, "data-file", "", "Override template test_data with JSON file")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Print request without sending")
	command.MarkFlagRequired("file")
	command.MarkFlagsMutuallyExclusive("data-json", "data-file")
	return command
}

func (r *runtime) newTemplateValidateFileCommand() *cobra.Command {
	var file string
	command := &cobra.Command{
		Use:   "validate-file",
		Short: "Validate a Notifuse template JSON file locally",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			body, err := input.JSON("", file, r.deps.stdin)
			if err != nil {
				return err
			}
			if err := validateTemplateBody(body); err != nil {
				return err
			}
			result, err := json.Marshal(map[string]any{"valid": true})
			if err != nil {
				return err
			}
			return output.JSON(r.deps.stdout, result, r.pretty)
		},
	}
	command.Flags().StringVar(&file, "file", "", "Template JSON file, or - for stdin")
	command.MarkFlagRequired("file")
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

func (r *runtime) newTransactionalTestSendCommand() *cobra.Command {
	var bodyJSON, bodyFile string
	var dryRun bool
	command := &cobra.Command{
		Use:   "test-send",
		Short: "Send one transactional test email to an allowlisted recipient",
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
			if err := validateTransactionalTestRecipient(profileName, profile, body); err != nil {
				return err
			}
			body, err = withWorkspaceBody(profile, body)
			if err != nil {
				return err
			}
			const path = "/api/transactional.send"
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

func nestedObject(object map[string]any, keys ...string) map[string]any {
	current := object
	for _, key := range keys {
		next, ok := current[key].(map[string]any)
		if !ok {
			return nil
		}
		current = next
	}
	return current
}

func validateTemplateBody(body []byte) error {
	object, err := input.Object(body)
	if err != nil {
		return err
	}
	for _, key := range []string{"id", "name", "channel", "category"} {
		if strings.TrimSpace(fmt.Sprint(object[key])) == "" {
			return apperr.New(apperr.ExitUsage, "template %s is required", key)
		}
	}
	channel := strings.TrimSpace(fmt.Sprint(object["channel"]))
	switch channel {
	case "email":
		email := nestedObject(object, "email")
		if email == nil {
			return apperr.New(apperr.ExitUsage, "email template requires email object")
		}
		for _, key := range []string{"subject", "compiled_preview", "visual_editor_tree"} {
			if _, ok := email[key]; !ok {
				return apperr.New(apperr.ExitUsage, "email template requires email.%s", key)
			}
		}
		if err := validateVisualEditorTree(email["visual_editor_tree"]); err != nil {
			return err
		}
	case "web":
		if nestedObject(object, "web") == nil {
			return apperr.New(apperr.ExitUsage, "web template requires web object")
		}
	default:
		return apperr.New(apperr.ExitUsage, "template channel must be email or web")
	}
	return nil
}

func compileBodyFromTemplate(body []byte, messageID string, dataJSON string, dataFile string, stdin io.Reader) ([]byte, error) {
	object, err := input.Object(body)
	if err != nil {
		return nil, err
	}
	email := nestedObject(object, "email")
	if email == nil {
		if _, ok := object["visual_editor_tree"]; ok {
			return withCompileOverrides(object, messageID, dataJSON, dataFile, stdin)
		}
		return nil, apperr.New(apperr.ExitUsage, "template file must contain email.visual_editor_tree or be a compile request")
	}
	if _, ok := email["visual_editor_tree"]; !ok {
		return nil, apperr.New(apperr.ExitUsage, "email.visual_editor_tree is required")
	}
	if err := validateVisualEditorTree(email["visual_editor_tree"]); err != nil {
		return nil, err
	}
	if strings.TrimSpace(messageID) == "" {
		messageID = strings.TrimSpace(fmt.Sprint(object["id"]))
	}
	if strings.TrimSpace(messageID) == "" {
		messageID = "preview"
	}
	compile := map[string]any{
		"message_id":         messageID,
		"visual_editor_tree": email["visual_editor_tree"],
		"channel":            "email",
	}
	if subject, ok := email["subject"]; ok {
		compile["subject"] = subject
	}
	if preview, ok := email["subject_preview"]; ok {
		compile["subject_preview"] = preview
	}
	if testData, ok := object["test_data"]; ok {
		compile["test_data"] = testData
	}
	return withCompileOverrides(compile, messageID, dataJSON, dataFile, stdin)
}

func withCompileOverrides(object map[string]any, messageID string, dataJSON string, dataFile string, stdin io.Reader) ([]byte, error) {
	if strings.TrimSpace(messageID) != "" {
		object["message_id"] = strings.TrimSpace(messageID)
	}
	if _, ok := object["message_id"]; !ok || strings.TrimSpace(fmt.Sprint(object["message_id"])) == "" {
		object["message_id"] = "preview"
	}
	if dataJSON != "" || dataFile != "" {
		data, err := input.JSON(dataJSON, dataFile, stdin)
		if err != nil {
			return nil, err
		}
		dataObject, err := input.Object(data)
		if err != nil {
			return nil, err
		}
		object["test_data"] = dataObject
	}
	return json.Marshal(object)
}

func validateTransactionalTestRecipient(profileName string, profile config.Profile, body []byte) error {
	object, err := input.Object(body)
	if err != nil {
		return err
	}
	notification := nestedObject(object, "notification")
	if notification == nil {
		return apperr.New(apperr.ExitUsage, "transactional test-send body must include notification object")
	}
	contact := nestedObject(notification, "contact")
	if contact == nil {
		return apperr.New(apperr.ExitUsage, "transactional test-send body must include notification.contact object")
	}
	email := firstString(contact, "email")
	if email == "" {
		return apperr.New(apperr.ExitUsage, "transactional test-send body must include notification.contact.email")
	}
	if !profile.AllowsRecipient(email) {
		return apperr.New(apperr.ExitUsage, "recipient %s is not in allowed_test_recipients for profile %s", email, profileName)
	}
	return nil
}

func validateVisualEditorTree(value any) error {
	tree, ok := value.(map[string]any)
	if !ok {
		return apperr.New(apperr.ExitUsage, "email.visual_editor_tree must be an object")
	}
	if fmt.Sprint(tree["type"]) != "mjml" {
		return apperr.New(apperr.ExitUsage, "email.visual_editor_tree.type must be mjml")
	}
	children, ok := tree["children"].([]any)
	if !ok || len(children) == 0 {
		return apperr.New(apperr.ExitUsage, "email.visual_editor_tree.children must not be empty")
	}
	return nil
}

func outputJSON(r *runtime, body []byte) error {
	return output.JSON(r.deps.stdout, body, r.pretty)
}

func addBodyFlags(command *cobra.Command, bodyJSON, bodyFile *string) {
	command.Flags().StringVar(bodyJSON, "body-json", "", "Inline JSON request body")
	command.Flags().StringVar(bodyFile, "body-file", "", "JSON request body file, or - for stdin")
	command.MarkFlagsMutuallyExclusive("body-json", "body-file")
}
