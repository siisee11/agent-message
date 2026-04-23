package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

const howtoDirEnv = "AGENT_MESSAGE_HOWTO_DIR"

func newHowtoCommand(rt *Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "howto",
		Short: "Read bundled how-to guides",
	}

	cmd.AddCommand(
		newHowtoListCommand(rt),
		newHowtoReadCommand(rt),
	)

	return cmd
}

func newHowtoListCommand(rt *Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available how-to guides",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runHowtoList(rt)
		},
	}
}

func newHowtoReadCommand(rt *Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "read <filename>",
		Short: "Print a how-to guide",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runHowtoRead(rt, args[0])
		},
	}
}

func runHowtoList(rt *Runtime) error {
	if rt == nil || rt.Stdout == nil {
		return errors.New("runtime stdout is required")
	}

	dir, err := resolveHowtoDir()
	if err != nil {
		return err
	}

	files, err := listHowtoFiles(dir)
	if err != nil {
		return err
	}

	if rt.JSONOutput {
		return writeJSON(rt.Stdout, map[string]any{
			"files": files,
		})
	}

	for _, file := range files {
		if _, err := fmt.Fprintln(rt.Stdout, file); err != nil {
			return err
		}
	}
	return nil
}

func runHowtoRead(rt *Runtime, filename string) error {
	if rt == nil || rt.Stdout == nil {
		return errors.New("runtime stdout is required")
	}

	name, err := normalizeHowtoFilename(filename)
	if err != nil {
		return err
	}

	dir, err := resolveHowtoDir()
	if err != nil {
		return err
	}

	path := filepath.Join(dir, name)
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("unknown how-to file %q; run `agent-message howto list`", name)
		}
		return fmt.Errorf("read how-to file %q: %w", name, err)
	}
	content := string(contentBytes)

	if rt.JSONOutput {
		return writeJSON(rt.Stdout, map[string]any{
			"file":    name,
			"content": content,
		})
	}

	if _, err := fmt.Fprint(rt.Stdout, content); err != nil {
		return err
	}
	if !strings.HasSuffix(content, "\n") {
		_, err = fmt.Fprintln(rt.Stdout)
		return err
	}
	return nil
}

func resolveHowtoDir() (string, error) {
	if configured := strings.TrimSpace(os.Getenv(howtoDirEnv)); configured != "" {
		if isDir(configured) {
			return configured, nil
		}
		return "", fmt.Errorf("%s does not point to a directory: %s", howtoDirEnv, configured)
	}

	if cwd, err := os.Getwd(); err == nil {
		if dir, ok := findHowtoDirUpward(cwd); ok {
			return dir, nil
		}
	}

	if executable, err := os.Executable(); err == nil {
		if dir, ok := findHowtoDirUpward(filepath.Dir(executable)); ok {
			return dir, nil
		}
	}

	return "", errors.New("how-to directory not found")
}

func findHowtoDirUpward(start string) (string, bool) {
	dir, err := filepath.Abs(start)
	if err != nil {
		dir = filepath.Clean(start)
	}

	for {
		candidate := filepath.Join(dir, "howto")
		if isDir(candidate) {
			return candidate, true
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func listHowtoFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read how-to directory: %w", err)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) != ".md" {
			continue
		}
		files = append(files, name)
	}
	sort.Strings(files)
	return files, nil
}

func normalizeHowtoFilename(filename string) (string, error) {
	name := strings.TrimSpace(filename)
	if name == "" {
		return "", errors.New("how-to filename is required")
	}
	if filepath.IsAbs(name) || strings.ContainsAny(name, `/\`) {
		return "", errors.New("how-to filename must be a file name, not a path")
	}
	if name == "." || name == ".." || strings.Contains(name, "..") {
		return "", errors.New("how-to filename must not contain dot segments")
	}
	if filepath.Ext(name) == "" {
		name += ".md"
	}
	if filepath.Ext(name) != ".md" {
		return "", errors.New("how-to filename must be a Markdown file")
	}
	return name, nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
