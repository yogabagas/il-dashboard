package dtos

type ContactReq struct {
	Phone        string                 `json:"phone"`
	Email        string                 `json:"email"`
	Name         string                 `json:"name"`
	Tags         []string               `json:"tags"`
	ClientId     string                 `json:"clientId"`
	CustomFields map[string]interface{} `json:"customFields"`
}

type ContactUpdateReq struct {
	Phone        string                 `json:"phone,omitempty"`
	Email        string                 `json:"email,omitempty"`
	Name         string                 `json:"name,omitempty"`
	Tags         []string               `json:"tags,omitempty"`
	CustomFields map[string]interface{} `json:"customFields,omitempty"`
	Status       string                 `json:"status,omitempty"`
	ClientId     string                 `json:"clientId"`
}

type Contact struct {
	ID             string                 `json:"id"`
	ClientId       string                 `json:"clientId"`
	Phone          string                 `json:"phone,omitempty"`
	Email          string                 `json:"email,omitempty"`
	Name           string                 `json:"name"`
	Tags           []string               `json:"tags,omitempty"`
	CustomFields   map[string]interface{} `json:"customFields,omitempty"`
	Status         string                 `json:"status"`
	UnsubscribedAt string                 `json:"unsubscribedAt,omitempty"`
	CreatedBy      string                 `json:"createdBy,omitempty"`
	UpdatedBy      string                 `json:"updatedBy,omitempty"`
	CreatedAt      string                 `json:"createdAt,omitempty"`
	UpdatedAt      string                 `json:"updatedAt,omitempty"`
}

type ContactImportReq struct {
	FileData []ContactImportRow `json:"fileData"`
}

type ContactImportRow struct {
	Phone        string                 `json:"phone"`
	Email        string                 `json:"email"`
	Name         string                 `json:"name"`
	ClientId     string                 `json:"clientId"`
	Tags         string                 `json:"tags"` // comma-separated
	CustomFields map[string]interface{} `json:"customFields"`
}

type ContactImportResp struct {
	TotalRows      int      `json:"totalRows"`
	SuccessCount   int      `json:"successCount"`
	FailedCount    int      `json:"failedCount"`
	FailedRows     []int    `json:"failedRows,omitempty"`
	FailedMessages []string `json:"failedMessages,omitempty"`
	ClientId       string   `json:"clientId"`
}

type ContactUploadFileReq struct {
	CampaignId string `json:"campaignId" form:"campaignId"`
}

type ContactUploadFileResp struct {
	TotalRows           int      `json:"totalRows"`
	ContactsCreated     int      `json:"contactsCreated"`
	TargetsCreated      int      `json:"targetsCreated"`
	FailedCount         int      `json:"failedCount"`
	FailedRows          []int    `json:"failedRows,omitempty"`
	FailedMessages      []string `json:"failedMessages,omitempty"`
	CampaignId          string   `json:"campaignId"`
	CacheKey            string   `json:"cacheKey"`
}
