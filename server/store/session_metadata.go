package store

import (
	"strings"

	"agent-message/server/models"
)

const (
	cwdPrefix      = "CWD:"
	hostnamePrefix = "Hostname:"
)

func populateConversationSessionMetadata(summary *models.ConversationSummary, sessionText string) {
	sessionFolder, sessionHostname := extractSessionMetadata(sessionText)
	summary.SessionFolder = sessionFolder
	summary.SessionHostname = sessionHostname
}

func extractSessionMetadata(value string) (string, string) {
	var cwd string
	var hostname string

	for _, line := range strings.Split(value, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, cwdPrefix):
			cwd = strings.TrimSpace(strings.TrimPrefix(trimmed, cwdPrefix))
		case strings.HasPrefix(trimmed, hostnamePrefix):
			hostname = strings.TrimSpace(strings.TrimPrefix(trimmed, hostnamePrefix))
		}
	}

	return lastPathSegment(cwd), hostname
}

func lastPathSegment(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	withoutTrailingSeparators := strings.TrimRight(trimmed, `/\`)
	if withoutTrailingSeparators == "" {
		return trimmed
	}

	lastSeparator := strings.LastIndexAny(withoutTrailingSeparators, `/\`)
	if lastSeparator < 0 {
		return withoutTrailingSeparators
	}

	return withoutTrailingSeparators[lastSeparator+1:]
}
