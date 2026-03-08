package services

import (
	"encoding/base64"
	"sync"

	"github.com/ariandi/gocom"
	"github.com/ariandi/gocom/logger"
	"github.com/ariandi/gocom/pubsub"
	"github.com/jinzhu/copier"
	common "gitlab.com/anti_metter/switching_common"
	"gitlab.com/anti_metter/switching_common/auth"
	"gitlab.com/anti_metter/switching_common/client"
	"gitlab.com/anti_metter/switching_common/role"
	"gitlab.com/anti_metter/switching_common/user"
	"gitlab.com/bot3342545/il-dashboard/constans"
	"gitlab.com/bot3342545/il-dashboard/dtos"
	"golang.org/x/crypto/bcrypt"
)

type UserService interface {
	Create(req dtos.UserReq, authInfo auth.AuthInfo) (*dtos.User, *gocom.CodedError)
	Get(id string, authInfo auth.AuthInfo) (*dtos.User, *gocom.CodedError)
	Update(id string, req dtos.UserReq, authInfo auth.AuthInfo) (*dtos.User, *gocom.CodedError)
	Delete(id string, authInfo auth.AuthInfo) *gocom.CodedError
	Search(filter, clientId, roleId string, pageNo, rowPerPage int, info auth.AuthInfo) ([]dtos.User, bool, int64)
}

type UserServiceImpl struct{}

func (o *UserServiceImpl) Create(req dtos.UserReq, authInfo auth.AuthInfo) (*dtos.User, *gocom.CodedError) {
	existing := user.GetSvc().GetByEmail(req.Email)
	if existing != nil {
		return nil, common.ERR_ALREADY_EXIST
	}

	if err := req.Validate(); err != nil {
		return nil, err
	}

	logger.Infof("[UserServiceImpl Create] - client is: %s \n", authInfo.ClientId)
	if authInfo.ClientId != common.PROVIDER_ID {
		req.ClientId = authInfo.ClientId
	}

	mdl := user.User{}
	_ = copier.Copy(&mdl, req)

	hashed, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	mdl.Password = base64.StdEncoding.EncodeToString(hashed)

	mdl.CreatedBy = authInfo.UserId
	mdl.UpdatedBy = authInfo.UserId

	err := user.GetRepo().Create(&mdl).Error
	if err != nil {
		return nil, common.ERR_UNABLE_TO_CREATE
	}

	//_ = repo.GetUserRepo().SetRole(mdl.ID, req.Roles)
	_ = user.GetRepo().ClearByUser(mdl.ID)
	_ = user.GetRepo().SetRole(mdl.ID, req.Roles)

	ret := &dtos.User{}
	_ = copier.Copy(ret, mdl)

	cl := client.GetSvc().GetById(req.ClientId)
	if cl != nil {
		ret.ClientName = cl.Name
	}

	var roles []dtos.Role
	userRoleList := user.GetRepo().ListByUser(mdl.ID)
	for _, userRole := range userRoleList {
		rl := role.GetRepo().GetByCode(userRole.RoleId)
		rlDto := dtos.Role{}
		_ = copier.Copy(&rlDto, rl)

		roles = append(roles, rlDto)
	}

	ret.Roles = roles

	_ = pubsub.Get().Publish("update_user", mdl.ID)

	return ret, nil
}

func (o *UserServiceImpl) Update(id string, req dtos.UserReq, authInfo auth.AuthInfo) (*dtos.User, *gocom.CodedError) {
	mdl := user.GetSvc().GetById(id)
	if mdl == nil {
		return nil, common.ERR_NOT_FOUND
	}

	mdl.Name = editVal(req.Name, mdl.Name)
	mdl.Phone = editVal(req.Phone, mdl.Phone)
	mdl.UpdatedBy = authInfo.UserId

	if req.Nik != nil {
		mdl.Nik = editVal(*req.Nik, mdl.Nik)
	}

	if req.Email != "" && req.Email != mdl.Email {
		existing := user.GetSvc().GetByEmail(req.Email)
		if existing != nil {
			return nil, common.ERR_ALREADY_EXIST
		}

		mdl.Email = editVal(req.Email, mdl.Email)
	}

	if req.Password != "" {
		existingPassword := user.GetRepo().GetById(id)
		hashedPassword, _ := base64.StdEncoding.DecodeString(existingPassword.Password)
		err := bcrypt.CompareHashAndPassword(hashedPassword, []byte(req.OldPassword))
		if err != nil {
			return nil, constans.ErrInvalidOldPassword
		}

		hashed, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		mdl.Password = base64.StdEncoding.EncodeToString(hashed)
	}

	ret := &dtos.User{}
	_ = copier.Copy(ret, mdl)

	cl := client.GetSvc().GetById(mdl.ClientId)

	if cl != nil {
		ret.ClientName = cl.Name
	}

	//err := repo.GetUserRepo().Update(mdl)
	err := user.GetRepo().Update(mdl).Error
	if err != nil {
		return nil, common.ERR_UNABLE_TO_UPDATE
	}

	if len(req.Roles) > 0 {
		_ = user.GetRepo().ClearByUser(mdl.ID)
		_ = user.GetRepo().SetRole(mdl.ID, req.Roles)
	}

	userRoleList := user.GetUserRoleRepo().ListByUsers(mdl.ID)
	var roles []dtos.Role
	roles = []dtos.Role{}

	for _, userRole := range userRoleList {
		rl := role.GetSvc().GetById(userRole.RoleId)
		rlDto := dtos.Role{}
		_ = copier.Copy(&rlDto, rl)

		roles = append(roles, rlDto)
	}

	ret.Roles = roles

	_ = pubsub.Get().Publish("update_user", id)

	return ret, nil
}

func (o *UserServiceImpl) Get(id string, authInfo auth.AuthInfo) (*dtos.User, *gocom.CodedError) {

	mdl := user.GetRepo().GetById(id)
	if mdl == nil {
		return nil, common.ERR_NOT_FOUND
	}

	//if authInfo.ClientId != common.PROVIDER_ID && mdl.ClientId != authInfo.ClientId {
	//	return nil, common.ERR_NOT_FOUND
	//}

	ret := &dtos.User{}
	_ = copier.Copy(ret, mdl)

	cl := client.GetRepo().GetById(mdl.ClientId)
	if cl != nil {
		ret.ClientName = cl.Name
	}

	userRoleList := user.GetRepo().ListByUser(mdl.ID)
	var roles []dtos.Role

	for _, userRole := range userRoleList {

		rl := role.GetRepo().GetById(userRole.RoleId)
		rlDto := dtos.Role{}
		_ = copier.Copy(&rlDto, rl)

		roles = append(roles, rlDto)
	}

	ret.Roles = roles

	return ret, nil
}

func (o *UserServiceImpl) Delete(id string, authInfo auth.AuthInfo) *gocom.CodedError {

	mdl := user.GetRepo().GetById(id)
	if mdl == nil {
		return common.ERR_NOT_FOUND
	}

	isAdmin := o.GetUserPermission(authInfo)
	if !isAdmin {
		userAuth := user.GetSvc().GetById(authInfo.UserId)
		if userAuth.ClientId != mdl.ClientId {
			logger.Info("[UserServiceImpl Delete] - user not allowed to delete")
			return common.ERR_UNABLE_TO_DELETE
		}
	}

	err := user.GetRepo().Delete(id)

	if err != nil {
		logger.Info("[UserServiceImpl Delete] - error delete user", err)
		return common.ERR_UNABLE_TO_DELETE
	}

	_ = pubsub.Get().Publish("update_user", id)

	return nil
}

func (o *UserServiceImpl) Search(filter, clientId, roleId string, pageNo, rowPerPage int, info auth.AuthInfo) ([]dtos.User, bool, int64) {
	// redeploy
	ret := []dtos.User{}

	list, haveNext, count := user.GetRepo().Search(filter, clientId, roleId, pageNo, rowPerPage)

	roles, userRoleMap := o.rolesForUsers(list)

	// roles to map
	roleMap := map[string]role.Role{}
	for _, rl := range roles {
		roleMap[rl.ID] = rl
	}

	for _, item := range list {
		us := dtos.User{}
		copier.Copy(&us, item)

		rls := o.roleForUser(userRoleMap, item.ID, roleMap)
		us.Roles = rls

		cl := client.GetSvc().GetById(item.ClientId)

		if cl != nil {
			us.ClientName = cl.Name
		}

		ret = append(ret, us)

	}

	return ret, haveNext, count
}

func (o *UserServiceImpl) roleForUser(userRoleMap map[string][]string, userId string, roleMap map[string]role.Role) []dtos.Role {
	var rls []dtos.Role
	if _, ok := userRoleMap[userId]; ok {
		for _, rlId := range userRoleMap[userId] {
			rl := roleMap[rlId]
			rlDto := dtos.Role{}
			copier.Copy(&rlDto, rl)

			rls = append(rls, rlDto)
		}
	}
	return rls
}

func (o *UserServiceImpl) rolesForUsers(list []user.User) ([]role.Role, map[string][]string) {
	ids := []string{}
	for _, u := range list {
		ids = append(ids, u.ID)
	}

	roles := role.GetRepo().GetRolesForUsers(ids...)
	userRoleList := user.GetUserRoleRepo().ListByUsers(ids...)

	// userRoleList to map
	userRoleMap := map[string][]string{}
	for _, url := range userRoleList {
		if _, ok := userRoleMap[url.UserId]; !ok {
			userRoleMap[url.UserId] = []string{}
		}

		userRoleMap[url.UserId] = append(userRoleMap[url.UserId], url.RoleId)
	}
	return roles, userRoleMap
}

func (o *UserServiceImpl) GetUserPermission(authInfo auth.AuthInfo) bool {
	isAdmin := false

	userRoles := user.GetRepo().ListRole(authInfo.UserId)
	if userRoles == nil {
		logger.Info("[UserServiceImpl GetUserPermission] - userRoles is nil")
		return false
	}

	for _, userRole := range userRoles {
		if userRole.Code == auth.ADMIN {
			isAdmin = true
			break
		}
	}

	return isAdmin
}

func editVal(reqVal, mdlVal string) string {

	if reqVal == "" {
		return mdlVal
	}

	return reqVal
}

//----------------------------------------

var userService UserService
var userServiceOnce sync.Once

func SetUserService(svc UserService) {
	userService = svc
}

func GetUserService() UserService {
	if userService == nil {
		userServiceOnce.Do(func() {
			tmp := &UserServiceImpl{}
			userService = tmp
		})
	}

	return userService
}
