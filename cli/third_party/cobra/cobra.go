package cobra

import (
	"errors"
	"fmt"
	"os"
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
	mergedPersistent := mergeFlagRegistries(inheritedPersistent, toRegistry(c.persistentFlags))

	if len(c.subcommands) > 0 {
		remaining, err := mergedPersistent.parsePrefix(args)
		if err != nil {
			return c, err
		}
		if len(remaining) == 0 {
			if c.RunE == nil {
				return c, errors.New("command is required")
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

// FlagSet stores command flag definitions.
type FlagSet struct {
	specs []*flagSpec
}

type flagSpec struct {
	name      string
	shorthand string
	kind      flagKind
	stringDst *string
	intDst    *int
}

type flagKind int

const (
	flagKindString flagKind = iota + 1
	flagKindInt
)

func NewFlagSet() *FlagSet {
	return &FlagSet{specs: make([]*flagSpec, 0, 4)}
}

func (f *FlagSet) StringVar(dst *string, name, value, _ string) {
	if dst != nil {
		*dst = value
	}
	f.specs = append(f.specs, &flagSpec{
		name:      strings.TrimSpace(name),
		kind:      flagKindString,
		stringDst: dst,
	})
}

func (f *FlagSet) IntP(name, shorthand string, value int, _ string) *int {
	dst := new(int)
	*dst = value
	f.specs = append(f.specs, &flagSpec{
		name:      strings.TrimSpace(name),
		shorthand: strings.TrimSpace(shorthand),
		kind:      flagKindInt,
		intDst:    dst,
	})
	return dst
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
		if !hasValue {
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
	if !hasInlineValue {
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
	default:
		return errors.New("unsupported flag type")
	}
}
