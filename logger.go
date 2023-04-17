package fiberextend

import (
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
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

func (p *IFiberEx) LogError(err error, fields ...zap.Field) {
	p.Log.Error(err.Error(), fields...)
	if p.Sentry != nil {
		p.Sentry.CaptureException(err, &sentry.EventHint{
			Data:              fields,
			OriginalException: err,
		}, p.Config.SentryScope)
	}
}

func (p *IFiberEx) LogFatal(err error, fields ...zap.Field) {
	p.Log.Fatal(err.Error(), fields...)
	if p.Sentry != nil {
		p.Sentry.CaptureException(err, &sentry.EventHint{
			Data:              fields,
			OriginalException: err,
		}, p.Config.SentryScope)
	}
}

func (p *IFiberEx) LogWarn(err error, fields ...zap.Field) {
	p.Log.Warn(err.Error(), fields...)
	if p.Sentry != nil {
		p.Sentry.CaptureException(err, &sentry.EventHint{
			Data:              fields,
			OriginalException: err,
		}, p.Config.SentryScope)
	}
}
