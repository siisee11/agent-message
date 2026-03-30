package ralphloop

import (
	"regexp"
	"strings"
	"unicode"
)

type sanitizationResult struct {
	Text    string
	Changed bool
	Reasons []string
}

var rolePrefixPattern = regexp.MustCompile(`(?im)^([ \t]*)(system|assistant|user|developer|tool)([ \t]*:)`)

func sanitizeUntrustedText(value string) sanitizationResult {
	reasons := make([]string, 0, 2)
	sanitized := stripUnsafeControlChars(value)
	if sanitized != value {
		reasons = append(reasons, "removed_control_chars")
	}

	neutralized := strings.NewReplacer(
		"\u2028", " ",
		"\u2029", " ",
		"<system>", "[system]",
		"</system>", "[/system]",
		"<assistant>", "[assistant]",
		"</assistant>", "[/assistant]",
		"<user>", "[user]",
		"</user>", "[/user]",
		"<developer>", "[developer]",
		"</developer>", "[/developer]",
		"<tool>", "[tool]",
		"</tool>", "[/tool]",
	).Replace(sanitized)
	neutralized = rolePrefixPattern.ReplaceAllString(neutralized, "${1}[$2]${3}")
	if neutralized != sanitized {
		reasons = append(reasons, "neutralized_prompt_markers")
	}

	trimmed := strings.TrimSpace(neutralized)
	changed := trimmed != value
	return sanitizationResult{
		Text:    trimmed,
		Changed: changed,
		Reasons: uniqueStrings(reasons),
	}
}

func stripUnsafeControlChars(value string) string {
	var builder strings.Builder
	for _, r := range value {
		switch {
		case r == '\n' || r == '\t':
			builder.WriteRune(r)
		case unicode.IsControl(r):
			continue
		default:
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func applySanitizationMetadata(record map[string]any, result sanitizationResult) {
	record["sanitized"] = result.Changed
	if result.Changed && len(result.Reasons) > 0 {
		record["sanitization_changes"] = result.Reasons
	}
}
