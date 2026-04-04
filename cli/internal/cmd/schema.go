package cmd

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

type schemaDocument struct {
	SchemaVersion string          `json:"schema_version"`
	Binary        string          `json:"binary"`
	Commands      []commandSchema `json:"commands"`
}

type commandSchema struct {
	Name              string                `json:"name"`
	Path              string                `json:"path"`
	Use               string                `json:"use"`
	Summary           string                `json:"summary,omitempty"`
	Subcommands       []string              `json:"subcommands,omitempty"`
	Prerequisites     []string              `json:"prerequisites,omitempty"`
	InheritedFlags    []schemaFlag          `json:"inherited_flags,omitempty"`
	PersistentFlags   []schemaFlag          `json:"persistent_flags,omitempty"`
	LocalFlags        []schemaFlag          `json:"local_flags,omitempty"`
	PositionalArgs    []schemaArgument      `json:"positional_args,omitempty"`
	InputModes        []commandInputMode    `json:"input_modes,omitempty"`
	RequestSchemas    []commandRequestShape `json:"request_schemas,omitempty"`
	OutputFormats     []string              `json:"output_formats,omitempty"`
	Examples          []string              `json:"examples,omitempty"`
	Notes             []string              `json:"notes,omitempty"`
	SupportsJSONHelp  bool                  `json:"supports_json_help"`
	SupportsJSONValue bool                  `json:"supports_json_value,omitempty"`
}

type schemaArgument struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Required    bool     `json:"required"`
	Variadic    bool     `json:"variadic,omitempty"`
	Description string   `json:"description,omitempty"`
	Pattern     string   `json:"pattern,omitempty"`
	Enum        []string `json:"enum,omitempty"`
	Constraints []string `json:"constraints,omitempty"`
}

type schemaFlag struct {
	Name        string   `json:"name"`
	Shorthand   string   `json:"shorthand,omitempty"`
	Type        string   `json:"type"`
	Required    bool     `json:"required,omitempty"`
	Default     any      `json:"default,omitempty"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
	Pattern     string   `json:"pattern,omitempty"`
	Minimum     *int     `json:"minimum,omitempty"`
	Maximum     *int     `json:"maximum,omitempty"`
	Constraints []string `json:"constraints,omitempty"`
}

type commandInputMode struct {
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Conditions   []string          `json:"conditions,omitempty"`
	RequestShape *schemaValueShape `json:"request_shape,omitempty"`
	References   []schemaReference `json:"references,omitempty"`
}

type commandRequestShape struct {
	Name        string            `json:"name"`
	ContentType string            `json:"content_type,omitempty"`
	Description string            `json:"description,omitempty"`
	Shape       schemaValueShape  `json:"shape"`
	References  []schemaReference `json:"references,omitempty"`
}

type schemaReference struct {
	Kind        string `json:"kind"`
	Command     string `json:"command"`
	Description string `json:"description,omitempty"`
}

type schemaValueShape struct {
	Type                 string                      `json:"type,omitempty"`
	Description          string                      `json:"description,omitempty"`
	Enum                 []string                    `json:"enum,omitempty"`
	Const                any                         `json:"const,omitempty"`
	Pattern              string                      `json:"pattern,omitempty"`
	Format               string                      `json:"format,omitempty"`
	Minimum              *int                        `json:"minimum,omitempty"`
	Maximum              *int                        `json:"maximum,omitempty"`
	MinLength            *int                        `json:"min_length,omitempty"`
	MaxLength            *int                        `json:"max_length,omitempty"`
	Required             []string                    `json:"required,omitempty"`
	Properties           map[string]schemaValueShape `json:"properties,omitempty"`
	Items                *schemaValueShape           `json:"items,omitempty"`
	OneOf                []schemaValueShape          `json:"one_of,omitempty"`
	AdditionalProperties *bool                       `json:"additional_properties,omitempty"`
}

type parameterMetadata struct {
	Description string
	Pattern     string
	Enum        []string
	Minimum     *int
	Maximum     *int
	Constraints []string
	Required    bool
}

type commandSchemaDescriptor struct {
	Arguments     map[string]parameterMetadata
	Flags         map[string]parameterMetadata
	Prerequisites []string
	InputModes    []commandInputMode
	RequestShapes []commandRequestShape
	OutputFormats []string
	Examples      []string
	Notes         []string
}

func newSchemaCommand(rt *Runtime, rootProvider func() *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:   "schema [command-path...]",
		Short: "Print machine-readable command schemas resolved from the current CLI binary",
		RunE: func(_ *cobra.Command, args []string) error {
			return runSchema(rt, rootProvider, args)
		},
	}
}

func runSchema(rt *Runtime, rootProvider func() *cobra.Command, args []string) error {
	if err := ensureRuntime(rt); err != nil {
		return err
	}
	if rootProvider == nil {
		return errors.New("schema root provider is not configured")
	}

	root := rootProvider()
	if root == nil {
		return errors.New("schema root command is not initialized")
	}

	document, err := buildSchemaDocument(root, args)
	if err != nil {
		return err
	}
	return writeJSON(rt.Stdout, document)
}

func buildSchemaDocument(root *cobra.Command, targetPath []string) (schemaDocument, error) {
	if root == nil {
		return schemaDocument{}, errors.New("root command is required")
	}

	target, err := findCommandByPath(root, targetPath)
	if err != nil {
		return schemaDocument{}, err
	}

	commands := collectCommandSchemas(target)
	return schemaDocument{
		SchemaVersion: "2026-04-04",
		Binary:        root.Name(),
		Commands:      commands,
	}, nil
}

func findCommandByPath(root *cobra.Command, path []string) (*cobra.Command, error) {
	if root == nil {
		return nil, errors.New("root command is required")
	}
	if len(path) == 0 {
		return root, nil
	}

	segments := append([]string(nil), path...)
	if len(segments) > 0 && segments[0] == root.Name() {
		segments = segments[1:]
	}

	current := root
	for _, segment := range segments {
		found := false
		for _, child := range sortedChildren(current) {
			if child.Name() == segment {
				current = child
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("unknown command path: %s", strings.Join(path, " "))
		}
	}
	return current, nil
}

func collectCommandSchemas(root *cobra.Command) []commandSchema {
	if root == nil {
		return nil
	}

	paths := make([]*cobra.Command, 0, 16)
	var walk func(*cobra.Command)
	walk = func(command *cobra.Command) {
		paths = append(paths, command)
		for _, child := range sortedChildren(command) {
			walk(child)
		}
	}
	walk(root)

	schemas := make([]commandSchema, 0, len(paths))
	for _, command := range paths {
		schemas = append(schemas, buildCommandSchema(command))
	}
	return schemas
}

func buildCommandSchema(command *cobra.Command) commandSchema {
	path := command.CommandPath()
	descriptor := schemaRegistry()[path]
	globalFlags := globalFlagMetadata()
	inherited := buildSchemaFlags(command.InheritedPersistentFlagInfos(), globalFlags)
	persistent := buildSchemaFlags(command.PersistentFlagInfos(), mergeParameterMetadata(globalFlags, descriptor.Flags))
	local := buildSchemaFlags(command.LocalFlagInfos(), mergeParameterMetadata(globalFlags, descriptor.Flags))

	schema := commandSchema{
		Name:              command.Name(),
		Path:              path,
		Use:               strings.TrimSpace(command.Use),
		Summary:           strings.TrimSpace(command.Short),
		Subcommands:       childNames(command),
		Prerequisites:     append([]string(nil), descriptor.Prerequisites...),
		InheritedFlags:    inherited,
		PersistentFlags:   persistent,
		LocalFlags:        local,
		PositionalArgs:    buildSchemaArguments(command, descriptor.Arguments),
		InputModes:        append([]commandInputMode(nil), descriptor.InputModes...),
		RequestSchemas:    append([]commandRequestShape(nil), descriptor.RequestShapes...),
		OutputFormats:     append([]string(nil), descriptor.OutputFormats...),
		Examples:          append([]string(nil), descriptor.Examples...),
		Notes:             append([]string(nil), descriptor.Notes...),
		SupportsJSONHelp:  true,
		SupportsJSONValue: len(descriptor.OutputFormats) > 0,
	}
	return schema
}

func buildSchemaArguments(command *cobra.Command, metadata map[string]parameterMetadata) []schemaArgument {
	args := parseUsageArguments(command.Use)
	out := make([]schemaArgument, 0, len(args))
	for _, arg := range args {
		meta := metadata[arg.Name]
		if meta.Required {
			arg.Required = true
		}
		if strings.TrimSpace(meta.Description) != "" {
			arg.Description = meta.Description
		}
		if meta.Pattern != "" {
			arg.Pattern = meta.Pattern
		}
		if len(meta.Enum) > 0 {
			arg.Enum = append([]string(nil), meta.Enum...)
		}
		if len(meta.Constraints) > 0 {
			arg.Constraints = append([]string(nil), meta.Constraints...)
		}
		out = append(out, arg)
	}
	return out
}

func buildSchemaFlags(flags []cobra.FlagInfo, metadata map[string]parameterMetadata) []schemaFlag {
	out := make([]schemaFlag, 0, len(flags))
	for _, flag := range flags {
		meta := metadata[flag.Name]
		description := strings.TrimSpace(meta.Description)
		if description == "" {
			description = strings.TrimSpace(flag.Usage)
		}
		out = append(out, schemaFlag{
			Name:        flag.Name,
			Shorthand:   flag.Shorthand,
			Type:        flag.Type,
			Required:    meta.Required,
			Default:     flag.Default,
			Description: description,
			Enum:        append([]string(nil), meta.Enum...),
			Pattern:     meta.Pattern,
			Minimum:     meta.Minimum,
			Maximum:     meta.Maximum,
			Constraints: append([]string(nil), meta.Constraints...),
		})
	}
	return out
}

func parseUsageArguments(use string) []schemaArgument {
	fields := strings.Fields(strings.TrimSpace(use))
	if len(fields) <= 1 {
		return nil
	}

	args := make([]schemaArgument, 0, len(fields)-1)
	for _, token := range fields[1:] {
		required := strings.HasPrefix(token, "<") && strings.HasSuffix(token, ">")
		optional := strings.HasPrefix(token, "[") && strings.HasSuffix(token, "]")
		if !required && !optional {
			continue
		}

		name := strings.TrimSuffix(strings.TrimPrefix(strings.TrimSuffix(strings.TrimPrefix(token, "<"), ">"), "["), "]")
		variadic := strings.HasSuffix(name, "...")
		name = strings.TrimSuffix(name, "...")
		if strings.TrimSpace(name) == "" {
			continue
		}

		args = append(args, schemaArgument{
			Name:     name,
			Type:     "string",
			Required: required,
			Variadic: variadic,
		})
	}
	return args
}

func sortedChildren(command *cobra.Command) []*cobra.Command {
	children := command.Children()
	sort.Slice(children, func(i, j int) bool {
		return children[i].Name() < children[j].Name()
	})
	return children
}

func childNames(command *cobra.Command) []string {
	children := sortedChildren(command)
	if len(children) == 0 {
		return nil
	}
	names := make([]string, 0, len(children))
	for _, child := range children {
		names = append(names, child.Name())
	}
	return names
}

func schemaRegistry() map[string]commandSchemaDescriptor {
	usernameArg := parameterMetadata{
		Description: "Username accepted by the Agent Message server",
		Pattern:     `^[A-Za-z0-9._-]{3,32}$`,
		Constraints: []string{"must not contain control characters"},
	}
	messageIDArg := parameterMetadata{
		Description: "Explicit message ID or a numeric index from the last read session",
		Constraints: []string{"explicit message IDs reject control characters, dot segments, percent-encoded segments, and path/query syntax"},
	}
	indexFlag := parameterMetadata{
		Description: "Index from the most recent `agent-message read <username>` session",
		Minimum:     intPtr(1),
	}

	return map[string]commandSchemaDescriptor{
		"agent-message": {
			Flags: mergeParameterMetadata(globalFlagMetadata(), nil),
			Notes: []string{
				"This schema output is generated from the current command tree at runtime and enriched with typed command metadata.",
				"Use `agent-message schema <command>` to scope the output to one command path.",
			},
			Examples: []string{
				"agent-message schema",
				"agent-message schema send",
				"agent-message schema config set",
			},
		},
		"agent-message schema": {
			Arguments: map[string]parameterMetadata{
				"command-path": {Description: "Optional command path to narrow the schema output, for example `send` or `config set`"},
			},
			OutputFormats: []string{"json"},
			Notes: []string{
				"Without arguments it emits schemas for the full CLI tree.",
				"With a command path it emits that command and any nested subcommands.",
			},
			Examples: []string{
				"agent-message schema",
				"agent-message schema send",
				"agent-message schema profile switch",
			},
		},
		"agent-message catalog": {
			Notes: []string{
				"Catalog commands expose the server-side json_render contract that the CLI references for rich message payloads.",
			},
		},
		"agent-message catalog prompt": {
			Prerequisites: []string{"configured_server_url"},
			OutputFormats: []string{"text", "json"},
			Notes: []string{
				"Returns the live server prompt that defines the current json_render component catalog and patch contract.",
			},
			Examples: []string{
				"agent-message catalog prompt",
				"agent-message catalog prompt --json",
			},
		},
		"agent-message config": {},
		"agent-message config path": {
			OutputFormats: []string{"text", "json"},
		},
		"agent-message config get": {
			Arguments: map[string]parameterMetadata{
				"key": {Description: "Optional config key to fetch; omit to print the full config", Enum: []string{"master", "server_url"}},
			},
			OutputFormats: []string{"text", "json"},
		},
		"agent-message config set": {
			Arguments: map[string]parameterMetadata{
				"key":   {Description: "Config key to update", Enum: []string{"master", "server_url"}},
				"value": {Description: "New value to persist for the selected config key"},
			},
			OutputFormats: []string{"text", "json"},
		},
		"agent-message config unset": {
			Arguments: map[string]parameterMetadata{
				"key": {Description: "Config key to clear", Enum: []string{"master", "server_url"}},
			},
			OutputFormats: []string{"text", "json"},
		},
		"agent-message profile": {},
		"agent-message profile list": {
			OutputFormats: []string{"text", "json"},
		},
		"agent-message profile current": {
			OutputFormats: []string{"text", "json"},
		},
		"agent-message profile switch": {
			Arguments: map[string]parameterMetadata{
				"profile": {Description: "Saved profile name to activate for subsequent commands"},
			},
			OutputFormats: []string{"text", "json"},
		},
		"agent-message onboard": {
			Prerequisites: []string{"configured_server_url"},
			OutputFormats: []string{"text", "json"},
			Notes: []string{
				"Interactive command that reads username and password from stdin prompts.",
			},
		},
		"agent-message register": {
			Arguments: map[string]parameterMetadata{
				"username": usernameArg,
				"password": {Description: "Password or PIN string sent to the auth API"},
			},
			Flags: map[string]parameterMetadata{
				"payload":       {Description: "Inline raw JSON payload matching the register request body"},
				"payload-file":  {Description: "Read the raw register JSON payload from a file"},
				"payload-stdin": {Description: "Read the raw register JSON payload from stdin"},
			},
			Prerequisites: []string{"configured_server_url"},
			InputModes: []commandInputMode{
				{
					Name:        "positional_fields",
					Description: "Provide username and password as positional arguments",
					Conditions:  []string{"requires `<username> <password>`"},
					RequestShape: shapePtr(requestObjectShape("Register request body", map[string]schemaValueShape{
						"username": stringShape("Username accepted by the server", withPattern(`^[A-Za-z0-9._-]{3,32}$`)),
						"password": stringShape("Password or PIN string"),
					}, "username", "password")),
				},
				{
					Name:        "raw_payload",
					Description: "Provide the register request body directly as JSON",
					Conditions:  []string{"choose only one of --payload, --payload-file, or --payload-stdin", "no positional args are allowed in this mode"},
					RequestShape: shapePtr(requestObjectShape("Register request body", map[string]schemaValueShape{
						"username": stringShape("Username accepted by the server", withPattern(`^[A-Za-z0-9._-]{3,32}$`)),
						"password": stringShape("Password or PIN string"),
					}, "username", "password")),
				},
			},
			RequestShapes: []commandRequestShape{
				{
					Name:        "auth_register_body",
					ContentType: "application/json",
					Description: "JSON body sent to POST /api/auth/register",
					Shape: objectShape("Register request body", map[string]schemaValueShape{
						"username": stringShape("Username accepted by the server", withPattern(`^[A-Za-z0-9._-]{3,32}$`)),
						"password": stringShape("Password or PIN string"),
					}, "username", "password"),
				},
			},
			OutputFormats: []string{"text", "json"},
			Examples: []string{
				"agent-message register alice secret123",
				"agent-message register --payload '{\"username\":\"alice\",\"password\":\"secret123\"}'",
			},
		},
		"agent-message login": {
			Arguments: map[string]parameterMetadata{
				"username": usernameArg,
				"password": {Description: "Password or PIN string sent to the auth API"},
			},
			Flags: map[string]parameterMetadata{
				"payload":       {Description: "Inline raw JSON payload matching the login request body"},
				"payload-file":  {Description: "Read the raw login JSON payload from a file"},
				"payload-stdin": {Description: "Read the raw login JSON payload from stdin"},
			},
			Prerequisites: []string{"configured_server_url"},
			InputModes: []commandInputMode{
				{
					Name:        "positional_fields",
					Description: "Provide username and password as positional arguments",
					Conditions:  []string{"requires `<username> <password>`"},
					RequestShape: shapePtr(requestObjectShape("Login request body", map[string]schemaValueShape{
						"username": stringShape("Username accepted by the server", withPattern(`^[A-Za-z0-9._-]{3,32}$`)),
						"password": stringShape("Password or PIN string"),
					}, "username", "password")),
				},
				{
					Name:        "raw_payload",
					Description: "Provide the login request body directly as JSON",
					Conditions:  []string{"choose only one of --payload, --payload-file, or --payload-stdin", "no positional args are allowed in this mode"},
					RequestShape: shapePtr(requestObjectShape("Login request body", map[string]schemaValueShape{
						"username": stringShape("Username accepted by the server", withPattern(`^[A-Za-z0-9._-]{3,32}$`)),
						"password": stringShape("Password or PIN string"),
					}, "username", "password")),
				},
			},
			RequestShapes: []commandRequestShape{
				{
					Name:        "auth_login_body",
					ContentType: "application/json",
					Description: "JSON body sent to POST /api/auth/login",
					Shape: objectShape("Login request body", map[string]schemaValueShape{
						"username": stringShape("Username accepted by the server", withPattern(`^[A-Za-z0-9._-]{3,32}$`)),
						"password": stringShape("Password or PIN string"),
					}, "username", "password"),
				},
			},
			OutputFormats: []string{"text", "json"},
			Examples: []string{
				"agent-message login alice secret123",
				"agent-message login --payload-file ./login.json",
			},
		},
		"agent-message logout": {
			Prerequisites: []string{"logged_in_for_remote_logout"},
			OutputFormats: []string{"text", "json"},
		},
		"agent-message whoami": {
			Prerequisites: []string{"logged_in"},
			OutputFormats: []string{"text", "json"},
		},
		"agent-message ls": {
			Prerequisites: []string{"logged_in"},
			OutputFormats: []string{"text", "json"},
		},
		"agent-message open": {
			Arguments: map[string]parameterMetadata{
				"username": usernameArg,
			},
			Flags: map[string]parameterMetadata{
				"payload":       {Description: "Inline raw JSON payload matching the open conversation request body"},
				"payload-file":  {Description: "Read the raw open conversation JSON payload from a file"},
				"payload-stdin": {Description: "Read the raw open conversation JSON payload from stdin"},
			},
			Prerequisites: []string{"logged_in"},
			InputModes: []commandInputMode{
				{
					Name:        "positional_username",
					Description: "Provide the target username as a positional argument",
					Conditions:  []string{"requires `<username>`"},
					RequestShape: shapePtr(requestObjectShape("Open conversation request body", map[string]schemaValueShape{
						"username": stringShape("Recipient username", withPattern(`^[A-Za-z0-9._-]{3,32}$`)),
					}, "username")),
				},
				{
					Name:        "raw_payload",
					Description: "Provide the open conversation request body directly as JSON",
					Conditions:  []string{"choose only one of --payload, --payload-file, or --payload-stdin", "no positional args are allowed in this mode"},
					RequestShape: shapePtr(requestObjectShape("Open conversation request body", map[string]schemaValueShape{
						"username": stringShape("Recipient username", withPattern(`^[A-Za-z0-9._-]{3,32}$`)),
					}, "username")),
				},
			},
			RequestShapes: []commandRequestShape{
				{
					Name:        "open_conversation_body",
					ContentType: "application/json",
					Description: "JSON body sent to POST /api/conversations",
					Shape: objectShape("Open conversation request body", map[string]schemaValueShape{
						"username": stringShape("Recipient username", withPattern(`^[A-Za-z0-9._-]{3,32}$`)),
					}, "username"),
				},
			},
			OutputFormats: []string{"text", "json"},
			Examples: []string{
				"agent-message open bob",
				"agent-message open --payload '{\"username\":\"bob\"}'",
			},
		},
		"agent-message send": {
			Arguments: map[string]parameterMetadata{
				"username":            {Description: "Optional recipient username when `--to` is omitted and no master is configured", Pattern: `^[A-Za-z0-9._-]{3,32}$`, Constraints: []string{"resolved against `--to`, `master`, and positional rules"}},
				"text-or-inline-json": {Description: "Either plain text or an inline json_render object depending on the resolved send mode"},
			},
			Flags: map[string]parameterMetadata{
				"to":               {Description: "Override recipient username", Pattern: `^[A-Za-z0-9._-]{3,32}$`},
				"kind":             {Description: "Message kind to send", Enum: []string{"text", "json_render"}},
				"attach":           {Description: "Path to a file or image to attach to a text message", Constraints: []string{"attachments are only supported with kind `text`", "path must not contain control characters"}},
				"text":             {Description: "Explicit text content source"},
				"json-render":      {Description: "Explicit inline json_render payload"},
				"json-render-file": {Description: "Read a json_render payload from a file path"},
				"payload":          {Description: "Inline raw JSON payload matching the send request body"},
				"payload-file":     {Description: "Read the raw send JSON payload from a file"},
				"payload-stdin":    {Description: "Read the raw send JSON payload from stdin"},
				"stdin":            {Description: "Read message content from stdin"},
			},
			Prerequisites: []string{"logged_in"},
			InputModes: []commandInputMode{
				{
					Name:        "text_message",
					Description: "Send a plain text message, optionally with a file attachment",
					Conditions: []string{
						"kind resolves to `text`",
						"exactly one explicit content source may be selected among positional content, --text, and --stdin",
						"--attach is allowed only in this mode",
					},
					RequestShape: shapePtr(requestObjectShape("Text message request", map[string]schemaValueShape{
						"content": stringShape("Non-empty text message body", withMinLength(1)),
					}, "content")),
				},
				{
					Name:        "json_render_message",
					Description: "Send a json_render message whose nested payload is a JSON object",
					Conditions: []string{
						"kind resolves to `json_render`",
						"content must come from an inline object, --json-render, --json-render-file, or --stdin",
						"attachments are not supported in this mode",
					},
					RequestShape: shapePtr(requestObjectShape("json_render send request", map[string]schemaValueShape{
						"kind":             enumShape("Message kind", "json_render"),
						"json_render_spec": objectShape("Renderer payload object accepted by the server", nil),
					}, "kind", "json_render_spec")),
					References: []schemaReference{
						{
							Kind:        "live_nested_schema",
							Command:     "agent-message catalog prompt",
							Description: "Fetch the current server-side json_render contract for the nested json_render_spec object.",
						},
					},
				},
				{
					Name:        "attachment_message",
					Description: "Send a multipart/form-data request with an attachment and optional text",
					Conditions: []string{
						"--attach is set",
						"kind must resolve to `text`",
					},
					RequestShape: &schemaValueShape{
						Type:        "object",
						Description: "Multipart form fields accepted by the attachment endpoint",
						Properties: map[string]schemaValueShape{
							"content":    stringShape("Optional text content that accompanies the attachment"),
							"attachment": stringShape("File path resolved locally and uploaded as multipart data"),
						},
						Required: []string{"attachment"},
					},
				},
				{
					Name:        "raw_payload",
					Description: "Provide the send request body directly as JSON while resolving the recipient separately",
					Conditions: []string{
						"choose only one of --payload, --payload-file, or --payload-stdin",
						"recipient resolution still uses `--to`, `master`, or the positional username",
						"raw payload cannot be combined with `--attach`, `--text`, `--json-render`, `--json-render-file`, or `--stdin`",
					},
					RequestShape: shapePtr(requestObjectShape("Raw send request body", map[string]schemaValueShape{
						"content":          stringShape("Optional text message body"),
						"kind":             enumShape("Message kind", "text", "json_render"),
						"json_render_spec": objectShape("Nested renderer payload object", nil),
					})),
					References: []schemaReference{
						{
							Kind:        "live_nested_schema",
							Command:     "agent-message catalog prompt",
							Description: "Use this command for the live nested json_render_spec contract when kind is `json_render`.",
						},
					},
				},
			},
			RequestShapes: []commandRequestShape{
				{
					Name:        "text_message_body",
					ContentType: "application/json",
					Description: "JSON body sent to POST /api/conversations/:id/messages for text sends",
					Shape: requestObjectShape("Text send body", map[string]schemaValueShape{
						"content": stringShape("Non-empty text message body", withMinLength(1)),
					}, "content"),
				},
				{
					Name:        "json_render_body",
					ContentType: "application/json",
					Description: "JSON body sent to POST /api/conversations/:id/messages for json_render sends",
					Shape: requestObjectShape("json_render body", map[string]schemaValueShape{
						"kind":             enumShape("Message kind", "json_render"),
						"json_render_spec": objectShape("Nested renderer payload object", nil),
					}, "kind", "json_render_spec"),
					References: []schemaReference{
						{
							Kind:        "live_nested_schema",
							Command:     "agent-message catalog prompt",
							Description: "Use this command to retrieve the current nested json_render contract.",
						},
					},
				},
				{
					Name:        "attachment_multipart_body",
					ContentType: "multipart/form-data",
					Description: "Multipart form body sent when `--attach` is used",
					Shape: schemaValueShape{
						Type: "object",
						Properties: map[string]schemaValueShape{
							"content":    stringShape("Optional text field"),
							"attachment": stringShape("Attachment path on disk"),
						},
						Required: []string{"attachment"},
					},
				},
			},
			OutputFormats: []string{"text", "json"},
			Examples: []string{
				"agent-message send jay \"hello\"",
				"agent-message send --to jay --text \"hello\"",
				"agent-message send jay '{\"root\":\"r1\",\"elements\":{}}' --kind json_render",
				"agent-message send jay --attach ./screenshot.png --text \"latest build\"",
				"agent-message send --to jay --payload '{\"kind\":\"json_render\",\"json_render_spec\":{\"root\":\"r1\",\"elements\":{}}}'",
			},
			Notes: []string{
				"Recipient resolution prefers `--to`, then the configured `master`, then positional username rules.",
				"Exactly one explicit content source is allowed among `--text`, `--json-render`, `--json-render-file`, and `--stdin`.",
				"Raw payload flags accept the request body directly and are mutually exclusive with the convenience content flags.",
			},
		},
		"agent-message read": {
			Arguments: map[string]parameterMetadata{
				"username": usernameArg,
			},
			Flags: map[string]parameterMetadata{
				"n": {Description: "Number of most recent messages to fetch", Minimum: intPtr(1)},
			},
			Prerequisites: []string{"logged_in"},
			OutputFormats: []string{"text", "json"},
			Notes: []string{
				"Successful reads persist a local index-to-message mapping that edit/delete/react/unreact can reuse.",
			},
		},
		"agent-message edit": {
			Arguments: map[string]parameterMetadata{
				"message-id-or-index": messageIDArg,
				"text":                {Description: "Replacement message text", Constraints: []string{"must be non-empty after trimming"}},
			},
			Flags: map[string]parameterMetadata{
				"message-id":    {Description: "Explicit message ID to edit"},
				"index":         indexFlag,
				"payload":       {Description: "Inline raw JSON payload matching the edit request body"},
				"payload-file":  {Description: "Read the raw edit JSON payload from a file"},
				"payload-stdin": {Description: "Read the raw edit JSON payload from stdin"},
			},
			Prerequisites: []string{"logged_in"},
			InputModes: []commandInputMode{
				{
					Name:        "selector_plus_text",
					Description: "Provide the selector and replacement text via convenience arguments",
					Conditions:  []string{"use `<message-id-or-index> <text>` or combine `--message-id`/`--index` with `<text>`"},
					RequestShape: shapePtr(requestObjectShape("Edit message body", map[string]schemaValueShape{
						"content": stringShape("Replacement message text", withMinLength(1)),
					}, "content")),
				},
				{
					Name:        "raw_payload",
					Description: "Provide the edit request body directly as JSON while resolving the selector separately",
					Conditions:  []string{"choose only one of --payload, --payload-file, or --payload-stdin", "selector still comes from the positional argument or `--message-id`/`--index`"},
					RequestShape: shapePtr(requestObjectShape("Edit message body", map[string]schemaValueShape{
						"content": stringShape("Replacement message text", withMinLength(1)),
					}, "content")),
				},
			},
			RequestShapes: []commandRequestShape{
				{
					Name:        "edit_message_body",
					ContentType: "application/json",
					Description: "JSON body sent to PATCH /api/messages/:id",
					Shape: requestObjectShape("Edit message body", map[string]schemaValueShape{
						"content": stringShape("Replacement message text", withMinLength(1)),
					}, "content"),
				},
			},
			OutputFormats: []string{"text", "json"},
			Notes: []string{
				"Use either a positional selector, `--message-id`, or `--index`. `--message-id` and `--index` are mutually exclusive.",
				"Raw payload flags replace the convenience text argument for the request body only.",
			},
		},
		"agent-message delete": {
			Arguments: map[string]parameterMetadata{
				"message-id-or-index": messageIDArg,
			},
			Flags: map[string]parameterMetadata{
				"message-id": {Description: "Explicit message ID to delete"},
				"index":      indexFlag,
			},
			Prerequisites: []string{"logged_in"},
			OutputFormats: []string{"text", "json"},
			Notes: []string{
				"Use either a positional selector, `--message-id`, or `--index`. `--message-id` and `--index` are mutually exclusive.",
			},
		},
		"agent-message react": {
			Arguments: map[string]parameterMetadata{
				"message-id-or-index": messageIDArg,
				"emoji":               {Description: "Emoji to add as a reaction", Constraints: []string{"must be non-empty and must not contain control characters"}},
			},
			Flags: map[string]parameterMetadata{
				"message-id":    {Description: "Explicit message ID to react to"},
				"index":         indexFlag,
				"payload":       {Description: "Inline raw JSON payload matching the react request body"},
				"payload-file":  {Description: "Read the raw react JSON payload from a file"},
				"payload-stdin": {Description: "Read the raw react JSON payload from stdin"},
			},
			Prerequisites: []string{"logged_in"},
			InputModes: []commandInputMode{
				{
					Name:        "selector_plus_emoji",
					Description: "Provide the selector and emoji via convenience arguments",
					Conditions:  []string{"use `<message-id-or-index> <emoji>` or combine `--message-id`/`--index` with `<emoji>`"},
					RequestShape: shapePtr(requestObjectShape("Add reaction body", map[string]schemaValueShape{
						"emoji": stringShape("Emoji to add as a reaction", withMinLength(1)),
					}, "emoji")),
				},
				{
					Name:        "raw_payload",
					Description: "Provide the react request body directly as JSON while resolving the selector separately",
					Conditions:  []string{"choose only one of --payload, --payload-file, or --payload-stdin", "selector still comes from the positional argument or `--message-id`/`--index`"},
					RequestShape: shapePtr(requestObjectShape("Add reaction body", map[string]schemaValueShape{
						"emoji": stringShape("Emoji to add as a reaction", withMinLength(1)),
					}, "emoji")),
				},
			},
			RequestShapes: []commandRequestShape{
				{
					Name:        "add_reaction_body",
					ContentType: "application/json",
					Description: "JSON body sent to POST /api/messages/:id/reactions",
					Shape: requestObjectShape("Add reaction body", map[string]schemaValueShape{
						"emoji": stringShape("Emoji to add as a reaction", withMinLength(1)),
					}, "emoji"),
				},
			},
			OutputFormats: []string{"text", "json"},
		},
		"agent-message unreact": {
			Arguments: map[string]parameterMetadata{
				"message-id-or-index": messageIDArg,
				"emoji":               {Description: "Emoji to remove", Constraints: []string{"must be non-empty and must not contain control characters"}},
			},
			Flags: map[string]parameterMetadata{
				"message-id": {Description: "Explicit message ID to unreact"},
				"index":      indexFlag,
			},
			Prerequisites: []string{"logged_in"},
			OutputFormats: []string{"text", "json"},
		},
		"agent-message watch": {
			Arguments: map[string]parameterMetadata{
				"username": usernameArg,
			},
			Flags: map[string]parameterMetadata{
				"json": {Description: "Emit matching events as NDJSON instead of text"},
			},
			Prerequisites: []string{"logged_in"},
			OutputFormats: []string{"text", "ndjson"},
			Notes: []string{
				"Establishes an SSE connection and filters `message.new` events for the resolved conversation.",
			},
		},
		"agent-message wait": {
			Arguments: map[string]parameterMetadata{
				"username": usernameArg,
			},
			Flags: map[string]parameterMetadata{
				"json": {Description: "Emit the next matching event as NDJSON instead of text"},
			},
			Prerequisites: []string{"logged_in"},
			OutputFormats: []string{"text", "ndjson"},
			Notes: []string{
				"Behaves like `watch` but exits after the first matching message event.",
			},
		},
	}
}

func globalFlagMetadata() map[string]parameterMetadata {
	return map[string]parameterMetadata{
		"config":     {Description: "Path to the config file to load"},
		"server-url": {Description: "Override server URL for this command only", Pattern: `^https?://`, Constraints: []string{"must not contain control characters, query strings, or fragments"}},
		"from":       {Description: "Use a saved profile for this command without switching the active profile"},
		"json":       {Description: "Emit machine-readable JSON when the command supports it"},
	}
}

func mergeParameterMetadata(base map[string]parameterMetadata, extra map[string]parameterMetadata) map[string]parameterMetadata {
	if len(base) == 0 && len(extra) == 0 {
		return nil
	}
	merged := make(map[string]parameterMetadata, len(base)+len(extra))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range extra {
		merged[key] = value
	}
	return merged
}

func stringShape(description string, modifiers ...func(*schemaValueShape)) schemaValueShape {
	shape := schemaValueShape{
		Type:        "string",
		Description: description,
	}
	for _, modifier := range modifiers {
		modifier(&shape)
	}
	return shape
}

func enumShape(description string, values ...string) schemaValueShape {
	return schemaValueShape{
		Type:        "string",
		Description: description,
		Enum:        append([]string(nil), values...),
	}
}

func objectShape(description string, properties map[string]schemaValueShape, required ...string) schemaValueShape {
	allow := true
	shape := schemaValueShape{
		Type:                 "object",
		Description:          description,
		AdditionalProperties: &allow,
	}
	if len(properties) > 0 {
		shape.Properties = properties
	}
	if len(required) > 0 {
		shape.Required = append([]string(nil), required...)
	}
	return shape
}

func requestObjectShape(description string, properties map[string]schemaValueShape, required ...string) schemaValueShape {
	shape := objectShape(description, properties, required...)
	return shape
}

func withPattern(pattern string) func(*schemaValueShape) {
	return func(shape *schemaValueShape) {
		shape.Pattern = pattern
	}
}

func withMinLength(length int) func(*schemaValueShape) {
	return func(shape *schemaValueShape) {
		shape.MinLength = intPtr(length)
	}
}

func intPtr(value int) *int {
	return &value
}

func shapePtr(shape schemaValueShape) *schemaValueShape {
	return &shape
}
