package entity

import (
	"time"
)

type HistoryEntry struct {
	ID             string            `db:"id" json:"id"`
	TemplateName   string            `db:"template_name" json:"template_name"`
	Variables      map[string]string `db:"-" json:"variables"`
	VariablesJSON  string            `db:"variables_json" json:"-"`
	ExpandedPrompt string            `db:"expanded_prompt" json:"expanded_prompt"`
	Completion     string            `db:"completion" json:"completion"`
	PromptTokens   int               `db:"prompt_tokens" json:"prompt_tokens"`
	CompTokens     int               `db:"completion_tokens" json:"completion_tokens"`
	TotalTokens    int               `db:"total_tokens" json:"total_tokens"`
	Pinned         bool              `db:"pinned" json:"pinned"`
	CreatedAt      time.Time         `db:"created_at" json:"created_at"`
}
