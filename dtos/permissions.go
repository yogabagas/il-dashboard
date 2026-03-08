package dtos

type AccessModuleReq struct {
	Name     string `json:"name"`
	ParentId string `json:"parentId"`
}

type AccessModule struct {
	AccessModuleReq
	ID         string         `json:"id"`
	SubModule  []AccessModule `json:"subModule"`
	Permission []Permission   `json:"permissions"`
}

type PermissionReq struct {
	ModuleId string `json:"moduleId"`
	Name     string `json:"name"`
	Code     string `json:"code"`
}

type Permission struct {
	PermissionReq
	ID         string `json:"id"`
	ModuleName string `json:"moduleName"`
}
