package gotoon

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// Encode 将任意数据结构编码为TOON格式
func Encode(data interface{}) (string, error) {
	if data == nil {
		return "", nil
	}
	tc := &TOONConverter{}

	return tc.ConvertStructToTOON(data)
}

// EncodeJSON 直接从JSON字符串编码为TOON格式
func EncodeJSON(jsonStr string) (string, error) {
	var data interface{}
	err := json.Unmarshal([]byte(jsonStr), &data)
	if err != nil {
		return "", err
	}
	tc := &TOONConverter{}

	return tc.ConvertStructToTOON(data)
}

// TOONConverter TOON格式转换器
type TOONConverter struct{}

// ConvertStructToTOON 转换结构体到TOON格式
func (tc *TOONConverter) ConvertStructToTOON(data interface{}) (string, error) {
	return tc.convertToTOON(data), nil
}

// convertToTOON 转换数据到TOON格式
func (tc *TOONConverter) convertToTOON(data interface{}) string {
	v := reflect.ValueOf(data)

	// 处理指针
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Slice:
		return tc.convertSliceToTOON(v.Interface())
	case reflect.Struct:
		return tc.convertStructToTOON(v.Interface())
	default:
		return fmt.Sprintf("%v", data)
	}
}

// convertSliceToTOON 转换切片到TOON格式
func (tc *TOONConverter) convertSliceToTOON(slice interface{}) string {
	v := reflect.ValueOf(slice)
	if v.Len() == 0 {
		return "[]"
	}

	// 获取元素类型
	elemType := v.Type().Elem()
	for elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}

	// 获取结构体字段名
	var fieldNames []string
	for i := 0; i < elemType.NumField(); i++ {
		field := elemType.Field(i)
		if jsonTag := field.Tag.Get("json"); jsonTag != "" {
			name := strings.Split(jsonTag, ",")[0]
			if name != "-" {
				fieldNames = append(fieldNames, name)
			}
		}
	}
	sort.Strings(fieldNames)

	// 构建TOON头部
	typeName := elemType.Name()
	if strings.HasSuffix(typeName, "Info") {
		typeName = strings.TrimSuffix(typeName, "Info")
	}
	header := fmt.Sprintf("%s[%d]{%s}:", strings.ToLower(typeName), v.Len(), strings.Join(fieldNames, ","))

	// 构建数据行
	var rows []string
	for i := 0; i < v.Len(); i++ {
		elem := v.Index(i)
		if elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}

		var values []string
		for _, fieldName := range fieldNames {
			// 找到对应的字段
			for j := 0; j < elem.NumField(); j++ {
				field := elem.Type().Field(j)
				if jsonTag := field.Tag.Get("json"); jsonTag != "" {
					name := strings.Split(jsonTag, ",")[0]
					if name == fieldName {
						fieldValue := elem.Field(j)
						values = append(values, tc.formatValue(fieldValue.Interface()))
						break
					}
				}
			}
		}
		rows = append(rows, strings.Join(values, ","))
	}

	return header + "\n  " + strings.Join(rows, "\n  ")
}

// convertStructToTOON 转换单个结构体到TOON格式
func (tc *TOONConverter) convertStructToTOON(strct interface{}) string {
	v := reflect.ValueOf(strct)
	t := reflect.TypeOf(strct)

	// 处理指针
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
		t = t.Elem()
	}

	var sections []string

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		if jsonTag := field.Tag.Get("json"); jsonTag != "" {
			name := strings.Split(jsonTag, ",")[0]
			if name != "-" {
				fieldType := field.Type
				// 处理切片字段
				if fieldType.Kind() == reflect.Slice {
					if fieldValue.Len() > 0 {
						sliceTOON := tc.convertSliceToTOON(fieldValue.Interface())
						sections = append(sections, fmt.Sprintf("%s", sliceTOON))
					}
				} else {
					// 处理普通字段
					value := tc.formatValue(fieldValue.Interface())
					sections = append(sections, fmt.Sprintf("%s: %s", name, value))
				}
			}
		}
	}

	return strings.Join(sections, "\n\n")
}

// formatValue 格式化值
func (tc *TOONConverter) formatValue(value interface{}) string {
	switch v := value.(type) {
	case nil:
		return ""
	case bool:
		return strconv.FormatBool(v)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case string:
		return v
	case []interface{}:
		var items []string
		for _, item := range v {
			items = append(items, tc.formatValue(item))
		}
		return "[" + strings.Join(items, ",") + "]"
	default:
		// 对于其他结构体，递归处理
		rv := reflect.ValueOf(value)
		if rv.Kind() == reflect.Ptr {
			rv = rv.Elem()
		}
		if rv.Kind() == reflect.Struct {
			return tc.convertStructToTOON(value)
		}
		return fmt.Sprintf("%v", v)
	}
}
