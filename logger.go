package fiberextend

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func zapLogger(logger *zap.Logger) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		start := time.Now().Local()
		chainErr := c.Next()
		if chainErr != nil {
			logger.Error(chainErr.Error())
		}
		stop := time.Now().Local()

		fields := []zap.Field{
			zap.Int("pid", os.Getpid()),
			zap.String("elaps", stop.Sub(start).String()),
			zap.Any("ip", c.IPs()),
			zap.String("requestid", c.Locals("requestid").(string)),
			zap.String("userid", c.Locals("userid").(string)),
			zap.Int("status", c.Response().StatusCode()),
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.String("body", string(c.Request().Body())),
		}

		if chainErr != nil {
			formatErr := chainErr.Error()
			fields = append(fields, zap.String("error", formatErr))
			logger.With(fields...).Error(formatErr)
		}
		logger.With(fields...).Info("api.request")

		return nil
	}
}

func (p *IFiberEx) SentryScope(fields []zap.Field) *sentry.Scope {
	scope := sentry.NewScope()
	enc := zapcore.NewJSONEncoder(zapcore.EncoderConfig{})
	buf, err := enc.EncodeEntry(zapcore.Entry{}, fields)
	if err == nil {
		extra := map[string]interface{}{}
		if e := json.Unmarshal(buf.Bytes(), &extra); e == nil {
			scope.SetExtras(extra)
		}
		// scope.SetExtra("fields", buf.String()) // 全体のdumpを取りたい場合は使う
	}
	return scope
}

func (p *IFiberEx) LogError(err error, fields ...zap.Field) {
	p.Log.With(p.LogCaller()).Error(err.Error(), fields...)
	if p.Sentry != nil {
		defer p.Sentry.Flush(2 * time.Second)
		p.Sentry.CaptureException(err, &sentry.EventHint{
			OriginalException: err,
		}, p.SentryScope(fields))
	}
}

func (p *IFiberEx) LogFatal(err error, fields ...zap.Field) {
	p.Log.With(p.LogCaller()).Fatal(err.Error(), fields...)
	if p.Sentry != nil {
		defer p.Sentry.Flush(2 * time.Second)
		p.Sentry.CaptureException(err, &sentry.EventHint{
			OriginalException: err,
		}, p.SentryScope(fields))
	}
}

func (p *IFiberEx) LogWarn(err error, fields ...zap.Field) {
	p.Log.With(p.LogCaller()).Warn(err.Error(), fields...)
	if p.Sentry != nil {
		defer p.Sentry.Flush(2 * time.Second)
		p.Sentry.CaptureException(err, &sentry.EventHint{
			OriginalException: err,
		}, p.SentryScope(fields))
	}
}

func (p *IFiberEx) Println(args ...interface{}) {
	p.Log.Info(fmt.Sprintln(args...), p.LogCaller())
}

func (p *IFiberEx) Printf(base string, args ...interface{}) {
	p.Log.Info(fmt.Sprintf(base, args...), p.LogCaller())
}

func (p *IFiberEx) LogCaller() zapcore.Field {
	i := 1
	_, file, line, ok := runtime.Caller(i)
	for ok && filepath.Base(file) == "logger.go" {
		i++
		_, file, line, _ = runtime.Caller(i)
	}
	return zap.String("caller", fmt.Sprintf("%s:%d", file, line))
}
