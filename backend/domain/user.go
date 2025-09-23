package domain

import "gorm.io/gorm"

type User struct {
	Name     string `json:"name"`
	Password string `json:"password"`
	gorm.Model
}
