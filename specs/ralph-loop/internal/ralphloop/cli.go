package ralphloop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type OutputFormat string

const (
	OutputText   OutputFormat = "text"
	OutputJSON   OutputFormat = "json"
	OutputNDJSON OutputFormat = "ndjson"
)

const (
	commandMain      CommandKind = "main"
	commandInit      CommandKind = "init"
	commandTail      CommandKind = "tail"
	commandList      CommandKind = "ls"
	commandSchemaCmd CommandKind = "schema"
)

const (
	mainUsage = "Usage: ralph-loop \"<user prompt>\" [options]\n       ralph-loop init [options]\n       ralph-loop tail [selector] [options]\n       ralph-loop ls [selector] [options]\n       ralph-loop schema [command] [options]"
)

type CommandKind string

type CommonOptions struct {
	Output     OutputFormat
	OutputFile string
	Fields     []string
	Page       int
	PageSize   int
	PageAll    bool
}

type MainOptions struct {
	Prompt             string `json:"prompt"`
	Model              string `json:"model"`
	BaseBranch         string `json:"base_branch"`
	MaxIterations      int    `json:"max_iterations"`
	WorkBranch         string `json:"work_branch"`
	WorkBranchProvided bool   `json:"-"`
	TimeoutSeconds     int    `json:"timeout"`
	ApprovalPolicy     string `json:"approval_policy"`
	Sandbox            string `json:"sandbox"`
	PreserveWorktree   bool   `json:"preserve_worktree"`
	SkipPR             bool   `json:"skip_pr"`
	LandBase           bool   `json:"land_base"`
	DryRun             bool   `json:"dry_run"`
}

type InitOptions struct {
	BaseBranch         string `json:"base_branch"`
	WorkBranch         string `json:"work_branch"`
	WorkBranchProvided bool   `json:"-"`
	DryRun             bool   `json:"dry_run"`
}

type TailOptions struct {
	Selector string `json:"selector"`
	Lines    int    `json:"lines"`
	Follow   bool   `json:"follow"`
	Raw      bool   `json:"raw"`
}

type ListOptions struct {
	Selector string `json:"selector"`
}

type SchemaOptions struct {
	Command string `json:"command"`
}

type ParsedCommand struct {
	Kind          CommandKind
	Common        CommonOptions
	MainOptions   MainOptions
	InitOptions   InitOptions
	TailOptions   TailOptions
	ListOptions   ListOptions
	SchemaOptions SchemaOptions
}

type usageError struct {
	message string
}

func (err *usageError) Error() string {
	return err.message
}

func IsUsageError(err error) bool {
	var target *usageError
	return errors.As(err, &target)
}

type runContext struct {
	ctx          context.Context
	invokeCwd    string
	repoRoot     string
	stdin        io.Reader
	stdout       io.Writer
	stderr       io.Writer
	command      ParsedCommand
	textProgress io.Writer
}

func Run(args []string, cwd string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	format := detectOutputFormat(stdout)
	command, err := ParseCommand(args, stdin, format)
	if err != nil {
		return writeStartupError(stdout, stderr, format, commandNameFromArgs(args), err)
	}

	repoRoot, err := ResolveRepoRoot(cwd)
	if err != nil {
		return writeStartupError(stdout, stderr, command.Common.Output, string(command.Kind), err)
	}

	if err := validateCommand(command, cwd); err != nil {
		return writeCommandError(stdout, stderr, command.Common.Output, string(command.Kind), err)
	}

	runCtx := runContext{
		ctx:          context.Background(),
		invokeCwd:    cwd,
		repoRoot:     repoRoot,
		stdin:        stdin,
		stdout:       stdout,
		stderr:       stderr,
		command:      command,
		textProgress: stderr,
	}
	if command.Kind == commandMain {
		runCtx.textProgress = stdout
	}

	switch command.Kind {
	case commandInit:
		return executeInitCommand(runCtx)
	case commandTail:
		return executeTailCommand(runCtx)
	case commandList:
		return executeListCommand(runCtx)
	case commandSchemaCmd:
		return executeSchemaCommand(runCtx)
	default:
		return executeMainCommand(runCtx)
	}
}

func ResolveRepoRoot(cwd string) (string, error) {
	command := exec.Command("git", "rev-parse", "--show-toplevel")
	command.Dir = cwd
	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("failed to resolve repository root from %s: %w", cwd, err)
	}
	root := strings.TrimSpace(string(output))
	if root == "" {
		return "", fmt.Errorf("failed to resolve repository root from %s: empty git output", cwd)
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err == nil {
		root = resolved
	}
	return root, nil
}

func ParseCommand(args []string, stdin io.Reader, defaultOutput OutputFormat) (ParsedCommand, error) {
	kind, remainder := detectCommand(args)
	switch kind {
	case commandInit:
		return parseInitCommand(remainder, stdin, defaultOutput)
	case commandTail:
		return parseTailCommand(remainder, stdin, defaultOutput)
	case commandList:
		return parseListCommand(remainder, stdin, defaultOutput)
	case commandSchemaCmd:
		return parseSchemaCommand(remainder, stdin, defaultOutput)
	default:
		return parseMainCommand(remainder, stdin, defaultOutput)
	}
}

func detectCommand(args []string) (CommandKind, []string) {
	if len(args) == 0 {
		return commandMain, nil
	}
	switch args[0] {
	case "init":
		return commandInit, args[1:]
	case "tail":
		return commandTail, args[1:]
	case "ls":
		return commandList, args[1:]
	case "schema":
		return commandSchemaCmd, args[1:]
	default:
		return commandMain, args
	}
}

func parseMainCommand(args []string, stdin io.Reader, defaultOutput OutputFormat) (ParsedCommand, error) {
	command := ParsedCommand{
		Kind: commandMain,
		Common: CommonOptions{
			Output:   defaultOutput,
			Page:     1,
			PageSize: 50,
		},
		MainOptions: MainOptions{
			Model:            "gpt-5.3-codex",
			BaseBranch:       "main",
			MaxIterations:    20,
			TimeoutSeconds:   43200,
			ApprovalPolicy:   "never",
			Sandbox:          "workspace-write",
			PreserveWorktree: false,
		},
	}
	payload, positionals, err := parseArgsAndPayload(args, stdin, &command, descriptorFor(command.Kind))
	if err != nil {
		return ParsedCommand{}, err
	}

	if payload != nil {
		if err := mergeJSONPayload(payload, &command.MainOptions, &command.Common); err != nil {
			return ParsedCommand{}, err
		}
	}
	if len(positionals) > 0 {
		command.MainOptions.Prompt = strings.TrimSpace(strings.Join(positionals, " "))
	}
	if command.MainOptions.Prompt == "" {
		return ParsedCommand{}, &usageError{message: mainUsage}
	}
	if command.MainOptions.WorkBranch == "" {
		command.MainOptions.WorkBranch = "ralph-" + trimToLength(slugifyPrompt(command.MainOptions.Prompt), 58)
	}
	return command, nil
}

func parseInitCommand(args []string, stdin io.Reader, defaultOutput OutputFormat) (ParsedCommand, error) {
	command := ParsedCommand{
		Kind: commandInit,
		Common: CommonOptions{
			Output:   defaultOutput,
			Page:     1,
			PageSize: 50,
		},
		InitOptions: InitOptions{
			BaseBranch: "main",
		},
	}
	payload, positionals, err := parseArgsAndPayload(args, stdin, &command, descriptorFor(command.Kind))
	if err != nil {
		return ParsedCommand{}, err
	}
	if len(positionals) > 0 {
		return ParsedCommand{}, fmt.Errorf("init does not accept positional arguments: %s", strings.Join(positionals, " "))
	}
	if payload != nil {
		if err := mergeJSONPayload(payload, &command.InitOptions, &command.Common); err != nil {
			return ParsedCommand{}, err
		}
	}
	return command, nil
}

func parseTailCommand(args []string, stdin io.Reader, defaultOutput OutputFormat) (ParsedCommand, error) {
	command := ParsedCommand{
		Kind: commandTail,
		Common: CommonOptions{
			Output:   defaultOutput,
			Page:     1,
			PageSize: 50,
		},
		TailOptions: TailOptions{
			Lines: 40,
		},
	}
	payload, positionals, err := parseArgsAndPayload(args, stdin, &command, descriptorFor(command.Kind))
	if err != nil {
		return ParsedCommand{}, err
	}
	if payload != nil {
		if err := mergeJSONPayload(payload, &command.TailOptions, &command.Common); err != nil {
			return ParsedCommand{}, err
		}
	}
	if len(positionals) > 1 {
		return ParsedCommand{}, fmt.Errorf("expected at most one selector, received: %s", strings.Join(positionals, " "))
	}
	if len(positionals) == 1 {
		command.TailOptions.Selector = positionals[0]
	}
	return command, nil
}

func parseListCommand(args []string, stdin io.Reader, defaultOutput OutputFormat) (ParsedCommand, error) {
	command := ParsedCommand{
		Kind: commandList,
		Common: CommonOptions{
			Output:   defaultOutput,
			Page:     1,
			PageSize: 50,
		},
	}
	payload, positionals, err := parseArgsAndPayload(args, stdin, &command, descriptorFor(command.Kind))
	if err != nil {
		return ParsedCommand{}, err
	}
	if payload != nil {
		if err := mergeJSONPayload(payload, &command.ListOptions, &command.Common); err != nil {
			return ParsedCommand{}, err
		}
	}
	if len(positionals) > 1 {
		return ParsedCommand{}, fmt.Errorf("expected at most one selector, received: %s", strings.Join(positionals, " "))
	}
	if len(positionals) == 1 {
		command.ListOptions.Selector = positionals[0]
	}
	return command, nil
}

func parseSchemaCommand(args []string, stdin io.Reader, defaultOutput OutputFormat) (ParsedCommand, error) {
	command := ParsedCommand{
		Kind: commandSchemaCmd,
		Common: CommonOptions{
			Output:   defaultOutput,
			Page:     1,
			PageSize: 50,
		},
	}
	payload, positionals, err := parseArgsAndPayload(args, stdin, &command, descriptorFor(command.Kind))
	if err != nil {
		return ParsedCommand{}, err
	}
	if payload != nil {
		if err := mergeJSONPayload(payload, &command.SchemaOptions, &command.Common); err != nil {
			return ParsedCommand{}, err
		}
	}
	if len(positionals) > 1 {
		return ParsedCommand{}, fmt.Errorf("expected at most one command name, received: %s", strings.Join(positionals, " "))
	}
	if len(positionals) == 1 && command.SchemaOptions.Command == "" {
		command.SchemaOptions.Command = positionals[0]
	}
	return command, nil
}

func parseArgsAndPayload(args []string, stdin io.Reader, command *ParsedCommand, descriptor commandDescriptor) (map[string]any, []string, error) {
	positionals := make([]string, 0, len(args))
	var payloadText string
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if arg == "--help" || arg == "-h" {
			return nil, nil, &usageError{message: mainUsage}
		}
		if handled, err := consumeOptionSpec(commonOptionSpecs(), command, args, &index, stdin, arg, &payloadText); handled || err != nil {
			if err != nil {
				return nil, nil, err
			}
			continue
		}
		if handled, err := consumeOptionSpec(descriptor.OptionSpecs, command, args, &index, stdin, arg, &payloadText); handled || err != nil {
			if err != nil {
				return nil, nil, err
			}
			continue
		}
		if strings.HasPrefix(arg, "--") || strings.HasPrefix(arg, "-") {
			return nil, nil, fmt.Errorf("unknown option: %s", arg)
		}
		positionals = append(positionals, arg)
	}

	if payloadText == "" {
		return nil, positionals, nil
	}
	payload := map[string]any{}
	if err := json.Unmarshal([]byte(payloadText), &payload); err != nil {
		return nil, nil, fmt.Errorf("invalid JSON payload: %w", err)
	}
	return payload, positionals, nil
}

func consumeOptionSpec(specs []optionSpec, command *ParsedCommand, args []string, index *int, stdin io.Reader, arg string, payloadText *string) (bool, error) {
	for _, spec := range specs {
		if !matchesOptionSpec(spec, arg) {
			continue
		}
		value := ""
		if spec.RequiresValue {
			next, err := requireValue(args, index, arg)
			if err != nil {
				return true, err
			}
			value = next
		}
		if err := spec.Apply(command, value, stdin, payloadText); err != nil {
			return true, err
		}
		return true, nil
	}
	return false, nil
}

func mergeJSONPayload(payload map[string]any, target any, common *CommonOptions) error {
	if payload == nil {
		return nil
	}
	clone := map[string]any{}
	for key, value := range payload {
		clone[key] = value
	}
	if rawOutput, ok := clone["output"].(string); ok {
		common.Output = OutputFormat(rawOutput)
		delete(clone, "output")
	}
	if rawFields, ok := clone["fields"].(string); ok {
		common.Fields = splitCSV(rawFields)
		delete(clone, "fields")
	}
	if rawOutputFile, ok := clone["output_file"].(string); ok {
		common.OutputFile = rawOutputFile
		delete(clone, "output_file")
	}
	if rawPage, ok := jsonNumberToInt(clone["page"]); ok {
		common.Page = rawPage
		delete(clone, "page")
	}
	if rawPageSize, ok := jsonNumberToInt(clone["page_size"]); ok {
		common.PageSize = rawPageSize
		delete(clone, "page_size")
	}
	if rawPageAll, ok := clone["page_all"].(bool); ok {
		common.PageAll = rawPageAll
		delete(clone, "page_all")
	}
	encoded, err := json.Marshal(clone)
	if err != nil {
		return err
	}
	return json.Unmarshal(encoded, target)
}

func requireValue(args []string, index *int, flag string) (string, error) {
	*index = *index + 1
	if *index >= len(args) {
		return "", fmt.Errorf("missing value for %s", flag)
	}
	return args[*index], nil
}

func detectOutputFormat(stdout io.Writer) OutputFormat {
	file, ok := stdout.(*os.File)
	if !ok {
		return OutputJSON
	}
	info, err := file.Stat()
	if err != nil {
		return OutputJSON
	}
	if info.Mode()&os.ModeCharDevice != 0 {
		return OutputText
	}
	return OutputJSON
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func commandNameFromArgs(args []string) string {
	if len(args) == 0 {
		return string(commandMain)
	}
	switch args[0] {
	case "init", "tail", "ls", "schema":
		return args[0]
	default:
		return string(commandMain)
	}
}

func writeStartupError(stdout io.Writer, stderr io.Writer, format OutputFormat, command string, err error) int {
	if IsUsageError(err) {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return 2
	}
	return writeCommandError(stdout, stderr, format, command, err)
}

func writeCommandError(stdout io.Writer, stderr io.Writer, format OutputFormat, command string, err error) int {
	if format == OutputText {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return 1
	}
	messageResult := sanitizeUntrustedText(err.Error())
	errorRecord := map[string]any{
		"code":    classifyError(err),
		"message": messageResult.Text,
	}
	applySanitizationMetadata(errorRecord, messageResult)
	payload := map[string]any{
		"command": command,
		"status":  "failed",
		"error":   errorRecord,
	}
	_ = writeStructuredOutput(stdout, format, payload)
	return 1
}

func classifyError(err error) string {
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "usage"), strings.Contains(message, "missing value"), strings.Contains(message, "invalid value"):
		return "invalid_request"
	case strings.Contains(message, "output-file"):
		return "invalid_output_file"
	case strings.Contains(message, "selector"):
		return "invalid_selector"
	default:
		return "command_failed"
	}
}

func writeStructuredOutput(w io.Writer, format OutputFormat, payload any) error {
	switch format {
	case OutputNDJSON:
		switch converted := payload.(type) {
		case []map[string]any:
			for _, record := range converted {
				if err := writeJSONLine(w, record); err != nil {
					return err
				}
			}
			return nil
		case []any:
			for _, record := range converted {
				if err := writeJSONLine(w, record); err != nil {
					return err
				}
			}
			return nil
		default:
			return writeJSONLine(w, payload)
		}
	default:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(payload)
	}
}

func writeJSONLine(w io.Writer, payload any) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "%s\n", encoded)
	return err
}

func slugifyPrompt(prompt string) string {
	slug := strings.ToLower(strings.TrimSpace(prompt))
	slug = slugInvalidCharsPattern.ReplaceAllString(slug, "-")
	slug = slugTrimDashesPattern.ReplaceAllString(slug, "")
	slug = slugMultiDashPattern.ReplaceAllString(slug, "-")
	if slug == "" {
		return "task"
	}
	return slug
}

func trimToLength(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return strings.Trim(value[:limit], "-")
}

var (
	slugInvalidCharsPattern = regexp.MustCompile(`[^a-z0-9]+`)
	slugTrimDashesPattern   = regexp.MustCompile(`^-+|-+$`)
	slugMultiDashPattern    = regexp.MustCompile(`-+`)
)

func jsonNumberToInt(value any) (int, bool) {
	switch converted := value.(type) {
	case float64:
		return int(converted), true
	case int:
		return converted, true
	case json.Number:
		parsed, err := converted.Int64()
		if err != nil {
			return 0, false
		}
		return int(parsed), true
	default:
		return 0, false
	}
}

func matchesOptionSpec(spec optionSpec, arg string) bool {
	if arg == spec.Flag {
		return true
	}
	for _, alias := range spec.Field.Alias {
		if arg == alias {
			return true
		}
	}
	return false
}
