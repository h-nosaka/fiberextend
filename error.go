package fiberextend

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
