package store

import (
	"demo/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type MySQL struct {
	*gorm.DB
}

func NewMySQL(config *config.Config) *MySQL {
	db, err := gorm.Open(mysql.Open(config.MySQL.Dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	return &MySQL{db}
}
