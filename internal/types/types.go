package types

// NOTE: These types are intentionally duplicated (manually) from the React frontend
// (`vibes/2025-12-15/agent-ui-system/client/src/types/schemas.ts`).
//
// We will introduce schema/codegen later; for now we keep wire-compat JSON fields.

type RequestStatus string

const (
	StatusPending   RequestStatus = "pending"
	StatusCompleted RequestStatus = "completed"
	StatusTimeout   RequestStatus = "timeout"
	StatusError     RequestStatus = "error"
)

type WidgetType string

const (
	WidgetConfirm WidgetType = "confirm"
	WidgetSelect  WidgetType = "select"
	WidgetForm    WidgetType = "form"
	WidgetUpload  WidgetType = "upload"
	WidgetTable   WidgetType = "table"
)

// UIRequest is the canonical request object exchanged between CLI/server/frontend.
//
// Note: We keep `Input`/`Output` as `any` for now to preserve flexibility.
// The CLI can marshal/unmarshal to the typed *Input/*Output structs where desired.
type UIRequest struct {
	ID          string        `json:"id"`
	Type        WidgetType    `json:"type"`
	SessionID   string        `json:"sessionId"`
	Input       any           `json:"input"`
	Output      any           `json:"output,omitempty"`
	Status      RequestStatus `json:"status"`
	CreatedAt   string        `json:"createdAt"`
	CompletedAt *string       `json:"completedAt,omitempty"`
	ExpiresAt   string        `json:"expiresAt"`
	Error       *string       `json:"error,omitempty"`
}

type ConfirmInput struct {
	Title       string  `json:"title"`
	Message     *string `json:"message,omitempty"`
	ApproveText *string `json:"approveText,omitempty"`
	RejectText  *string `json:"rejectText,omitempty"`
}

type ConfirmOutput struct {
	Approved  bool   `json:"approved"`
	Timestamp string `json:"timestamp"`
}

type SelectInput struct {
	Title      string   `json:"title"`
	Options    []string `json:"options"`
	Multi      *bool    `json:"multi,omitempty"`
	Searchable *bool    `json:"searchable,omitempty"`
}

type SelectOutput struct {
	Selected any `json:"selected"` // string | []string
}

type FormInput struct {
	Title  string `json:"title"`
	Schema any    `json:"schema"` // JSON Schema
}

type FormOutput struct {
	Data any `json:"data"`
}

type UploadInput struct {
	Title       string   `json:"title"`
	Accept      []string `json:"accept,omitempty"`
	Multiple    *bool    `json:"multiple,omitempty"`
	MaxSize     *int64   `json:"maxSize,omitempty"`
	CallbackURL *string  `json:"callbackUrl,omitempty"`
}

type UploadOutput struct {
	Files []UploadedFile `json:"files"`
}

type UploadedFile struct {
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	Path     string `json:"path"`
	MimeType string `json:"mimeType"`
}

type TableInput struct {
	Title       string   `json:"title"`
	Data        []any    `json:"data"`
	Columns     []string `json:"columns,omitempty"`
	MultiSelect *bool    `json:"multiSelect,omitempty"`
	Searchable  *bool    `json:"searchable,omitempty"`
}

type TableOutput struct {
	Selected any `json:"selected"` // any | []any
}
