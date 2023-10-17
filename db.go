package fiberextend

import (
	"fmt"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func (p *IFiberExConfig) NewDB() *gorm.DB {
	if p.TestMode != nil && *p.TestMode {
		p.DBConfig.DBName += "_test"
	}
	var db *gorm.DB
	var err error
	if p.DBConfig.IsPostgres != nil && *p.DBConfig.IsPostgres {
		sslmode := "enable"
		if p.DevMode != nil && *p.DevMode {
			sslmode = "disable"
		}
		host := strings.Split(p.DBConfig.Addr, ":")
		dsn := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=%s", p.DBConfig.User, p.DBConfig.Pass, p.DBConfig.DBName, host[0], host[1], sslmode)
		db, err = gorm.Open(postgres.Open(dsn), p.DBConfig.Config)
		if err != nil {
			dsn := fmt.Sprintf("user=%s password=%s host=%s port=%s sslmode=%s", p.DBConfig.User, p.DBConfig.Pass, host[0], host[1], sslmode)
			db, err = gorm.Open(postgres.Open(dsn), p.DBConfig.Config)
		}
	} else {
		dsn := fmt.Sprintf(
			"%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", // mysql dsn
			p.DBConfig.User,
			p.DBConfig.Pass,
			p.DBConfig.Addr,
			p.DBConfig.DBName,
		)
		db, err = gorm.Open(mysql.Open(dsn), p.DBConfig.Config)
	}
	if err != nil {
		panic(err)
	}
	return db
}
