package fiberextend

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func (p *IFiberExConfig) NewDB() *gorm.DB {
	if p.TestMode != nil && *p.TestMode {
		p.DBConfig.DBName += "_test"
	}
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", // mysql dsn
		p.DBConfig.User,
		p.DBConfig.Pass,
		p.DBConfig.Addr,
		p.DBConfig.DBName,
	)
	db, err := gorm.Open(mysql.Open(dsn), p.DBConfig.Config)
	if err != nil {
		panic(err)
	}
	return db
}
