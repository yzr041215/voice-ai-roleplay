package domain

import "gorm.io/gorm"

type User struct {
	Name     string `json:"name"`
	Password string `json:"password"`
	gorm.Model
}

type CreateUserReq struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type LoginReq struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}
type LoginResp struct {
	Token string `json:"token"`
}
