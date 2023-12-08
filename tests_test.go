package fiberextend_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/h-nosaka/fiberextend"
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
			return ex.Result(c, 200, map[string]interface{}{
				"status": "ok",
				"now":    rs.Local().String(),
				"list":   []string{"foo", "bar"},
			})
		})
	})
	test.Run("test1", func() {
		data := "data"
		if err := test.Redis.Set("test_key", data); err != nil {
			t.Error(err)
		}
		test.Api("api1", &ext.ITestRequest{Method: "GET", Path: "/"}, 200, []*ext.ITestCase{
			{Path: `result.status`, Want: "ok"},
			{Method: ext.TestMethodNotEqual, Path: `result.now`, Want: nil},
			{Path: `result.list.0`, Want: "foo"},
			{Path: `result.list.1`, Want: "bar"},
			{
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

type ArrayStruct struct {
	Data  ArrayStructData
	List  []string
	PData *ArrayStructData
	PList *[]*string
}

type ArrayStructData struct {
	List  []ArrayStructList
	PList *[]*ArrayStructList
}

type ArrayStructList struct {
	Name  string
	PName *string
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
	array := ArrayStruct{
		Data: ArrayStructData{
			List: []ArrayStructList{
				{Name: "foo"},
				{Name: "bar"},
			},
		},
		List: []string{
			"foo",
			"bar",
		},
		PData: &ArrayStructData{
			PList: &[]*ArrayStructList{
				{PName: fiberextend.String("foo")},
				{PName: fiberextend.String("bar")},
			},
		},
		PList: &[]*string{
			fiberextend.String("foo"),
			fiberextend.String("bar"),
		},
	}
	test.Run("test1", func() {
		test.Exec("sub1", func() interface{} {
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
		test.Exec("sub2", func() interface{} {
			return array
		}, &ext.ITestCase{
			It:   "it1",
			Want: "foo",
			Path: "Data.List.0.Name",
		}, &ext.ITestCase{
			It:   "it2",
			Want: "bar",
			Path: "Data.List.1.Name",
		}, &ext.ITestCase{
			It:   "it3",
			Want: "foo",
			Path: "List.0",
		}, &ext.ITestCase{
			It:   "it4",
			Want: "bar",
			Path: "List.1",
		}, &ext.ITestCase{
			It:   "it5",
			Want: "foo",
			Path: "PData.PList.0.PName",
		}, &ext.ITestCase{
			It:   "it6",
			Want: "bar",
			Path: "PData.PList.1.PName",
		}, &ext.ITestCase{
			It:   "it7",
			Want: "foo",
			Path: "PList.0",
		}, &ext.ITestCase{
			It:   "it8",
			Want: "bar",
			Path: "PList.1",
		})
	})
}
