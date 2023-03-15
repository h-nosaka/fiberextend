package fiberextend

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/jrallison/go-workers"
	"github.com/redis/go-redis/v9"
	"github.com/steinfletcher/apitest"
	jsonpath "github.com/steinfletcher/apitest-jsonpath"
	"golang.org/x/exp/slices"
)

type IFiberExTest struct {
	Ex     *IFiberEx
	App    *fiber.App
	t      *testing.T
	Redis  *miniredis.Miniredis
	Tester *apitest.APITest
}

type ITestMethod int

const (
	TestMethodEqual ITestMethod = iota
	TestMethodNotEqual
	TestMethodContains
	TestMethodPresent
	TestMethodNotPresent
	TestMethodMatches
	TestMethodLen
	TestMethodGreaterThan
	TestMethodLessThan
)

type ITestCase struct {
	It     string             // テストケースの説明
	Method ITestMethod        // assertしたい内容
	Want   interface{}        // 期待値
	Path   string             // jsonpath `$.id`
	Store  func() interface{} // データの取得 pathが指定されていない場合に使用
}

type ITestRequest struct {
	Method  string
	Path    string
	Headers map[string]string
	Body    interface{}
}

func NewTest(t *testing.T, config IFiberExConfig) *IFiberExTest {
	config.TestMode = Bool(true)
	// redisをminiredisに置き換え
	var r *miniredis.Miniredis
	if config.UseRedis {
		r = miniredis.RunT(t)
		if config.RedisOptions == nil {
			config.RedisOptions = &redis.Options{}
		}
		config.RedisOptions.Addr = r.Addr()
		config.JobAddr = r.Addr()

	}
	ex := New(config)
	app := ex.NewApp()
	test := &IFiberExTest{
		Ex:    ex,
		App:   app,
		t:     t,
		Redis: r,
	}
	// apitestを初期化
	test.Tester = apitest.New().HandlerFunc(test.fiberToHandlerFunc())
	return test
}

func (p *IFiberExTest) Routes(routes func(*IFiberEx)) {
	routes(p.Ex)
}

func (p *IFiberExTest) DryJobs(names ...string) {
	jobs := []*IJob{}
	for _, name := range names {
		jobs = append(jobs, &IJob{
			Name: name,
			Proc: func(msg *workers.Msg) {
				p.t.Logf("start job: %s, %+v", name, msg)
			},
			Concurrency: 10,
		})
	}
	p.Ex.NewJob(jobs...)
	p.Ex.JobRun()
}

func (p *IFiberExTest) Run(it string, tests func()) {
	p.It(it)
	if p.Ex.Config.UseDB {
		p.Ex.DB = p.Ex.DB.Begin() // トランザクション開始
		p.Ex.DB.SavePoint(it)
	}
	// テスト実行
	tests()
	// ロールバック
	if p.Ex.Config.UseDB {
		p.Ex.DB = p.Ex.DB.RollbackTo(it) // dbをロールバックする
	}
	if p.Ex.Config.UseRedis {
		p.Redis.FlushAll() // miniredisの中身をクリアする
	}
	if p.Ex.Config.UseES {
		_, err := p.Ex.ES.Indices.Delete([]string{"*"}) // すべてのindexを削除する
		if err != nil {
			p.t.Error(err)
		}
	}
}

func (p *IFiberExTest) It(message string) {
	i := 1
	_, file, line, ok := runtime.Caller(i)
	for ok && filepath.Base(file) == "tests.go" {
		i++
		_, file, line, _ = runtime.Caller(i)
	}
	p.t.Logf("%s:%d: %s", file, line, message)
}

func (p *IFiberExTest) Api(message string, request *ITestRequest, status int, asserts ...*ITestCase) {
	p.It(message)
	api := request.Call(p.Tester).Expect(p.t).Status(status)
	for _, assert := range asserts {
		p.It(assert.It)
		api = api.Assert(assert.ApiAssert())
	}
	api.End()
}

func (p *IFiberExTest) fiberToHandlerFunc() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := p.App.Test(r)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		// copy headers
		for k, vv := range resp.Header {
			for _, v := range vv {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)

		// copy body
		if _, err := io.Copy(w, resp.Body); err != nil {
			panic(err)
		}
	}
}

func (p ITestRequest) Call(test *apitest.APITest) *apitest.Request {
	var app *apitest.Request
	switch p.Method {
	case "POST":
		app = test.Post(p.Path)
	case "PATCH":
		app = test.Patch(p.Path)
	case "PUT":
		app = test.Put(p.Path)
	case "DELETE":
		app = test.Delete(p.Path)
	default: // "GET"
		app = test.Get(p.Path)
	}
	app = app.Header("Content-Type", "application/json")
	for key, value := range p.Headers {
		app = app.Header(key, value)
	}
	if p.Body != nil {
		app = app.Body(p.ToString())
	}
	return app
}

func (p *ITestRequest) ToString() string {
	switch p.Body.(type) {
	case string:
		return p.Body.(string)
	default:
		if rs, err := json.Marshal(p.Body); err == nil {
			return string(rs)
		}
	}
	return ""
}

func (p *ITestCase) Error(message string) error {
	i := 1
	_, file, line, ok := runtime.Caller(i)
	files := []string{"tests.go", "apitest.go"}
	for ok && slices.Contains(files, filepath.Base(file)) {
		i++
		_, file, line, _ = runtime.Caller(i)
	}
	return fmt.Errorf("%s:%d: %s", file, line, message)
}

func (p *ITestCase) Assert() error {
	value := p.Store()
	switch p.Method {
	case TestMethodEqual:
		if value != p.Want {
			return p.Error(fmt.Sprintf("assert equal: value: %+v, want: %+v", value, p.Want))
		}
	case TestMethodNotEqual:
		if value == p.Want {
			return p.Error(fmt.Sprintf("assert not equal: value: %+v, want: %+v", value, p.Want))
		}
	case TestMethodContains:
		if strings.Contains(value.(string), p.Want.(string)) {
			return p.Error(fmt.Sprintf("assert contains: value: %+v, want: %+v", value, p.Want))
		}
	case TestMethodMatches:
		r, err := regexp.Compile(p.Want.(string))
		if err != nil {
			return p.Error(fmt.Sprintf("assert match: %s", err))
		}
		if !r.Match([]byte(value.(string))) {
			return p.Error(fmt.Sprintf("assert match: value: %+v, want: %+v", value, p.Want))
		}
	case TestMethodLen:
		if value.(int) != p.Want.(int) {
			return p.Error(fmt.Sprintf("assert len: value: %+v, want: %+v", value, p.Want))
		}
	case TestMethodGreaterThan:
		if value.(int) < p.Want.(int) {
			return p.Error(fmt.Sprintf("assert greater than: value: %+v, want: %+v", value, p.Want))
		}
	case TestMethodLessThan:
		if value.(int) > p.Want.(int) {
			return p.Error(fmt.Sprintf("assert less than: value: %+v, want: %+v", value, p.Want))
		}
	default:
		return p.Error("error: not support TestMethod")
	}
	return nil
}

func (p *ITestCase) ApiAssert() func(*http.Response, *http.Request) error {
	if len(p.Path) > 0 {
		switch p.Method {
		case TestMethodEqual:
			return jsonpath.Equal(p.Path, p.Want)
		case TestMethodNotEqual:
			return jsonpath.NotEqual(p.Path, p.Want)
		case TestMethodContains:
			return jsonpath.Contains(p.Path, p.Want)
		case TestMethodPresent:
			return jsonpath.Present(p.Path)
		case TestMethodNotPresent:
			return jsonpath.NotPresent(p.Path)
		case TestMethodMatches:
			return jsonpath.Matches(p.Path, p.Want.(string))
		case TestMethodLen:
			return jsonpath.Len(p.Path, p.Want.(int))
		case TestMethodGreaterThan:
			return jsonpath.GreaterThan(p.Path, p.Want.(int))
		case TestMethodLessThan:
			return jsonpath.LessThan(p.Path, p.Want.(int))
		}
	}
	return func(res *http.Response, req *http.Request) error {
		return p.Assert()
	}
}

func (p *IFiberExTest) Job(it string, before func(), job func(), asserts ...*ITestCase) {
	p.It(it)
	before()
	job()
	for _, assert := range asserts {
		p.It(assert.It)
		if err := assert.Assert(); err != nil {
			p.t.Error(err)
		}
	}
}
