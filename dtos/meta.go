package dtos

type CreatePhoneNumberToWABAReq struct {
	CountryCode        string `json:"cc"`                   // Country dial code of the phone number (for example, 1).
	MigratePhoneNumber bool   `json:"migrate_phone_number"` // Set to true to migrate a registered WhatsApp Business Phone Number from one WhatsApp Business Account to another.
	PhoneNumber        string `json:"phone_number"`         // Phone number without the country code or plus symbol (+).
	PreverifiedID      string `json:"preverified_id"`       // Preverified ID related to this phone number
	VerifiedName       string `json:"verified_name"`        // Name of the business as it appears in the WhatsApp app or WhatsApp Business app profile.
}

type CreatePhoneNumberToWABAResp struct {
	ID string `json:"id"` // ID of the phone number
}

type RequestOTPCodeReq struct {
	PhoneNumberId string `json:"-"`
	CodeMethode   string `json:"code_methode"` // Method of the code (for example, sms, email).
	Language      string `json:"language"`     // Language of the code (for example, en, id).
}

type RequestOTPCodeResp struct {
	ID string `json:"id"` // ID of the phone number
}

type VerifyOTPCodeReq struct {
	PhoneNumberId string `json:"-"`
	Code          string `json:"code"` // Code of the OTP (for example, 123456).
}

type VerifyOTPCodeResp struct {
	SuccessfulVerification struct {
		Summary string `json:"summary"`
		Value   struct {
			Success bool   `json:"success"`
			ID      string `json:"id"`
		} `json:"value"`
	} `json:"successful_verification"`
}

type ExchangeTokenReq struct {
	Code string `json:"code"`
}

type ExchangeTokenResp struct {
	AccessToken string `json:"access_token"`
}

type SubscribeWebhookReq struct {
	BusinessToken string `json:"business_token"`
	WabaId        string `json:"waba_id"`
}

type SubscribeWebhookResp struct {
	Success bool `json:"success"`
}
