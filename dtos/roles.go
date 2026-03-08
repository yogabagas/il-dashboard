package dtos

type RoleReq struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

type Role struct {
	Name string `json:"name"`
	Code string `json:"code"`
	ID   string `json:"id"`
}

type RoleMenuReq struct {
	Menus []string `json:"menus"`
}

type RolePermissionReq struct {
	Permissions []string `json:"permissions"`
}
