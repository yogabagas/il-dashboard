package dtos

type WATemplateUploadFileReq struct {
	FileName string `json:"fileName"`
	File     string `json:"file"`
}

type WATemplateReq struct {
	Name       string                `json:"name"`       // Display name (can have spaces, special chars)
	WATemplate string                `json:"waTemplate"` // WhatsApp template name (lowercase, underscores only, max 50 chars)
	Category   string                `json:"category"`
	Language   string                `json:"language"`
	ClientId   string                `json:"clientId"`
	Components []WATemplateComponent `json:"components"`
	Status     string                `json:"status,omitempty"`
	Media      string                `json:"media,omitempty"`
}

type WATemplateUpdateReq struct {
	Name       string                `json:"name,omitempty"`
	WATemplate string                `json:"waTemplate,omitempty"`
	Category   string                `json:"category,omitempty"`
	Language   string                `json:"language,omitempty"`
	Components []WATemplateComponent `json:"components,omitempty"`
	Status     string                `json:"status,omitempty"`
	Media      string                `json:"media,omitempty"`
}

type WATemplateComponent struct {
	Type                      string                      `json:"type"`
	AddSecurityRecommendation bool                        `json:"add_security_recommendation,omitempty"`
	Format                    string                      `json:"format,omitempty"` // IMAGE, VIDEO, DOCUMENT for HEADER
	Text                      string                      `json:"text,omitempty"`
	Buttons                   []WATemplateButton          `json:"buttons,omitempty"`
	Examples                  []string                    `json:"examples,omitempty"` // For BODY text parameters
	Example                   *WATemplateComponentExample `json:"example,omitempty"`  // For HEADER media (WhatsApp API format)
}

type WATemplateComponentExample struct {
	HeaderHandle string `json:"header_handle,omitempty"` // File handle for IMAGE/VIDEO/DOCUMENT header
}

type WATemplateButton struct {
	Type        string `json:"type"`
	Text        string `json:"text"`
	URL         string `json:"url,omitempty"`
	PhoneNumber string `json:"phoneNumber,omitempty"`
}

type WATemplate struct {
	ID             string                `json:"id"`
	Name           string                `json:"name"`
	WATemplate     string                `json:"waTemplate"`
	Category       string                `json:"category"`
	Language       string                `json:"language"`
	ClientId       string                `json:"clientId"`
	ClientName     string                `json:"clientName,omitempty"`
	Components     []WATemplateComponent `json:"components"`
	Status         string                `json:"status"`
	WAStatus       string                `json:"waStatus,omitempty"`
	WATemplateId   string                `json:"waTemplateId,omitempty"`
	RejectedReason string                `json:"rejectedReason,omitempty"`
	CreatedAt      string                `json:"createdAt,omitempty"`
	UpdatedAt      string                `json:"updatedAt,omitempty"`
	SubmittedAt    string                `json:"submittedAt,omitempty"`
	ApprovedAt     string                `json:"approvedAt,omitempty"`
	Media          string                `json:"media,omitempty"`
}
