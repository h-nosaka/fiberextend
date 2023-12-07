package fiberextend

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/ettle/strcase"
	"github.com/gertd/go-pluralize"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// json化時に文字列をフィルタリングする
type FilterString string

func FilterStrings(src string) FilterString {
	return FilterString(src)
}

func (p FilterString) String() string {
	return string(p)
}

func (p FilterString) MarshalJSON() ([]byte, error) {
	return json.Marshal("****")
}

func (p *FilterString) UnmarshalJSON(data []byte) error {
	value := ""
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*p = FilterString(value)
	return nil
}

// UUIDをbase64に変換
func UuidBase64Encoding(src string) (string, error) {
	id, err := uuid.Parse(src)
	if err != nil {
		return src, err
	}
	buf, err := id.MarshalBinary()
	if err != nil {
		return src, err
	}
	return base64.URLEncoding.EncodeToString(buf), nil
}

// base64にエンコードされたUUIDをUUID形式に変換
func UuidBase64Decoding(src string) (string, error) {
	buf, err := base64.URLEncoding.DecodeString(src)
	if err != nil {
		return src, err
	}
	id, err := uuid.ParseBytes(buf)
	if err != nil {
		return src, err
	}
	return id.String(), nil
}

// bool型をint型に変換
func BoolToUint(ok bool) uint {
	if ok {
		return 1
	}
	return 0
}

// map型をarray型に変換
func MapToValueArray(src map[string]interface{}) []interface{} {
	rs := []interface{}{}
	for _, value := range src {
		rs = append(rs, value)
	}
	return rs
}

// 型をjson文字列に変換
func ToJson(src interface{}) string {
	rs, err := json.Marshal(src)
	if err != nil {
		return "{}"
	}
	return string(rs)
}

// 型を整形したjson文字列に変換
func ToPrettyJson(src interface{}) string {
	rs, err := json.MarshalIndent(src, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(rs)
}

// 現在時刻をポインタで取得
func Now() *time.Time {
	now := time.Now().Local()
	return &now
}

// スネークケースに変換
func ConvertSnakeCase(src string) string {
	con := pluralize.NewClient()
	return strcase.ToSnake(con.Singular(src))
}

// パスカルケースに変換
func ConvertPascalCase(src string) string {
	con := pluralize.NewClient()
	return strcase.ToCamel(con.Singular(src))
}

// キャメルケースに変換
func ConvertCamelCase(src string) string {
	con := pluralize.NewClient()
	return strcase.ToCamel(con.Singular(src))
}

// jsonを使って異なる型に変換
func ConvertStruct(src interface{}, out interface{}) error {
	rs, err := json.Marshal(src)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(rs, out); err != nil {
		return err
	}
	return nil
}

// jsonを使ってデータを複製
func DeepCopy(src interface{}, out interface{}) error {
	buf, err := json.Marshal(src)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(buf, out); err != nil {
		return err
	}
	return nil
}

// stringをintに変換
func Atoi(src string) int {
	rs, err := strconv.Atoi(src)
	if err != nil {
		return 0
	}
	return rs
}

// stringをint64に変換
func Atoi64(src string) int64 {
	rs, err := strconv.ParseInt(src, 10, 0)
	if err != nil {
		return 0
	}
	return rs
}

// int64をstringに変換
func Itoa(src int64) string {
	return strconv.Itoa(int(src))
}

// 構造体から指定のパスの値を取得
func StructPath(src interface{}, path string) (result interface{}) {
	defer func() {
		if rec := recover(); rec != nil {
			result = nil
		}
	}()
	ref := reflect.ValueOf(src)
	key := path
	if strings.Contains(path, ".") {
		item := strings.Split(path, ".")
		key = item[0]
		value := ref.FieldByName(key)
		// if value.IsNil() {
		// 	return nil
		// }
		if value.Kind() == reflect.Ptr {
			return StructPath(value.Elem().Interface(), strings.Join(item[1:], "."))
		}
		return StructPath(value.Interface(), strings.Join(item[1:], "."))
	}
	return ref.FieldByName(key).Interface()
}

// リカバー時にログを出力する
func Recover(ex *IFiberEx) {
	if rec := recover(); rec != nil {
		ex.LogFatal(errors.New("Recovered"), zap.Any("recover", rec))
		panic(rec) // ログ取得したらfiberのエラーハンドリングに任せる
	}
}
