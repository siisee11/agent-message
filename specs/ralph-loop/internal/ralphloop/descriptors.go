package ralphloop

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

type optionSpec struct {
	Field               schemaField
	Flag                string
	RequiresValue       bool
	IncludeInRawPayload bool
	Apply               func(command *ParsedCommand, value string, stdin io.Reader, payloadText *string) error
}

type commandDescriptor struct {
	Kind           CommandKind
	Description    string
	MutatesState   bool
	SupportsDryRun bool
	Positionals    []schemaField
	OptionSpecs    []optionSpec
}

func commandDescriptors() []commandDescriptor {
	return []commandDescriptor{
		{
			Kind:           commandMain,
			Description:    "Run the Ralph loop through setup, coding, and optional PR phases",
			MutatesState:   true,
			SupportsDryRun: true,
			Positionals: []schemaField{
				{Name: "prompt", Description: "User task prompt", Type: "string", Required: true},
			},
			OptionSpecs: []optionSpec{
				stringOption("model", "Codex model", "gpt-5.3-codex", nil, func(command *ParsedCommand, value string) {
					command.MainOptions.Model = value
				}),
				stringOption("base_branch", "Base branch", "main", nil, func(command *ParsedCommand, value string) {
					command.MainOptions.BaseBranch = value
				}),
				intOption("max_iterations", "Maximum loop iterations", 20, nil, func(command *ParsedCommand, value int) {
					command.MainOptions.MaxIterations = value
				}),
				stringOption("work_branch", "Working branch name", nil, nil, func(command *ParsedCommand, value string) {
					command.MainOptions.WorkBranch = value
					command.MainOptions.WorkBranchProvided = true
				}),
				intOption("timeout", "Maximum wall clock time in seconds", 43200, nil, func(command *ParsedCommand, value int) {
					command.MainOptions.TimeoutSeconds = value
				}),
				stringOption("approval_policy", "Codex approval policy", "never", nil, func(command *ParsedCommand, value string) {
					command.MainOptions.ApprovalPolicy = value
				}),
				stringOption("sandbox", "Codex sandbox policy", "workspace-write", nil, func(command *ParsedCommand, value string) {
					command.MainOptions.Sandbox = value
				}),
				boolOption("preserve_worktree", "Keep the generated worktree", false, nil, func(command *ParsedCommand) {
					command.MainOptions.PreserveWorktree = true
				}),
				boolOption("skip_pr", "Stop after the coding loop without running the PR agent", false, nil, func(command *ParsedCommand) {
					command.MainOptions.SkipPR = true
				}),
				boolOption("land_base", "After coding completes, land the work branch commits onto the local base branch. Requires skip_pr.", false, nil, func(command *ParsedCommand) {
					command.MainOptions.LandBase = true
				}),
				boolOption("dry_run", "Validate and describe the request", false, nil, func(command *ParsedCommand) {
					command.MainOptions.DryRun = true
				}),
			},
		},
		{
			Kind:           commandInit,
			Description:    "Prepare a clean worktree, install dependencies, and verify the build",
			MutatesState:   true,
			SupportsDryRun: true,
			OptionSpecs: []optionSpec{
				stringOption("base_branch", "Base branch", "main", nil, func(command *ParsedCommand, value string) {
					command.InitOptions.BaseBranch = value
				}),
				stringOption("work_branch", "Working branch name", nil, nil, func(command *ParsedCommand, value string) {
					command.InitOptions.WorkBranch = value
					command.InitOptions.WorkBranchProvided = true
				}),
				boolOption("dry_run", "Validate and describe the request", false, nil, func(command *ParsedCommand) {
					command.InitOptions.DryRun = true
				}),
			},
		},
		{
			Kind:           commandList,
			Description:    "List active Ralph loop sessions",
			MutatesState:   false,
			SupportsDryRun: false,
			Positionals: []schemaField{
				{Name: "selector", Description: "Optional session selector", Type: "string"},
			},
		},
		{
			Kind:           commandTail,
			Description:    "Inspect Ralph loop logs",
			MutatesState:   false,
			SupportsDryRun: false,
			Positionals: []schemaField{
				{Name: "selector", Description: "Optional log selector", Type: "string"},
			},
			OptionSpecs: []optionSpec{
				intOption("lines", "Number of log lines", 40, []string{"-n"}, func(command *ParsedCommand, value int) {
					command.TailOptions.Lines = value
				}),
				boolOption("follow", "Follow appended lines", false, []string{"-f"}, func(command *ParsedCommand) {
					command.TailOptions.Follow = true
				}),
				boolOption("raw", "Return raw log payloads", false, nil, func(command *ParsedCommand) {
					command.TailOptions.Raw = true
				}),
			},
		},
		{
			Kind:           commandSchemaCmd,
			Description:    "Describe the live command schemas",
			MutatesState:   false,
			SupportsDryRun: false,
			Positionals: []schemaField{
				{Name: "command", Description: "Optional command name", Type: "string"},
			},
			OptionSpecs: []optionSpec{
				stringOption("command", "Command name to describe", nil, nil, func(command *ParsedCommand, value string) {
					command.SchemaOptions.Command = value
				}),
			},
		},
	}
}

func descriptorFor(kind CommandKind) commandDescriptor {
	for _, descriptor := range commandDescriptors() {
		if descriptor.Kind == kind {
			return descriptor
		}
	}
	return commandDescriptor{Kind: kind}
}

func commonOptionSpecs() []optionSpec {
	return []optionSpec{
		{
			Field:         schemaField{Name: "json", Description: "Raw JSON payload or - for stdin", Type: "string"},
			Flag:          "--json",
			RequiresValue: true,
			Apply: func(command *ParsedCommand, value string, stdin io.Reader, payloadText *string) error {
				if value == "-" {
					data, err := io.ReadAll(stdin)
					if err != nil {
						return err
					}
					*payloadText = string(data)
					return nil
				}
				*payloadText = value
				return nil
			},
		},
		{
			Field:               schemaField{Name: "output", Description: "Output format", Type: "string", Default: "text|json", Enum: []string{"text", "json", "ndjson"}},
			Flag:                "--output",
			RequiresValue:       true,
			IncludeInRawPayload: true,
			Apply: func(command *ParsedCommand, value string, stdin io.Reader, payloadText *string) error {
				command.Common.Output = OutputFormat(value)
				return nil
			},
		},
		{
			Field:               schemaField{Name: "output_file", Description: "Write machine-readable output to a file under the current working directory", Type: "string"},
			Flag:                "--output-file",
			RequiresValue:       true,
			IncludeInRawPayload: true,
			Apply: func(command *ParsedCommand, value string, stdin io.Reader, payloadText *string) error {
				command.Common.OutputFile = value
				return nil
			},
		},
		{
			Field:               schemaField{Name: "fields", Description: "Comma-separated field mask for read commands", Type: "string"},
			Flag:                "--fields",
			RequiresValue:       true,
			IncludeInRawPayload: true,
			Apply: func(command *ParsedCommand, value string, stdin io.Reader, payloadText *string) error {
				command.Common.Fields = splitCSV(value)
				return nil
			},
		},
		{
			Field:               schemaField{Name: "page", Description: "Page number for read commands", Type: "integer", Default: 1},
			Flag:                "--page",
			RequiresValue:       true,
			IncludeInRawPayload: true,
			Apply: func(command *ParsedCommand, value string, stdin io.Reader, payloadText *string) error {
				parsed, err := strconv.Atoi(value)
				if err != nil {
					return fmt.Errorf("invalid value for --page: %s", value)
				}
				command.Common.Page = parsed
				return nil
			},
		},
		{
			Field:               schemaField{Name: "page_size", Description: "Items per page for read commands", Type: "integer", Default: 50},
			Flag:                "--page-size",
			RequiresValue:       true,
			IncludeInRawPayload: true,
			Apply: func(command *ParsedCommand, value string, stdin io.Reader, payloadText *string) error {
				parsed, err := strconv.Atoi(value)
				if err != nil {
					return fmt.Errorf("invalid value for --page-size: %s", value)
				}
				command.Common.PageSize = parsed
				return nil
			},
		},
		{
			Field:               schemaField{Name: "page_all", Description: "Read all pages", Type: "boolean", Default: false},
			Flag:                "--page-all",
			RequiresValue:       false,
			IncludeInRawPayload: true,
			Apply: func(command *ParsedCommand, value string, stdin io.Reader, payloadText *string) error {
				command.Common.PageAll = true
				return nil
			},
		},
	}
}

func commandSchemaFromDescriptor(descriptor commandDescriptor) commandSchema {
	options := make([]schemaField, 0, len(commonOptionSpecs())+len(descriptor.OptionSpecs))
	for _, spec := range commonOptionSpecs() {
		options = append(options, spec.Field)
	}
	for _, spec := range descriptor.OptionSpecs {
		options = append(options, spec.Field)
	}

	rawPayload := append([]schemaField{}, descriptor.Positionals...)
	for _, spec := range commonOptionSpecs() {
		if spec.IncludeInRawPayload {
			rawPayload = append(rawPayload, spec.Field)
		}
	}
	for _, spec := range descriptor.OptionSpecs {
		if spec.IncludeInRawPayload && !containsSchemaField(rawPayload, spec.Field.Name) {
			rawPayload = append(rawPayload, spec.Field)
		}
	}

	return commandSchema{
		Command:        string(descriptor.Kind),
		Description:    descriptor.Description,
		MutatesState:   descriptor.MutatesState,
		SupportsDryRun: descriptor.SupportsDryRun,
		Positionals:    descriptor.Positionals,
		Options:        options,
		RawPayload:     rawPayload,
	}
}

func containsSchemaField(fields []schemaField, name string) bool {
	for _, field := range fields {
		if field.Name == name {
			return true
		}
	}
	return false
}

func stringOption(name string, description string, defaultValue any, aliases []string, apply func(command *ParsedCommand, value string)) optionSpec {
	return optionSpec{
		Field:               schemaField{Name: name, Description: description, Type: "string", Default: defaultValue, Alias: aliases},
		Flag:                optionFlag(name),
		RequiresValue:       true,
		IncludeInRawPayload: true,
		Apply: func(command *ParsedCommand, value string, stdin io.Reader, payloadText *string) error {
			apply(command, value)
			return nil
		},
	}
}

func intOption(name string, description string, defaultValue any, aliases []string, apply func(command *ParsedCommand, value int)) optionSpec {
	return optionSpec{
		Field:               schemaField{Name: name, Description: description, Type: "integer", Default: defaultValue, Alias: aliases},
		Flag:                optionFlag(name),
		RequiresValue:       true,
		IncludeInRawPayload: true,
		Apply: func(command *ParsedCommand, value string, stdin io.Reader, payloadText *string) error {
			parsed, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("invalid value for %s: %s", optionFlag(name), value)
			}
			apply(command, parsed)
			return nil
		},
	}
}

func boolOption(name string, description string, defaultValue bool, aliases []string, apply func(command *ParsedCommand)) optionSpec {
	return optionSpec{
		Field:               schemaField{Name: name, Description: description, Type: "boolean", Default: defaultValue, Alias: aliases},
		Flag:                optionFlag(name),
		RequiresValue:       false,
		IncludeInRawPayload: true,
		Apply: func(command *ParsedCommand, value string, stdin io.Reader, payloadText *string) error {
			apply(command)
			return nil
		},
	}
}

func optionFlag(name string) string {
	return "--" + strings.ReplaceAll(name, "_", "-")
}
