package fiberextend_test

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gofiber/fiber/v2"
	ext "github.com/h-nosaka/fiberextend"
	"github.com/redis/go-redis/v9"
)

func TestIt(t *testing.T) {
	test := ext.NewTest(t, ext.IFiberExConfig{})
	test.It("test1")
	test.Run("test2", func() {
		test.It("test3")
	})
	t.Error("test")
}

func TestApi(t *testing.T) {
	test := ext.NewTest(t, ext.IFiberExConfig{
		DevMode: ext.Bool(true),
		UseDB:   true,
		DBConfig: &ext.IDBConfig{
			Addr:   "db:3306",
			User:   "root",
			Pass:   "qwerty",
			DBName: "app",
		},
		UseRedis:     true,
		RedisOptions: &redis.Options{},
		UseES:        true,
		ESConfig: &elasticsearch.Config{
			Addresses: []string{"http://es:9200"},
		},
	})
	test.Routes(func(ex *ext.IFiberEx) {
		ex.App.Get("/", func(c *fiber.Ctx) error {
			var rs time.Time
			ex.DB.Raw("SELECT NOW()").Scan(&rs)
			return ex.Result(c, 200, map[string]interface{}{"status": "ok", "now": rs.Local().String()})
		})
	})
	test.Run("test1", func() {
		data := "data"
		if err := test.Redis.Set("test_key", data); err != nil {
			t.Error(err)
		}
		test.Api("api1", &ext.ITestRequest{Method: "GET", Path: "/"}, 200, []*ext.ITestCase{
			{
				Method: ext.TestMethodEqual,
				Path:   `$.result.status`,
				Want:   "ok",
			},
			{
				Method: ext.TestMethodNotEqual,
				Path:   `$.result.now`,
				Want:   nil,
			},
			{
				Method: ext.TestMethodEqual,
				Store: func() interface{} {
					rs, err := test.Ex.Redis.Get(context.TODO(), "test_key").Result()
					if err != nil {
						t.Error(err)
					}
					return rs
				},
				Want: data,
			},
			{
				Method: ext.TestMethodNotEqual,
				Store: func() interface{} {
					res, err := test.Ex.ES.Info()
					if err != nil {
						t.Error(err)
					}
					defer res.Body.Close()
					rs, err := io.ReadAll(res.Body)
					if err != nil {
						t.Error(err)
					}
					return string(rs)
				},
				Want: nil,
			},
		}...)
	})
}

func TestExec(t *testing.T) {
	test := ext.NewTest(t, ext.IFiberExConfig{
		DevMode: ext.Bool(true),
		UseDB:   false,
		DBConfig: &ext.IDBConfig{
			Addr:   "db:3306",
			User:   "root",
			Pass:   "qwerty",
			DBName: "app",
		},
		UseRedis:     true,
		RedisOptions: &redis.Options{},
		UseES:        false,
		ESConfig: &elasticsearch.Config{
			Addresses: []string{"http://es:9200"},
		},
	})
	test.Run("test1", func() {
		test.Exec("sub1", func() interface{} {
			fmt.Println(test.Ex.Config)
			return test.Ex.Config
		}, &ext.ITestCase{
			It:   "it1",
			Want: "db:3306",
			Path: "DBConfig.Addr",
		}, &ext.ITestCase{
			It:   "it2",
			Want: nil,
			Path: "DBConfig.Addr.Foo",
		})
	})
}
