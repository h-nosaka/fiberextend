package fiberextend

import (
	"fmt"
	"runtime"
	"strings"
)

type ErrorCode int

const (
	E00500 ErrorCode = iota
	E40001
	E99999
)

func (p ErrorCode) Errors() []IError {
	switch p {
	case E40001:
		return []IError{{Code: "E40001", Message: "Validation Error"}}
	case E99999:
		return []IError{{Code: "E99999", Message: "Undefined Error"}}
	}
	return []IError{{Code: "E99999", Message: "Undefined Error"}}
}

type Errors struct {
	msg   error
	trace []string
}

var LocalFilePath = "/opt/app"

func InitErrors(path string) {
	LocalFilePath = path
}

func NewErrors(err error) error {
	return &Errors{msg: err, trace: StackTrace()}
}

func (p *Errors) String() string {
	return p.msg.Error()
}

func (p *Errors) Error() string {
	return p.String()
}

func (p *Errors) Trace() []string {
	return p.trace
}

func StackTrace() []string {
	return trace(2, 16)
}

func trace(count int, max int) []string {
	src := []string{}
	var (
		file string
		line int
	)
	ok := true
	for ok {
		_, file, line, ok = runtime.Caller(count)
		if strings.Contains(file, LocalFilePath) {
			src = append(src, fmt.Sprintf("%s:%d", file, line))
		}
		count++
		if count > max {
			ok = false
		}
	}
	return src
}
