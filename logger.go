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
			logger.Error(chainErr.Error(), zap.String("requestid", c.Locals("requestid").(string)))
		}
		stop := time.Now().Local()

		fields := []zap.Field{
			zap.Int("pid", os.Getpid()),
			zap.String("elaps", stop.Sub(start).String()),
			zap.Any("ip", c.IPs()),
			zap.String("requestid", c.Locals("requestid").(string)),
			zap.String("userid", c.Locals("userid").(string)),
			zap.Int("status", c.Response().StatusCode()),
			zap.Any("query", c.Queries()),
			zap.String("body", string(c.Request().Body())),
			zap.String("response", string(c.Response().Body())),
		}
		logger.With(fields...).Info(fmt.Sprintf("Access: %s %s", c.Method(), c.Path()))

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

func (p *IFiberEx) LogTrace(err error, fields ...zap.Field) *zap.Logger {
	fields = append(fields, p.LogCaller())
	if err != nil {
		if e, ok := err.(*Errors); ok {
			fields = append(fields, zap.Any("stacktrace", e.Trace()))
		}
	}
	p.CatchPreparedStatementsError(err)
	if p.Sentry != nil {
		defer p.Sentry.Flush(2 * time.Second)
		p.Sentry.CaptureException(err, &sentry.EventHint{
			OriginalException: err,
		}, p.SentryScope(fields))
	}
	return p.Log.With(fields...)
}

func (p *IFiberEx) LogError(err error, fields ...zap.Field) {
	p.LogTrace(err, fields...).Error(err.Error())
}

func (p *IFiberEx) LogFatal(err error, fields ...zap.Field) {
	p.LogTrace(err, fields...).Fatal(err.Error())
}

func (p *IFiberEx) LogWarn(err error, fields ...zap.Field) {
	p.LogTrace(err, fields...).Warn(err.Error())
}

func (p *IFiberEx) LogInfo(msg string, fields ...zap.Field) {
	p.Log.With(p.LogCaller()).Info(msg, fields...)
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

func (p *IFiberEx) ApiLogFields(c *fiber.Ctx, fields ...zapcore.Field) []zapcore.Field {
	fields = append(fields,
		zap.String("requestid", c.Locals("requestid").(string)),
		zap.String("userid", c.Locals("userid").(string)),
	)
	return fields
}

func (p *IFiberEx) ApiErrorLogFields(c *fiber.Ctx, err error, fields ...zapcore.Field) []zapcore.Field {
	if err != nil {
		if _, ok := err.(*Errors); !ok {
			err = NewErrors(err)
		}
		fields = append(fields, zap.Any("stacktrace", err.(*Errors).Trace()))
	}
	fields = p.ApiLogFields(c, fields...)
	return fields
}
