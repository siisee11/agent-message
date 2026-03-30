package ralphloop

import "fmt"

type schemaField struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Type        string        `json:"type"`
	Required    bool          `json:"required,omitempty"`
	Default     any           `json:"default,omitempty"`
	Enum        []string      `json:"enum,omitempty"`
	Alias       []string      `json:"alias,omitempty"`
	Schema      []schemaField `json:"schema,omitempty"`
}

type commandSchema struct {
	Command        string        `json:"command"`
	Description    string        `json:"description"`
	MutatesState   bool          `json:"mutates_state"`
	SupportsDryRun bool          `json:"supports_dry_run"`
	Positionals    []schemaField `json:"positionals"`
	Options        []schemaField `json:"options"`
	RawPayload     []schemaField `json:"raw_payload_schema"`
}

func commandSchemas() []commandSchema {
	descriptors := commandDescriptors()
	schemas := make([]commandSchema, 0, len(descriptors))
	for _, descriptor := range descriptors {
		schemas = append(schemas, commandSchemaFromDescriptor(descriptor))
	}
	return schemas
}

func executeSchemaCommand(runCtx runContext) int {
	items := make([]map[string]any, 0, len(commandSchemas()))
	for _, schema := range commandSchemas() {
		if runCtx.command.SchemaOptions.Command != "" && runCtx.command.SchemaOptions.Command != schema.Command {
			continue
		}
		items = append(items, schemaToMap(schema))
	}
	if runCtx.command.SchemaOptions.Command != "" && len(items) == 0 {
		return writeCommandError(runCtx.stdout, runCtx.stderr, runCtx.command.Common.Output, string(runCtx.command.Kind), fmt.Errorf("unknown command schema: %s", runCtx.command.SchemaOptions.Command))
	}
	items = applyFieldMask(items, runCtx.command.Common.Fields)
	pages := paginateItems("schema", items, runCtx.command.Common)
	if runCtx.command.Common.Output == OutputText {
		return renderSchemaText(runCtx, pages)
	}
	if runCtx.command.Common.Output == OutputNDJSON {
		lines := make([]map[string]any, 0, len(pages))
		for _, page := range pages {
			lines = append(lines, envelopeToMap(page))
		}
		return writeCommandResult(runCtx, lines)
	}
	if runCtx.command.Common.PageAll {
		return writeCommandResult(runCtx, map[string]any{
			"command": "schema",
			"status":  "ok",
			"pages":   pages,
		})
	}
	return writeCommandResult(runCtx, envelopeToMap(pages[0]))
}

func schemaToMap(schema commandSchema) map[string]any {
	return map[string]any{
		"command":          schema.Command,
		"description":      schema.Description,
		"mutates_state":    schema.MutatesState,
		"supports_dry_run": schema.SupportsDryRun,
		"positionals":      schema.Positionals,
		"options":          schema.Options,
		"raw_payload":      schema.RawPayload,
	}
}

func envelopeToMap(page pageEnvelope) map[string]any {
	return map[string]any{
		"command":     page.Command,
		"status":      page.Status,
		"page":        page.Page,
		"page_size":   page.PageSize,
		"page_all":    page.PageAll,
		"total_items": page.TotalItems,
		"total_pages": page.TotalPages,
		"items":       page.Items,
	}
}

func renderSchemaText(runCtx runContext, pages []pageEnvelope) int {
	if len(pages) == 0 || len(pages[0].Items) == 0 {
		_, _ = fmt.Fprintln(runCtx.stdout, "No schema entries matched.")
		return 0
	}
	for _, item := range pages[0].Items {
		_, _ = fmt.Fprintf(runCtx.stdout, "%s: %v\n", item["command"], item["description"])
	}
	return 0
}
