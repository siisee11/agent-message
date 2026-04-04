package cobra

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// PositionalArgs validates positional command arguments.
type PositionalArgs func(cmd *Command, args []string) error

// NoArgs validates that no positional arguments were provided.
func NoArgs(_ *Command, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("accepts 0 arg(s), received %d", len(args))
	}
	return nil
}

// ExactArgs validates that exactly n positional arguments were provided.
func ExactArgs(n int) PositionalArgs {
	return func(_ *Command, args []string) error {
		if len(args) != n {
			return fmt.Errorf("accepts %d arg(s), received %d", n, len(args))
		}
		return nil
	}
}

// Command is a minimal subset of Cobra's command model.
type Command struct {
	Use               string
	Short             string
	Args              PositionalArgs
	RunE              func(cmd *Command, args []string) error
	PersistentPreRunE func(cmd *Command, args []string) error

	parent          *Command
	subcommands     []*Command
	flags           *FlagSet
	persistentFlags *FlagSet
	argsOverride    []string
}

type FlagInfo struct {
	Name       string `json:"name"`
	Shorthand  string `json:"shorthand,omitempty"`
	Type       string `json:"type"`
	Usage      string `json:"usage"`
	Default    any    `json:"default,omitempty"`
	Persistent bool   `json:"persistent,omitempty"`
}

func (c *Command) CommandPath() string {
	if c == nil {
		return ""
	}
	parts := make([]string, 0, 4)
	for current := c; current != nil; current = current.parent {
		if name := current.Name(); name != "" {
			parts = append(parts, name)
		}
	}
	for left, right := 0, len(parts)-1; left < right; left, right = left+1, right-1 {
		parts[left], parts[right] = parts[right], parts[left]
	}
	return strings.Join(parts, " ")
}

func (c *Command) Name() string {
	trimmed := strings.TrimSpace(c.Use)
	if trimmed == "" {
		return ""
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func (c *Command) AddCommand(cmds ...*Command) {
	for _, cmd := range cmds {
		if cmd == nil {
			continue
		}
		cmd.parent = c
		c.subcommands = append(c.subcommands, cmd)
	}
}

func (c *Command) Children() []*Command {
	if c == nil || len(c.subcommands) == 0 {
		return nil
	}
	return append([]*Command(nil), c.subcommands...)
}

func (c *Command) Flags() *FlagSet {
	if c.flags == nil {
		c.flags = NewFlagSet()
	}
	return c.flags
}

func (c *Command) PersistentFlags() *FlagSet {
	if c.persistentFlags == nil {
		c.persistentFlags = NewFlagSet()
	}
	return c.persistentFlags
}

func (c *Command) SetArgs(args []string) {
	c.argsOverride = append([]string(nil), args...)
}

func (c *Command) Execute() error {
	_, err := c.ExecuteC()
	return err
}

func (c *Command) ExecuteC() (*Command, error) {
	var args []string
	if c.argsOverride != nil {
		args = append([]string(nil), c.argsOverride...)
	} else {
		args = os.Args[1:]
	}
	return c.execute(args, nil)
}

func (c *Command) execute(args []string, inheritedPersistent *flagRegistry) (*Command, error) {
	if c == nil {
		return nil, errors.New("command is required")
	}
	if target, jsonMode, ok, err := c.resolveHelpRequest(args, inheritedPersistent); ok || err != nil {
		if err != nil {
			return c, err
		}
		target.printHelp(jsonMode)
		return target, nil
	}

	mergedPersistent := mergeFlagRegistries(inheritedPersistent, toRegistry(c.persistentFlags))

	if len(c.subcommands) > 0 {
		remaining, err := mergedPersistent.parsePrefix(args)
		if err != nil {
			return c, err
		}
		if len(remaining) == 0 {
			if c.RunE == nil {
				c.printHelp(containsJSONFlag(args))
				return c, nil
			}
			if err := c.runWithHooks(nil); err != nil {
				return c, err
			}
			return c, nil
		}

		subName := remaining[0]
		for _, child := range c.subcommands {
			if child.Name() == subName {
				return child.execute(remaining[1:], mergedPersistent)
			}
		}
		return c, fmt.Errorf("unknown command: %s", subName)
	}

	flagRegistry := mergeFlagRegistries(mergedPersistent, toRegistry(c.flags))
	positionals, err := flagRegistry.parseAll(args)
	if err != nil {
		return c, err
	}
	if c.Args != nil {
		if err := c.Args(c, positionals); err != nil {
			return c, err
		}
	}

	if err := c.runWithHooks(positionals); err != nil {
		return c, err
	}
	return c, nil
}

func (c *Command) runWithHooks(args []string) error {
	chain := make([]*Command, 0, 4)
	for current := c; current != nil; current = current.parent {
		chain = append(chain, current)
	}
	for i := len(chain) - 1; i >= 0; i-- {
		current := chain[i]
		if current.PersistentPreRunE != nil {
			if err := current.PersistentPreRunE(c, args); err != nil {
				return err
			}
		}
	}
	if c.RunE != nil {
		return c.RunE(c, args)
	}
	return nil
}

func (c *Command) resolveHelpRequest(args []string, inheritedPersistent *flagRegistry) (*Command, bool, bool, error) {
	if len(args) == 0 {
		return nil, false, false, nil
	}
	if strings.TrimSpace(args[0]) == "help" {
		target, err := c.findCommandForHelp(args[1:], inheritedPersistent)
		if err != nil {
			return nil, false, true, err
		}
		return target, containsJSONFlag(args), true, nil
	}
	if !containsHelpFlag(args) {
		return nil, false, false, nil
	}
	target, err := c.findCommandForHelp(filterArgs(args, func(arg string) bool {
		return arg == "--help" || arg == "-h"
	}), inheritedPersistent)
	if err != nil {
		return nil, false, true, err
	}
	return target, containsJSONFlag(args), true, nil
}

func (c *Command) findCommandForHelp(args []string, inheritedPersistent *flagRegistry) (*Command, error) {
	if c == nil {
		return nil, errors.New("command is required")
	}
	mergedPersistent := mergeFlagRegistries(inheritedPersistent, toRegistry(c.persistentFlags))
	remaining, err := mergedPersistent.parsePrefix(args)
	if err != nil {
		return nil, err
	}
	if len(remaining) == 0 || len(c.subcommands) == 0 {
		return c, nil
	}

	subName := remaining[0]
	for _, child := range c.subcommands {
		if child.Name() == subName {
			return child.findCommandForHelp(remaining[1:], mergedPersistent)
		}
	}
	return nil, fmt.Errorf("unknown command: %s", subName)
}

func (c *Command) printHelp(jsonMode bool) {
	if jsonMode {
		_ = writeHelpJSON(os.Stdout, c)
		return
	}
	_, _ = fmt.Fprint(os.Stdout, c.HelpText())
}

func (c *Command) HelpText() string {
	if c == nil {
		return ""
	}

	var builder strings.Builder
	if short := strings.TrimSpace(c.Short); short != "" {
		builder.WriteString(short)
		builder.WriteString("\n\n")
	}

	builder.WriteString("Usage:\n")
	builder.WriteString("  ")
	builder.WriteString(c.CommandPath())
	if use := strings.TrimSpace(c.Use); use != "" {
		suffix := strings.TrimSpace(strings.TrimPrefix(use, c.Name()))
		if suffix != "" {
			builder.WriteString(" ")
			builder.WriteString(suffix)
		}
	}
	if len(c.allVisibleFlags()) > 0 {
		builder.WriteString(" [flags]")
	}
	builder.WriteString("\n")

	if len(c.subcommands) > 0 {
		builder.WriteString("\nAvailable Commands:\n")
		for _, child := range c.sortedSubcommands() {
			builder.WriteString(fmt.Sprintf("  %-18s %s\n", child.Name(), strings.TrimSpace(child.Short)))
		}
	}

	flags := c.allVisibleFlags()
	if len(flags) > 0 {
		builder.WriteString("\nFlags:\n")
		for _, spec := range flags {
			builder.WriteString("  ")
			builder.WriteString(spec.helpLine())
			builder.WriteString("\n")
		}
	}
	return builder.String()
}

func (c *Command) allVisibleFlags() []*flagSpec {
	if c == nil {
		return nil
	}
	seen := make(map[string]struct{})
	flags := make([]*flagSpec, 0, 8)

	ancestors := make([]*Command, 0, 4)
	for current := c.parent; current != nil; current = current.parent {
		ancestors = append(ancestors, current)
	}
	for left, right := 0, len(ancestors)-1; left < right; left, right = left+1, right-1 {
		ancestors[left], ancestors[right] = ancestors[right], ancestors[left]
	}
	for _, ancestor := range ancestors {
		flags = appendUniqueFlagSpecs(flags, ancestor.persistentFlags, seen)
	}
	flags = appendUniqueFlagSpecs(flags, c.persistentFlags, seen)
	flags = appendUniqueFlagSpecs(flags, c.flags, seen)
	return flags
}

func appendUniqueFlagSpecs(dst []*flagSpec, set *FlagSet, seen map[string]struct{}) []*flagSpec {
	if set == nil {
		return dst
	}
	for _, spec := range set.specs {
		if spec == nil {
			continue
		}
		key := spec.name
		if key == "" {
			key = "-" + spec.shorthand
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		dst = append(dst, spec)
	}
	return dst
}

func (c *Command) sortedSubcommands() []*Command {
	out := append([]*Command(nil), c.subcommands...)
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name() < out[j].Name()
	})
	return out
}

func containsHelpFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}

func containsJSONFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--json" {
			return true
		}
	}
	return false
}

func filterArgs(args []string, drop func(string) bool) []string {
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		if drop(arg) {
			continue
		}
		filtered = append(filtered, arg)
	}
	return filtered
}

func writeHelpJSON(w *os.File, c *Command) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(map[string]any{
		"command":     c.Name(),
		"path":        c.CommandPath(),
		"use":         strings.TrimSpace(c.Use),
		"short":       strings.TrimSpace(c.Short),
		"subcommands": buildHelpJSONSubcommands(c),
		"flags":       buildHelpJSONFlags(c.allVisibleFlags()),
	})
}

func buildHelpJSONSubcommands(c *Command) []map[string]string {
	if c == nil || len(c.subcommands) == 0 {
		return []map[string]string{}
	}
	items := make([]map[string]string, 0, len(c.subcommands))
	for _, child := range c.sortedSubcommands() {
		items = append(items, map[string]string{
			"name":  child.Name(),
			"use":   strings.TrimSpace(child.Use),
			"short": strings.TrimSpace(child.Short),
		})
	}
	return items
}

func buildHelpJSONFlags(flags []*flagSpec) []map[string]string {
	items := make([]map[string]string, 0, len(flags))
	for _, spec := range flags {
		if spec == nil {
			continue
		}
		item := map[string]string{
			"name":  spec.name,
			"type":  spec.kind.helpType(),
			"usage": strings.TrimSpace(spec.usage),
		}
		if spec.shorthand != "" {
			item["shorthand"] = spec.shorthand
		}
		items = append(items, item)
	}
	return items
}

// FlagSet stores command flag definitions.
type FlagSet struct {
	specs []*flagSpec
}

type flagSpec struct {
	name      string
	shorthand string
	usage     string
	kind      flagKind
	defValue  any
	stringDst *string
	intDst    *int
	boolDst   *bool
}

type flagKind int

const (
	flagKindString flagKind = iota + 1
	flagKindInt
	flagKindBool
)

func (k flagKind) helpType() string {
	switch k {
	case flagKindString:
		return "string"
	case flagKindInt:
		return "int"
	case flagKindBool:
		return "bool"
	default:
		return "unknown"
	}
}

func (s *flagSpec) helpLine() string {
	if s == nil {
		return ""
	}
	parts := make([]string, 0, 2)
	if s.shorthand != "" {
		parts = append(parts, "-"+s.shorthand)
	}
	long := "--" + s.name
	if s.kind != flagKindBool {
		long += " <" + s.kind.helpType() + ">"
	}
	parts = append(parts, long)
	line := strings.Join(parts, ", ")
	if usage := strings.TrimSpace(s.usage); usage != "" {
		line += "  " + usage
	}
	return line
}

func NewFlagSet() *FlagSet {
	return &FlagSet{specs: make([]*flagSpec, 0, 4)}
}

func (f *FlagSet) StringVar(dst *string, name, value, usage string) {
	if dst != nil {
		*dst = value
	}
	f.specs = append(f.specs, &flagSpec{
		name:      strings.TrimSpace(name),
		usage:     strings.TrimSpace(usage),
		kind:      flagKindString,
		defValue:  value,
		stringDst: dst,
	})
}

func (f *FlagSet) IntP(name, shorthand string, value int, usage string) *int {
	dst := new(int)
	*dst = value
	f.specs = append(f.specs, &flagSpec{
		name:      strings.TrimSpace(name),
		shorthand: strings.TrimSpace(shorthand),
		usage:     strings.TrimSpace(usage),
		kind:      flagKindInt,
		defValue:  value,
		intDst:    dst,
	})
	return dst
}

func (f *FlagSet) BoolVar(dst *bool, name string, value bool, usage string) {
	if dst != nil {
		*dst = value
	}
	f.specs = append(f.specs, &flagSpec{
		name:     strings.TrimSpace(name),
		usage:    strings.TrimSpace(usage),
		kind:     flagKindBool,
		defValue: value,
		boolDst:  dst,
	})
}

func (c *Command) LocalFlagInfos() []FlagInfo {
	return buildFlagInfos(c.flags, false)
}

func (c *Command) PersistentFlagInfos() []FlagInfo {
	return buildFlagInfos(c.persistentFlags, true)
}

func (c *Command) InheritedPersistentFlagInfos() []FlagInfo {
	if c == nil {
		return nil
	}
	ancestors := make([]*Command, 0, 4)
	for current := c.parent; current != nil; current = current.parent {
		ancestors = append(ancestors, current)
	}
	for left, right := 0, len(ancestors)-1; left < right; left, right = left+1, right-1 {
		ancestors[left], ancestors[right] = ancestors[right], ancestors[left]
	}
	seen := make(map[string]struct{})
	out := make([]FlagInfo, 0, 8)
	for _, ancestor := range ancestors {
		for _, info := range buildFlagInfos(ancestor.persistentFlags, true) {
			key := info.Name
			if key == "" {
				key = "-" + info.Shorthand
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, info)
		}
	}
	return out
}

func buildFlagInfos(set *FlagSet, persistent bool) []FlagInfo {
	if set == nil || len(set.specs) == 0 {
		return nil
	}
	out := make([]FlagInfo, 0, len(set.specs))
	for _, spec := range set.specs {
		if spec == nil {
			continue
		}
		out = append(out, FlagInfo{
			Name:       spec.name,
			Shorthand:  spec.shorthand,
			Type:       spec.kind.helpType(),
			Usage:      spec.usage,
			Default:    spec.defValue,
			Persistent: persistent,
		})
	}
	return out
}

type flagRegistry struct {
	long  map[string]*flagSpec
	short map[string]*flagSpec
}

func toRegistry(set *FlagSet) *flagRegistry {
	reg := &flagRegistry{
		long:  make(map[string]*flagSpec),
		short: make(map[string]*flagSpec),
	}
	if set == nil {
		return reg
	}
	for _, spec := range set.specs {
		if spec == nil {
			continue
		}
		if spec.name != "" {
			reg.long[spec.name] = spec
		}
		if spec.shorthand != "" {
			reg.short[spec.shorthand] = spec
		}
	}
	return reg
}

func mergeFlagRegistries(registries ...*flagRegistry) *flagRegistry {
	merged := &flagRegistry{
		long:  make(map[string]*flagSpec),
		short: make(map[string]*flagSpec),
	}
	for _, reg := range registries {
		if reg == nil {
			continue
		}
		for name, spec := range reg.long {
			merged.long[name] = spec
		}
		for short, spec := range reg.short {
			merged.short[short] = spec
		}
	}
	return merged
}

func (r *flagRegistry) parsePrefix(args []string) ([]string, error) {
	idx := 0
	for idx < len(args) {
		token := args[idx]
		if token == "--" {
			return args[idx+1:], nil
		}
		if !strings.HasPrefix(token, "-") || token == "-" {
			return args[idx:], nil
		}
		next, err := r.consumeFlag(args, idx)
		if err != nil {
			return nil, err
		}
		idx = next
	}
	return nil, nil
}

func (r *flagRegistry) parseAll(args []string) ([]string, error) {
	positionals := make([]string, 0, len(args))
	idx := 0
	for idx < len(args) {
		token := args[idx]
		if token == "--" {
			positionals = append(positionals, args[idx+1:]...)
			return positionals, nil
		}
		if !strings.HasPrefix(token, "-") || token == "-" {
			positionals = append(positionals, token)
			idx++
			continue
		}
		next, err := r.consumeFlag(args, idx)
		if err != nil {
			return nil, err
		}
		idx = next
	}
	return positionals, nil
}

func (r *flagRegistry) consumeFlag(args []string, idx int) (int, error) {
	token := args[idx]
	if strings.HasPrefix(token, "--") {
		nameValue := strings.TrimPrefix(token, "--")
		name, value, hasValue := strings.Cut(nameValue, "=")
		spec, ok := r.long[name]
		if !ok {
			return 0, fmt.Errorf("unknown flag: --%s", name)
		}
		if !hasValue && spec.kind == flagKindBool {
			value = "true"
		} else if !hasValue {
			if idx+1 >= len(args) {
				return 0, fmt.Errorf("flag needs an argument: --%s", name)
			}
			value = args[idx+1]
			idx++
		}
		if err := applyFlagValue(spec, value); err != nil {
			return 0, err
		}
		return idx + 1, nil
	}

	short := strings.TrimPrefix(token, "-")
	if short == "" {
		return 0, fmt.Errorf("invalid flag syntax: %s", token)
	}

	name := short
	var value string
	hasInlineValue := false
	if strings.Contains(short, "=") {
		name, value, _ = strings.Cut(short, "=")
		hasInlineValue = true
	} else if len(short) > 1 {
		name = short[:1]
		value = short[1:]
		hasInlineValue = true
	}

	spec, ok := r.short[name]
	if !ok {
		return 0, fmt.Errorf("unknown shorthand flag: -%s", name)
	}
	if !hasInlineValue && spec.kind == flagKindBool {
		value = "true"
	} else if !hasInlineValue {
		if idx+1 >= len(args) {
			return 0, fmt.Errorf("flag needs an argument: -%s", name)
		}
		value = args[idx+1]
		idx++
	}

	if err := applyFlagValue(spec, value); err != nil {
		return 0, err
	}
	return idx + 1, nil
}

func applyFlagValue(spec *flagSpec, value string) error {
	if spec == nil {
		return errors.New("invalid flag definition")
	}
	trimmed := strings.TrimSpace(value)

	switch spec.kind {
	case flagKindString:
		if spec.stringDst == nil {
			return errors.New("string flag has no target")
		}
		*spec.stringDst = trimmed
		return nil
	case flagKindInt:
		if spec.intDst == nil {
			return errors.New("int flag has no target")
		}
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			if spec.name != "" {
				return fmt.Errorf("invalid argument %q for --%s", value, spec.name)
			}
			return fmt.Errorf("invalid integer value: %s", value)
		}
		*spec.intDst = parsed
		return nil
	case flagKindBool:
		if spec.boolDst == nil {
			return errors.New("bool flag has no target")
		}
		parsed, err := strconv.ParseBool(trimmed)
		if err != nil {
			if spec.name != "" {
				return fmt.Errorf("invalid argument %q for --%s", value, spec.name)
			}
			return fmt.Errorf("invalid boolean value: %s", value)
		}
		*spec.boolDst = parsed
		return nil
	default:
		return errors.New("unsupported flag type")
	}
}
