package dtos

type SenderReq struct {
	ClientId     string                 `json:"clientId,omitempty"`
	Type         string                 `json:"type"` // whatsapp, sms, email
	Name         string                 `json:"name"`
	Identifier   string                 `json:"identifier"`
	Config       map[string]interface{} `json:"config"`
	DailyLimit   int                    `json:"dailyLimit"`
	MonthlyLimit int                    `json:"monthlyLimit"`
}

type SenderUpdateReq struct {
	ClientId     string                 `json:"clientId,omitempty"`
	Name         string                 `json:"name,omitempty"`
	Identifier   string                 `json:"identifier,omitempty"`
	Config       map[string]interface{} `json:"config,omitempty"`
	Status       string                 `json:"status,omitempty"`
	IsVerified   bool                   `json:"isVerified,omitempty"`
	DailyLimit   int                    `json:"dailyLimit,omitempty"`
	MonthlyLimit int                    `json:"monthlyLimit,omitempty"`
}

type Sender struct {
	ID            string                 `json:"id"`
	ClientId      string                 `json:"clientId"`
	Type          string                 `json:"type"`
	Name          string                 `json:"name"`
	Identifier    string                 `json:"identifier"`
	Config        map[string]interface{} `json:"config,omitempty"`
	Status        string                 `json:"status"`
	IsVerified    bool                   `json:"isVerified"`
	DailyLimit    int                    `json:"dailyLimit"`
	MonthlyLimit  int                    `json:"monthlyLimit"`
	CreatedBy     string                 `json:"createdBy,omitempty"`
	UpdatedBy     string                 `json:"updatedBy,omitempty"`
	CreatedAt     string                 `json:"createdAt,omitempty"`
	UpdatedAt     string                 `json:"updatedAt,omitempty"`
	PhoneNumberId string                 `json:"phoneNumberId,omitempty"`
}

type MetaSenderReq struct {
	WabaID             string `json:"-"`
	CountryCode        string `json:"cc"`                   // Country dial code of the phone number (for example, 1).
	MigratePhoneNumber bool   `json:"migrate_phone_number"` // Set to true to migrate a registered WhatsApp Business Phone Number from one WhatsApp Business Account to another.
	PhoneNumber        string `json:"phone_number"`         // Phone number without the country code or plus symbol (+).
	PreverifiedID      string `json:"preverified_id"`       // Preverified ID related to this phone number
	VerifiedName       string `json:"verified_name"`        // Name of the business as it appears in the WhatsApp app or WhatsApp Business app profile.
}

type MetaSenderResp struct {
	ID string `json:"id"` // ID of the phone number
}

type RegisterPhoneNumberOnboardReq struct {
	MessagingProduct string `json:"messaging_product"`
	PhoneNumberID    string `json:"phone_number_id"`
	Pin              string `json:"pin"`
}

type RegisterPhoneNumberOnboardResp struct {
	Success bool `json:"success"`
}
