package fiberextend_test

import (
	"testing"

	"github.com/h-nosaka/fiberextend"
	ext "github.com/h-nosaka/fiberextend"
)

func TestSimpleValidation(t *testing.T) {
	ex := ext.New(ext.IFiberExConfig{})
	err := ex.SimpleValidation("abc", "test", "match=^[a-z]+$")
	if err != nil {
		t.Errorf("%+v", err)
	}
}

type StructTest struct {
	Name     string `json:"name" validate:"required,match=^[a-z]+$"`
	Email    string `json:"email" validate:"required,email"`
	Age      int    `json:"age" validate:"required,min=18,max=30"`
	Sex      int    `json:"sex" validate:"required"`
	Password string `json:"password" validate:"required,password=12"`
}

func TestValidation(t *testing.T) {
	ex := ext.New(ext.IFiberExConfig{})
	src := StructTest{
		Name:     "qwerty",
		Email:    "hoge@hoge.com",
		Age:      20,
		Sex:      1,
		Password: "Q1w2e3r4t5!!",
	}
	err := ex.Validation(&src)
	if err != nil {
		t.Errorf("%+v", err)
	}
}

func TestValidationError(t *testing.T) {
	ex := ext.New(ext.IFiberExConfig{})
	src := StructTest{
		Name:     "qwerty",
		Email:    "hoge@hoge.com",
		Age:      20,
		Sex:      1,
		Password: "Q1w2e3r4t5!",
	}
	err := ex.Validation(src)
	t.Log(fiberextend.ToPrettyJson(err))
	if err == nil {
		t.Errorf("%+v", err)
	}
}

type StructTest2 struct {
	Name string `json:"name,omitempty"`
}

func TestGetJsonTag(t *testing.T) {
	src := StructTest2{
		Name: "qwerty",
	}
	rs := ext.GetJsonTag(src, "Name")
	if rs != "name" {
		t.Errorf("result string: %s", rs)
	}
	t.Log(rs)
}
