package database

import (
	_ "gorm.io/driver/postgres"
	"gorm.io/gorm"
	_ "gorm.io/gorm"
)

func NewPostgre() *gorm.DB{

    return nil
}  

