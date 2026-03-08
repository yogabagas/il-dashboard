package dtos

type LoginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRes struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Email            string `json:"email"`
	AccessToken      string `json:"accessToken"`
	RefreshToken     string `json:"refreshToken"`
	ClientId         string `json:"clientId"`
	ClientName       string `json:"clientName"`
	Provider         string `json:"provider,omitempty"`
	RemainingAttempt int    `json:"remaining_attempt"`
	Roles            []Role `json:"roles"`
}

type RefreshTokenReq struct {
	RefreshToken string `json:"refreshToken"`
}
