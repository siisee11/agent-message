package models

// CatalogPromptResponse is returned by GET /api/catalog/prompt.
type CatalogPromptResponse struct {
	Prompt string `json:"prompt"`
}
