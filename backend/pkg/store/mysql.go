package store

import (
	"demo/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type MySQL struct {
	*gorm.DB
}

func NewMySQL(config *config.Config) (*MySQL, error) {
	db, err := gorm.Open(mysql.Open(config.MySQL.Dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return &MySQL{db}, nil
}
