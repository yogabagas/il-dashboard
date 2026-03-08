package services

import (
	"fmt"

	"github.com/ariandi/gocom"
	"github.com/ariandi/gocom/logger"
	"github.com/ariandi/gocom/pubsub"
	"gitlab.com/anti_metter/switching_common/auth"
	"gitlab.com/anti_metter/switching_common/role"
	rolemenu "gitlab.com/anti_metter/switching_common/roleMenu"
	rolepermission "gitlab.com/anti_metter/switching_common/rolePermission"
	"gitlab.com/bot3342545/il-dashboard/dtos"

	"slices"
	"strings"
	"sync"

	"github.com/jinzhu/copier"
	common "gitlab.com/anti_metter/switching_common"
)

type RoleService interface {
	Create(req dtos.RoleReq, authInfo auth.AuthInfo) (*dtos.Role, *gocom.CodedError)
	Update(id string, req dtos.RoleReq, authInfo auth.AuthInfo) (*dtos.Role, *gocom.CodedError)
	Search(filter, isClientRole string, pageNo, rowPerPage int) ([]dtos.Role, bool, int64)
	Delete(id string, authInfo auth.AuthInfo) *gocom.CodedError
	SetMenu(id string, req dtos.RoleMenuReq, authInfo auth.AuthInfo) ([]dtos.Menu, *gocom.CodedError)
	ListMenu(id, parentId string) []dtos.Menu
	TreeMenu(id string) []dtos.Menu
	SetPermission(id string, req dtos.RolePermissionReq, authInfo auth.AuthInfo) ([]dtos.Permission, *gocom.CodedError)
	ListPermission(id, moduleId string) []dtos.Permission
}

type RoleServiceImpl struct {
}

var defaultRole = []string{"owner", "agent", "admin"}

func (o *RoleServiceImpl) Create(req dtos.RoleReq, authInfo auth.AuthInfo) (*dtos.Role, *gocom.CodedError) {
	existing := role.GetRepo().GetByCode(req.Code)
	if existing != nil {
		return nil, common.ERR_ALREADY_EXIST
	}

	if authInfo.UserId == "" {
		authInfo.UserId = "system"
	}

	if req.Name == "" || strings.TrimSpace(req.Name) == "" {
		fmt.Println("unable create role with empty role name")
		return nil, common.ERR_UNABLE_TO_CREATE
	}

	mdl := role.Role{}
	_ = copier.Copy(&mdl, req)

	err := role.GetRepo().Create(&mdl).Error
	if err != nil {
		return nil, common.ERR_UNABLE_TO_CREATE
	}

	ret := dtos.Role{}
	_ = copier.Copy(&ret, mdl)

	_ = pubsub.Get().Publish("update_role", ret.ID)
	return &ret, nil
}

func (o *RoleServiceImpl) Update(id string, req dtos.RoleReq, authInfo auth.AuthInfo) (*dtos.Role, *gocom.CodedError) {

	if authInfo.ClientId != common.PROVIDER_ID {
		logger.Infof("[RoleService Update] unable to update role with client id %s \n", authInfo.ClientId)
		return nil, common.ERR_INVALID_REQUEST
	}

	existing := role.GetRepo().GetById(id)

	if existing == nil {
		return nil, common.ERR_NOT_FOUND
	}

	if req.Name == "" || strings.TrimSpace(req.Name) == "" {
		fmt.Println("unable create role with empty role name")
		return nil, common.ERR_UNABLE_TO_CREATE
	}

	if strings.ToUpper(existing.Code) != strings.ToUpper(req.Code) {
		checkRole := role.GetRepo().GetByCode(req.Code)
		if checkRole != nil {
			return nil, common.ERR_ALREADY_EXIST
		}
	}

	existing.Code = editVal(req.Code, existing.Code)
	existing.Name = editVal(req.Name, existing.Name)

	err := role.GetRepo().Update(existing)
	if err != nil {
		return nil, common.ERR_UNABLE_TO_UPDATE
	}

	_ = pubsub.Get().Publish("update_role", id)

	ret := dtos.Role{}
	_ = copier.Copy(&ret, existing)

	return &ret, nil
}

func (o *RoleServiceImpl) SetMenu(id string, req dtos.RoleMenuReq, authInfo auth.AuthInfo) ([]dtos.Menu, *gocom.CodedError) {

	if authInfo.ClientId != common.PROVIDER_ID {
		logger.Infof("[RoleService SetMenu] unable to update role with client id %s \n", authInfo.ClientId)
		return nil, common.ERR_INVALID_REQUEST
	}

	ret := []dtos.Menu{}
	roleRepo := role.GetRepo().GetById(id)

	if roleRepo == nil {
		return ret, common.ERR_NOT_FOUND
	}

	_ = rolemenu.GetRepo().ClearByRole(id)
	_ = rolemenu.GetRepo().SetRoleMenu(id, req.Menus)

	list := rolemenu.GetRepo().ListMenuByRole(id, "")
	for _, item := range list {
		mn := dtos.Menu{}
		_ = copier.Copy(&mn, item)

		ret = append(ret, mn)
	}

	return ret, nil
}

func (o *RoleServiceImpl) SubTreeMenu(id, parentId string) []dtos.Menu {

	ret := []dtos.Menu{}

	list := rolemenu.GetRepo().ListMenuByRole(id, parentId)

	for _, item := range list {

		mn := dtos.Menu{}
		_ = copier.Copy(&mn, item)
		mn.SubMenu = o.SubTreeMenu(id, mn.ID)

		ret = append(ret, mn)
	}

	return ret
}

func (o *RoleServiceImpl) TreeMenu(id string) []dtos.Menu {

	ret := []dtos.Menu{}

	list := rolemenu.GetRepo().ListMenuByRole(id, "ROOT")

	for _, item := range list {

		mn := dtos.Menu{}
		_ = copier.Copy(&mn, item)
		mn.SubMenu = o.SubTreeMenu(id, mn.ID)

		ret = append(ret, mn)
	}

	return ret
}

func (o *RoleServiceImpl) ListMenu(id, parentId string) []dtos.Menu {

	ret := []dtos.Menu{}
	list := rolemenu.GetRepo().ListMenuByRole(id, parentId)

	for _, item := range list {

		mn := dtos.Menu{}
		_ = copier.Copy(&mn, item)

		ret = append(ret, mn)
	}

	return ret
}

func (o *RoleServiceImpl) Search(filter, isClientRole string, pageNo, rowPerPage int) ([]dtos.Role, bool, int64) {

	var ret []dtos.Role
	ret = []dtos.Role{}
	list, haveNext, count := role.GetRepo().Search(filter, isClientRole, pageNo, rowPerPage)

	for _, mdl := range list {
		item := dtos.Role{}
		_ = copier.Copy(&item, mdl)
		ret = append(ret, item)
	}

	return ret, haveNext, count
}

func (o *RoleServiceImpl) SetPermission(id string, req dtos.RolePermissionReq, authInfo auth.AuthInfo) ([]dtos.Permission, *gocom.CodedError) {
	if authInfo.ClientId != common.PROVIDER_ID {
		logger.Infof("[RoleService SetPermission] unable to update role with client id %s \n", authInfo.ClientId)
		return nil, common.ERR_INVALID_REQUEST
	}

	ret := []dtos.Permission{}
	roleRepo := role.GetRepo().GetById(id)

	if roleRepo == nil {
		return ret, common.ERR_NOT_FOUND
	}

	_ = rolepermission.GetRepo().ClearByRole(id)
	_ = rolepermission.GetRepo().SetRolePermission(id, req.Permissions)

	list := rolepermission.GetRepo().ListPermissionByRole(id, "")

	for _, item := range list {
		mn := dtos.Permission{}
		_ = copier.Copy(&mn, item)
		ret = append(ret, mn)
	}

	return ret, nil
}

func (o *RoleServiceImpl) ListPermission(id, parentId string) []dtos.Permission {
	ret := []dtos.Permission{}
	list := rolepermission.GetRepo().ListPermissionByRole(id, parentId)

	for _, item := range list {
		mn := dtos.Permission{}
		_ = copier.Copy(&mn, item)
		ret = append(ret, mn)
	}

	return ret
}

func (o *RoleServiceImpl) Delete(id string, authInfo auth.AuthInfo) *gocom.CodedError {
	if authInfo.ClientId != common.PROVIDER_ID {
		logger.Infof("[RoleService Delete] unable to update role with client id %s \n", authInfo.ClientId)
		return common.ERR_INVALID_REQUEST
	}

	if slices.Contains(defaultRole, id) {
		return gocom.NewError(9999, "Unable to delete default role")
	}

	roleRepo := role.GetRepo().GetById(id)

	if roleRepo == nil {
		return common.ERR_NOT_FOUND
	}
	err := role.GetRepo().Delete(id)

	if err != nil {
		return common.ERR_UNABLE_TO_DELETE
	}
	_ = pubsub.Get().Publish("update_role", id)
	return nil
}

//-----------------------------------------

var roleService RoleService
var roleServiceOnce sync.Once

func SetRoleService(svc RoleService) {
	roleService = svc
}

func GetRoleService() RoleService {
	if roleService == nil {
		roleServiceOnce.Do(func() {
			tmp := &RoleServiceImpl{}
			roleService = tmp
		})
	}
	return roleService
}
