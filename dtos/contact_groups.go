package dtos

type ContactGroupReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ContactGroupUpdateReq struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type ContactGroup struct {
	ID            string `json:"id"`
	ClientId      string `json:"clientId"`
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	TotalContacts int    `json:"totalContacts"`
	CreatedBy     string `json:"createdBy,omitempty"`
	UpdatedBy     string `json:"updatedBy,omitempty"`
	CreatedAt     string `json:"createdAt,omitempty"`
	UpdatedAt     string `json:"updatedAt,omitempty"`
}

type ContactGroupMemberReq struct {
	ContactIds []string `json:"contactIds"`
}

type ContactGroupMemberResp struct {
	GroupId      string `json:"groupId"`
	AddedCount   int    `json:"addedCount"`
	SkippedCount int    `json:"skippedCount"`
}
