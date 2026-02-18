package mysql

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB(dsn string) error {
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	return err
}
