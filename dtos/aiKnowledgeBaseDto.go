package dtos

import "time"

// AiKnowledgeBaseCreateReq - Request for creating new knowledge
type AiKnowledgeBaseCreateReq struct {
	ClientId string `json:"client_id" binding:"required"`
	Category string `json:"category" binding:"required"` // layanan, tagihan, pengaduan, administrasi, umum
	Question string `json:"question" binding:"required"`
	Answer   string `json:"answer" binding:"required"`
	Keywords string `json:"keywords"`
	Priority int    `json:"priority"` // Default 0
}

// AiKnowledgeBaseUpdateReq - Request for updating knowledge
type AiKnowledgeBaseUpdateReq struct {
	ID       string `json:"id" binding:"required"`
	Category string `json:"category"`
	Question string `json:"question"`
	Answer   string `json:"answer"`
	Keywords string `json:"keywords"`
	IsActive *bool  `json:"is_active"`
	Priority *int   `json:"priority"`
}

// AiKnowledgeBaseResp - Response for knowledge
type AiKnowledgeBaseResp struct {
	ID        string    `json:"id"`
	ClientId  string    `json:"client_id"`
	Category  string    `json:"category"`
	Question  string    `json:"question"`
	Answer    string    `json:"answer"`
	Keywords  string    `json:"keywords"`
	IsActive  bool      `json:"is_active"`
	Priority  int       `json:"priority"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedBy string    `json:"created_by"`
	UpdatedBy string    `json:"updated_by"`
}

// AiKnowledgeBaseSearchReq - Request for searching knowledge
type AiKnowledgeBaseSearchReq struct {
	Filter     string     `json:"filter"`
	ClientId   string     `json:"client_id"`
	Category   string     `json:"category"`
	IsActive   *bool      `json:"is_active"`
	DateFrom   *time.Time `json:"date_from"`
	DateTo     *time.Time `json:"date_to"`
	PageNo     int        `json:"page_no"`
	RowPerPage int        `json:"row_per_page"`
}

// AiKnowledgeBaseSearchResp - Response for search
type AiKnowledgeBaseSearchResp struct {
	Data     []AiKnowledgeBaseResp `json:"data"`
	HaveNext bool                  `json:"have_next"`
	Total    int64                 `json:"total"`
}
