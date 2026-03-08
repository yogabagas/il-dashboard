package dtos

type MenuReq struct {
	Name     string `json:"name"`
	ParentId string `json:"parentId"`
	Target   string `json:"target"`
	IconURL  string `json:"iconURL"`
	OrderNo  int    `json:"orderNo"`
}

type Menu struct {
	MenuReq
	ID      string `json:"id"`
	SubMenu []Menu `json:"subMenu"`
}

type MenuWithoutSubMenu struct {
	MenuReq
	ID string `json:"id"`
}
