package entity

type VarType string

const (
	VarTypeString  VarType = "string"
	VarTypeText    VarType = "text"
	VarTypeSelect  VarType = "select"
	VarTypeConfirm VarType = "confirm"
)

type TemplateVar struct {
	Name        string   `json:"name"`
	Type        VarType  `json:"type"`
	Description string   `json:"description"`
	Default     string   `json:"default"`
	Options     []string `json:"options,omitempty"` // select 型の場合の選択肢
}
