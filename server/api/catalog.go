package api

import (
	_ "embed"
	"net/http"
	"strings"

	"agent-message/server/models"
)

var (
	//go:embed catalog_prompt.txt
	catalogPrompt string
)

type catalogHandler struct{}

func newCatalogHandler() *catalogHandler {
	return &catalogHandler{}
}

func (h *catalogHandler) handlePrompt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	writeJSON(w, http.StatusOK, models.CatalogPromptResponse{
		Prompt: strings.TrimSpace(catalogPrompt),
	})
}
