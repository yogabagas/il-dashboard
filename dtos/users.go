package dtos

import (
	"strings"
	"time"

	"github.com/ariandi/gocom"
	"gitlab.com/bot3342545/il-dashboard/constans"
)

type User struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Email      string    `json:"email" validate:"required,email"`
	Phone      string    `json:"phone"`
	ClientId   string    `json:"client_id"`
	ClientName string    `json:"client_name"`
	Roles      []Role    `json:"roles"`
	FullName   string    `json:"full_name"`
	Nik        string    `json:"nik"`
	CreatedAt  time.Time `json:"created_at"`
}

type UserReq struct {
	Name        string   `json:"name" validate:"required"`
	Email       string   `json:"email" validate:"required,email"`
	Password    string   `json:"password" validate:"required"`
	Phone       string   `json:"phone"`
	ClientId    string   `json:"client_id"`
	OldPassword string   `json:"old_password,omitempty"`
	Roles       []string `json:"roles"`
	Nik         *string  `json:"nik,omitempty"`
}

func (u *UserReq) Validate() *gocom.CodedError {
	if u == nil {
		return constans.ErrRequestEmpty
	}

	if strings.ToLower(u.Email) == "" {
		return constans.ErrEmailEmpty
	}

	if strings.ToLower(u.Name) == "" {
		return constans.ErrFullNameEmpty
	}

	if strings.ToLower(u.ClientId) == "" {
		return constans.ErrClientEmpty
	}

	if strings.ToLower(u.Password) == "" {
		return constans.ErrPasswordEmpty
	}

	return nil
}

type UserMediaRequest struct {
	TableName string
	RefId     string
	MediaName string
	FileType  string
	Media     string
}

type CustomerRequest struct {
	UserId     string
	NoCustomer string
}

type UserMedia struct {
	ID string
	UserMediaRequest
}

type Customer struct {
	ID          string
	UserId      string
	NoCustomer  string
	Address     string
	City        string
	SubDistrict string
	District    string
}
