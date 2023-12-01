package fiberextend

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type IMeta struct {
	Total   int64  `json:"total,omitempty"`   // トータル件数
	Page    int    `json:"page,omitempty"`    // ページ数
	Current int    `json:"current,omitempty"` // 現在のページ
	Elapsed string `json:"elapsed,omitempty"` // 所要時間
}

type IError struct {
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Param   string `json:"param,omitempty"`
	Message string `json:"message"`
}

type IResponse struct {
	Meta    *IMeta        `json:"meta,omitempty"`
	Result  interface{}   `json:"result,omitempty"`
	Results []interface{} `json:"results,omitempty"`
	Errors  []IError      `json:"error,omitempty"`
}

type IRequestPaging struct {
	Page *int `json:"page,omitempty"` // 表示ページ(1~)
	Per  *int `json:"per,omitempty"`  // 表示数
}

func (p *IFiberEx) MetaMiddleware() func(*fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		c.Locals("start_time", time.Now().Local())
		c.Locals("total_count", int64(0))
		c.Locals("page_max", 0)
		c.Locals("page_current", 0)
		c.Locals("userid", "-")
		return c.Next()
	}
}

func (p *IFiberEx) NewMeta(c *fiber.Ctx) *IMeta {
	stop := time.Now().Local()
	return &IMeta{
		Total:   c.Locals("total_count").(int64),
		Page:    c.Locals("page_max").(int),
		Current: c.Locals("page_current").(int),
		Elapsed: stop.Sub(c.Locals("start_time").(time.Time)).String(),
	}
}

func (p *IFiberEx) result(c *fiber.Ctx, code int, body *IResponse) error {
	body.Meta = p.NewMeta(c)
	rs, err := json.Marshal(body)
	if err != nil {
		return c.SendStatus(500)
	}
	c.Response().Header.Add("Content-Type", "application/json; charset=utf-8")
	return c.Status(code).SendString(string(rs))
}

func (p *IFiberEx) ResultError(c *fiber.Ctx, code int, err error, errors ...IError) error {
	if code == 500 {
		p.LogError(err, p.ApiLogFields(c)...)
	} else {
		p.Log.With(p.LogCaller()).Error(fmt.Sprintf("api error: %s", err), p.ApiLogFields(c)...)
	}
	return p.result(c, code, &IResponse{
		Errors: errors,
	})
}

func (p *IFiberEx) Result(c *fiber.Ctx, code int, results ...interface{}) error {
	cnt := len(results)
	if cnt == 0 {
		return p.ResultError(c, 204, fmt.Errorf("no content: %+v", results))
	} else if cnt > 1 {
		return p.result(c, code, &IResponse{Results: results})
	}
	return p.result(c, code, &IResponse{Result: results[0]})
}

// Deprecated: should not be used
func (p *IFiberEx) RequestParser(c *fiber.Ctx, params interface{}) bool {
	if c.Method() == "GET" {
		if err := c.QueryParser(params); err != nil {
			if err := p.ResultError(c, 400, err); err == nil {
				return false
			}
		}
	} else {
		if err := c.BodyParser(params); err != nil {
			if err := p.ResultError(c, 400, err); err == nil {
				return false
			}
		}
	}
	if err := p.Validation(params); len(err) > 0 {
		if err := p.ResultError(c, 400, fmt.Errorf("validation error: %+v", err), err...); err == nil {
			return false
		}
	}
	return true
}

func RequestParser[T comparable](ex *IFiberEx, c *fiber.Ctx, params *T) bool {
	if c.Method() == "GET" {
		if err := c.QueryParser(params); err != nil {
			if err := ex.ResultError(c, 400, err); err == nil {
				return false
			}
		}
	} else {
		if err := c.BodyParser(params); err != nil {
			if err := ex.ResultError(c, 400, err); err == nil {
				return false
			}
		}
	}
	if err := ex.Validation(*params); len(err) > 0 {
		if err := ex.ResultError(c, 400, fmt.Errorf("validation error: %+v", err), err...); err == nil {
			return false
		}
	}
	return true
}

func ValidateMatch(fl validator.FieldLevel) bool {
	r := regexp.MustCompile(fl.Param())
	return r.MatchString(fl.Field().String())
}

func (p *IFiberEx) Validation(src interface{}) []IError {
	err := p.Validator.Struct(src)
	if err != nil {
		return p.ValidationParser(src, err.(validator.ValidationErrors))
	}
	return nil
}

func (p *IFiberEx) SimpleValidation(src interface{}, field string, tag string) []IError {
	err := p.Validator.Var(src, tag)
	if err != nil {
		errors := p.ValidationParser(src, err.(validator.ValidationErrors))
		cnt := len(errors)
		for i := 0; i < cnt; i++ {
			errors[i].Field = field
		}
		return errors
	}
	return nil
}

func (p *IFiberEx) ValidationParser(src interface{}, errors validator.ValidationErrors) []IError {
	rs := []IError{}
	for _, err := range errors {
		rs = append(rs, IError{
			Code:    "E40001",
			Field:   GetJsonTag(src, err.Field()),
			Param:   err.Param(),
			Message: fmt.Sprintf("ValidationError.%s", err.Tag()), // TODO: 多言語対応が必要
		})
	}
	return rs
}

func GetJsonTag[T comparable](src T, field string) string {
	ref := reflect.TypeOf(src)
	rs := field
	if f, ok := ref.FieldByName(field); ok {
		rs = f.Tag.Get("json")
	}
	if strings.Contains(rs, ",") {
		item := strings.Split(rs, ",")
		rs = item[0]
	}
	return rs
}
