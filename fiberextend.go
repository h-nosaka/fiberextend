package fiberextend

import (
	"context"
	"flag"
	"fmt"
	"net"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/getsentry/sentry-go"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/gofiber/swagger"
	"github.com/google/uuid"
	"github.com/imdario/mergo"
	"github.com/redis/go-redis/v9"
	"github.com/things-go/gormzap"
	"go.uber.org/zap"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

var Ex *IFiberEx
var Log *zap.Logger
var GLog *glogger.Interface
var DB *gorm.DB
var Redis *redis.Client
var ES *elasticsearch.Client
var Validator *validator.Validate
var Sentry *sentry.Client

var background = context.Background()

type IFiberEx struct {
	NodeId    string
	Config    IFiberExConfig
	App       *fiber.App
	Log       *zap.Logger
	DB        *gorm.DB
	Redis     *redis.Client
	ES        *elasticsearch.Client
	Sentry    *sentry.Client
	Validator *validator.Validate
}

type IFiberExConfig struct {
	// 実行モード
	DevMode  *bool
	TestMode *bool
	// 内部パラメータ
	SecretToken   string
	TokenExpireAt time.Duration
	// fiber初期化パラメータ
	IconFile         *string
	IconUrl          *string
	CorsOrigin       *string
	CorsHeaders      *string
	CaseSensitive    *bool
	Concurrency      *int
	DisableKeepalive *bool
	ErrorHandler     func(*fiber.Ctx, error) error
	AppName          *string
	BodyLimit        *int
	// サービスホスト
	Host string
	// ページング処理
	PagePer *int
	// データベース接続
	UseDB    bool
	DBConfig *IDBConfig
	// キャッシュサーバ接続
	UseRedis     bool
	RedisOptions *redis.Options
	// elasticsearch接続
	UseES    bool
	ESConfig *elasticsearch.Config
	// SMTP
	SmtpUseMd5 bool
	SmtpFrom   string
	SmtpAddr   string
	SmtpUser   *string
	SmtpPass   *string
	// Job
	JobAddr     string
	JobDatabase int
	JobPool     int
	JobProcess  int
	// Sentry
	SentryDsn   *string
	SentryScope *sentry.Scope
	SentryEnv   string
	// Option 任意のパラメータの格納
	Options *IFiberExConfigOption
}

type IDBConfig struct {
	Config *gorm.Config
	User   string
	Pass   string
	Addr   string
	DBName string
}

type IFiberExConfigOption struct {
	src map[string]interface{}
}

func NewIFiberExConfigOption(src map[string]interface{}) *IFiberExConfigOption {
	return &IFiberExConfigOption{
		src: src,
	}
}

func (p *IFiberExConfigOption) GetString(key string) string {
	switch p.src[key].(type) {
	case string:
		return p.src[key].(string)
	default:
		return ""
	}
}

func (p *IFiberExConfigOption) GetInt(key string) int {
	switch p.src[key].(type) {
	case int:
		return p.src[key].(int)
	default:
		return 0
	}
}

func (p *IFiberExConfigOption) GetInt64(key string) int64 {
	switch p.src[key].(type) {
	case int64:
		return p.src[key].(int64)
	default:
		return 0
	}
}

func String(src string) *string {
	return &src
}

func Int(src int) *int {
	return &src
}

func Bool(src bool) *bool {
	return &src
}

func (p *IFiberEx) DefaultErrorHandler() func(*fiber.Ctx, error) error {
	return func(c *fiber.Ctx, err error) error {
		return p.ResultError(c, 500, err, E99999.Errors()...)
	}
}

var defaultIFiberExConfig *IFiberExConfig = &IFiberExConfig{
	DevMode:          Bool(false),
	TestMode:         Bool(false),
	CorsOrigin:       String("*"),
	CorsHeaders:      String("GET,POST,HEAD,PUT,DELETE,PATCH"),
	CaseSensitive:    Bool(true),
	Concurrency:      Int(256 * 1024),
	DisableKeepalive: Bool(false),
	AppName:          String("App"),
	BodyLimit:        Int(4 * 1024 * 1024),
	PagePer:          Int(30),
}

var defaultRedisOptions *redis.Options = &redis.Options{
	Addr:         "redis:6379",
	Username:     "",
	Password:     "",
	DB:           0,
	MaxRetries:   5,
	PoolSize:     100,
	MinIdleConns: 10,
	MaxIdleConns: 100,
	TLSConfig:    nil,
}

var defaultDBConfig *IDBConfig = &IDBConfig{
	User:   "",
	Pass:   "",
	Addr:   "db:3306",
	DBName: "",
	Config: &gorm.Config{},
}

var defaultESConfig *elasticsearch.Config = &elasticsearch.Config{
	Addresses:     []string{"http://es:9200"},
	Username:      "",
	Password:      "",
	RetryOnStatus: []int{502, 503, 504},
	DisableRetry:  true,
	MaxRetries:    3,
}

func New(config IFiberExConfig) *IFiberEx {
	// 設定の初期化
	if err := mergo.Merge(&config, defaultIFiberExConfig); err != nil {
		panic(err)
	}

	// logger初期化
	if Log == nil {
		var con zap.Config
		if config.DevMode != nil && *config.DevMode {
			con = zap.NewDevelopmentConfig()
			if config.TestMode != nil && *config.TestMode {
				con.Level.SetLevel(zap.ErrorLevel) // テストモードではエラーしかログ出力しない
			}
		} else {
			con = zap.NewProductionConfig()
		}
		logger, err := con.Build()
		if err != nil {
			panic(err)
		}
		Log = logger
		gzap := gormzap.New(Log,
			gormzap.WithConfig(glogger.Config{
				SlowThreshold:             200 * time.Millisecond,
				Colorful:                  true,
				IgnoreRecordNotFoundError: true,
				LogLevel:                  glogger.Warn,
			}),
		)
		GLog = &gzap
	}

	// DB初期化
	if DB == nil && config.UseDB {
		if config.DBConfig == nil {
			config.DBConfig = &IDBConfig{}
		}
		if err := mergo.Merge(config.DBConfig, defaultDBConfig); err != nil {
			panic(err)
		}
		if GLog != nil {
			config.DBConfig.Config.Logger = *GLog
		}
		DB = config.NewDB()
	}

	// Redis初期化
	if Redis == nil && config.UseRedis {
		if config.RedisOptions == nil {
			config.RedisOptions = &redis.Options{}
		}
		if err := mergo.Merge(config.RedisOptions, defaultRedisOptions); err != nil {
			panic(err)
		}
		Redis = config.NewRedis()
	}

	// ES初期化
	if ES == nil && config.UseES {
		if config.ESConfig == nil {
			config.ESConfig = &elasticsearch.Config{}
		}
		if err := mergo.Merge(config.ESConfig, defaultESConfig); err != nil {
			panic(err)
		}
		ES = config.NewES()
	}

	// Validator初期化
	Validator = validator.New()
	if err := Validator.RegisterValidation("match", ValidateMatch); err != nil {
		panic(err)
	}

	// uuid
	obj, err := uuid.NewRandom()
	if err != nil {
		panic(err)
	}

	// sentry
	if config.SentryDsn != nil {
		Sentry, err = sentry.NewClient(sentry.ClientOptions{
			Dsn:         *config.SentryDsn,
			Environment: config.SentryEnv,
		})
		if err != nil {
			panic(err)
		}
		if config.SentryScope != nil {
			config.SentryScope = sentry.NewScope()
		}
	}

	Log.Info("fiberextend.New", zap.String("NodeId", obj.String()))

	Ex = &IFiberEx{
		NodeId:    obj.String(),
		Config:    config,
		Log:       Log,
		DB:        DB,
		Redis:     Redis,
		ES:        ES,
		Validator: Validator,
		Sentry:    Sentry,
	}
	return Ex
}

func RunCommand() (string, []string, bool) {
	flag.Parse()
	args := flag.Args()
	if len(args) > 1 {
		if args[0] == "run" {
			return args[1], args[2:], true
		}
	}
	return "", args, false
}

func (p *IFiberEx) NewApp() *fiber.App {
	errHandler := p.DefaultErrorHandler()
	if p.Config.ErrorHandler != nil {
		errHandler = p.Config.ErrorHandler
	}
	app := fiber.New(fiber.Config{
		CaseSensitive:    *p.Config.CaseSensitive,
		Concurrency:      *p.Config.Concurrency,
		DisableKeepalive: *p.Config.DisableKeepalive,
		ErrorHandler:     errHandler,
		AppName:          *p.Config.AppName,
		BodyLimit:        *p.Config.BodyLimit,
	})

	app.Use(recover.New())
	app.Use(p.MetaMiddleware())
	app.Use(cors.New(cors.Config{
		AllowOrigins: *p.Config.CorsOrigin,
		AllowHeaders: *p.Config.CorsHeaders,
	}))
	app.Use(requestid.New())
	app.Use(zapLogger(p.Log))
	if p.Config.IconFile != nil && p.Config.IconUrl != nil {
		app.Use(favicon.New(favicon.Config{
			File: *p.Config.IconFile,
			URL:  *p.Config.IconUrl,
		}))
	} else {
		app.Use(favicon.New())
	}

	if p.Config.DevMode != nil && *p.Config.DevMode {
		app.Static("docs/", "./docs")
		app.Get("/swagger/*", swagger.New(swagger.Config{
			URL:          fmt.Sprintf("http://%s/docs/swagger.json", p.Config.Host),
			DeepLinking:  false,
			DocExpansion: "none",
		}))
	}

	p.App = app

	return app
}

func (p *IFiberEx) IpAddr() string {
	var ip string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ip
	}
	for _, addr := range addrs {
		nip, ok := addr.(*net.IPNet)
		if ok && !nip.IP.IsLoopback() && nip.IP.To4() != nil {
			ip = nip.IP.String()
		}
	}
	return ip
}
