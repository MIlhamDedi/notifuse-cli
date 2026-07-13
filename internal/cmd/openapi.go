package cmd

import (
	"encoding/json"
	"strings"

	"github.com/milhamdedi/notifuse-cli/internal/openapi"
	"github.com/milhamdedi/notifuse-cli/internal/output"
	"github.com/spf13/cobra"
)

func (r *runtime) newOpenAPICommand() *cobra.Command {
	command := &cobra.Command{Use: "openapi", Short: "Inspect embedded Notifuse OpenAPI metadata"}
	command.AddCommand(r.newOpenAPIListCommand())
	return command
}

func (r *runtime) newOpenAPIListCommand() *cobra.Command {
	var filter string
	command := &cobra.Command{
		Use:   "list",
		Short: "List embedded API operations",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			doc, err := openapi.Load(r.spec)
			if err != nil {
				return err
			}
			var entries []openapi.Entry
			filter = strings.ToLower(strings.TrimSpace(filter))
			for _, entry := range doc.Entries() {
				haystack := strings.ToLower(entry.Method + " " + entry.Path + " " + entry.OperationID + " " + entry.Summary)
				if filter == "" || strings.Contains(haystack, filter) {
					entries = append(entries, entry)
				}
			}
			body, err := json.Marshal(map[string]any{"title": doc.Info.Title, "version": doc.Info.Version, "operations": entries})
			if err != nil {
				return err
			}
			return output.JSON(r.deps.stdout, body, r.pretty)
		},
	}
	command.Flags().StringVar(&filter, "filter", "", "Filter operations by text")
	return command
}
