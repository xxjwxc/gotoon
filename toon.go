package gotoon

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Options 配置选项
type Options struct {
	IndentSize     int    // 缩进大小
	Delimiter      string // 列分隔符
	UseTabular     bool   // 启用表格格式
	KeyFolding     bool   // 启用键折叠
	ShowArraySizes bool   // 显示数组大小
}

// DefaultOptions 返回默认配置选项
func DefaultOptions() Options {
	return Options{
		IndentSize:     2,
		Delimiter:      ", ",
		UseTabular:     true,
		KeyFolding:     true,
		ShowArraySizes: true,
	}
}

// Encode 将任意数据结构编码为TOON格式
func Encode(data interface{}, options Options) (string, error) {
	if data == nil {
		return "", nil
	}

	var builder strings.Builder
	err := encodeValue(data, options, 0, &builder)
	if err != nil {
		return "", err
	}

	return builder.String(), nil
}

// EncodeJSON 直接从JSON字符串编码为TOON格式
func EncodeJSON(jsonStr string, options Options) (string, error) {
	var data interface{}
	err := json.Unmarshal([]byte(jsonStr), &data)
	if err != nil {
		return "", err
	}

	return Encode(data, options)
}

// Decode 将TOON格式字符串解码为Go数据结构
func Decode(toonStr string) (interface{}, error) {
	lexer := newLexer(toonStr)
	parser := newParser(lexer)
	return parser.parse()
}

// DecodeJSON 将TOON格式字符串解码为JSON字符串
func DecodeJSON(toonStr string) (string, error) {
	data, err := Decode(toonStr)
	if err != nil {
		return "", err
	}

	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

func encodeValue(value interface{}, options Options, indent int, builder *strings.Builder) error {
	rv := reflect.ValueOf(value)
	for rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface {
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Map:
		return encodeMap(value, options, indent, builder)
	case reflect.Slice, reflect.Array:
		return encodeArray(value, options, indent, builder)
	case reflect.String:
		builder.WriteString(fmt.Sprintf("%q", rv.String()))
	case reflect.Bool:
		builder.WriteString(fmt.Sprintf("%t", rv.Bool()))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		builder.WriteString(fmt.Sprintf("%d", rv.Int()))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		builder.WriteString(fmt.Sprintf("%d", rv.Uint()))
	case reflect.Float32, reflect.Float64:
		builder.WriteString(fmt.Sprintf("%g", rv.Float()))
	default:
		if rv.CanInterface() {
			return encodeValue(rv.Interface(), options, indent, builder)
		}
		builder.WriteString(fmt.Sprintf("%v", value))
	}

	return nil
}

func encodeMap(value interface{}, options Options, indent int, builder *strings.Builder) error {
	m, ok := value.(map[string]interface{})
	if !ok {
		rv := reflect.ValueOf(value)
		if rv.Kind() != reflect.Map {
			return fmt.Errorf("expected map, got %T", value)
		}

		m = make(map[string]interface{})
		for _, k := range rv.MapKeys() {
			if k.Kind() == reflect.String {
				m[k.String()] = rv.MapIndex(k).Interface()
			}
		}
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	writeIndent(builder, indent, options)
	builder.WriteString("{")
	if len(m) > 0 {
		builder.WriteString("\n")
	}

	for i, k := range keys {
		v := m[k]
		writeIndent(builder, indent+options.IndentSize, options)
		builder.WriteString(fmt.Sprintf("%s: ", k))
		err := encodeValue(v, options, indent+options.IndentSize, builder)
		if err != nil {
			return err
		}
		if i < len(m)-1 {
			builder.WriteString(",")
		}
		builder.WriteString("\n")
	}

	if len(m) > 0 {
		writeIndent(builder, indent, options)
	}
	builder.WriteString("}")

	return nil
}

func encodeArray(value interface{}, options Options, indent int, builder *strings.Builder) error {
	var slice []interface{}
	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		slice = make([]interface{}, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			slice[i] = rv.Index(i).Interface()
		}
	} else {
		return fmt.Errorf("expected slice or array, got %T", value)
	}

	useTabular := options.UseTabular && canUseTabularFormat(slice)

	// 直接写入数组开始，不额外增加缩进
	builder.WriteString("[")
	if options.ShowArraySizes {
		builder.WriteString(fmt.Sprintf(" %d ", len(slice)))
	}

	if useTabular {
		builder.WriteString("{")
		keys := getCommonKeys(slice)
		builder.WriteString(strings.Join(keys, options.Delimiter))
		builder.WriteString("}:\n")

		// 写入每行数据
		for i, item := range slice {
			writeIndent(builder, indent+options.IndentSize, options)
			row := make([]string, 0, len(keys))
			itemMap, _ := item.(map[string]interface{})
			for _, k := range keys {
				val := itemMap[k]
				row = append(row, formatValue(val))
			}
			builder.WriteString(strings.Join(row, options.Delimiter))
			if i < len(slice)-1 {
				builder.WriteString("\n")
			}
		}
	} else {
		// 普通数组格式输出
		if len(slice) > 0 {
			builder.WriteString("\n")
		}

		// 写入每个元素
		for i, item := range slice {
			writeIndent(builder, indent+options.IndentSize, options)
			err := encodeValue(item, options, indent+options.IndentSize, builder)
			if err != nil {
				return err
			}
			if i < len(slice)-1 {
				builder.WriteString(",\n")
			}
		}

		// 写入数组结束
		if len(slice) > 0 {
			builder.WriteString("\n")
			writeIndent(builder, indent, options)
		}
		builder.WriteString("]")
	}

	// 表格格式需要单独处理数组结束符
	if useTabular {
		builder.WriteString("\n")
		writeIndent(builder, indent, options)
		builder.WriteString("]")
	}

	return nil
}

func canUseTabularFormat(slice []interface{}) bool {
	if len(slice) == 0 {
		return false
	}

	for _, item := range slice {
		if _, ok := item.(map[string]interface{}); !ok {
			return false
		}
	}

	keys := getCommonKeys(slice)
	return len(keys) > 0
}

func getCommonKeys(slice []interface{}) []string {
	if len(slice) == 0 {
		return nil
	}

	first, _ := slice[0].(map[string]interface{})
	commonKeys := make(map[string]bool)
	for k := range first {
		commonKeys[k] = true
	}

	for _, item := range slice[1:] {
		itemMap, _ := item.(map[string]interface{})
		for k := range commonKeys {
			if _, exists := itemMap[k]; !exists {
				delete(commonKeys, k)
			}
		}
		if len(commonKeys) == 0 {
			break
		}
	}

	keys := make([]string, 0, len(commonKeys))
	for k := range commonKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return keys
}

func formatValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("%q", v)
	case bool:
		return fmt.Sprintf("%t", v)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%g", v)
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%v", v)
	}
}

func writeIndent(builder *strings.Builder, indent int, options Options) {
	builder.WriteString(strings.Repeat(" ", indent))
}

// ------------------------------
// TOON解析器实现
// ------------------------------

// tokenType 标记类型
type tokenType int

const (
	tokenEOF        tokenType = iota
	tokenLBrace               // {
	tokenRBrace               // }
	tokenLBracket             // [
	tokenRBracket             // ]
	tokenColon                // :
	tokenComma                // ,
	tokenNumber               // 数字
	tokenString               // 字符串
	tokenBoolean              // true/false
	tokenNull                 // null
	tokenIdentifier           // 标识符
	tokenColonColon           // ::
	tokenSize                 // 数组大小数字
)

// token 标记
type token struct {
	typ   tokenType
	value string
	pos   int
}

// lexer 词法分析器
type lexer struct {
	input string
	pos   int
	start int
	width int
}

func newLexer(input string) *lexer {
	return &lexer{
		input: input,
	}
}

func (l *lexer) nextToken() token {
	for {
		l.skipWhitespace()
		l.start = l.pos

		if l.pos >= len(l.input) {
			return l.emit(tokenEOF)
		}

		c := l.next()
		switch c {
		case '{':
			return l.emit(tokenLBrace)
		case '}':
			return l.emit(tokenRBrace)
		case '[':
			return l.emit(tokenLBracket)
		case ']':
			return l.emit(tokenRBracket)
		case ':':
			if l.peek() == ':' {
				l.next()
				return l.emit(tokenColonColon)
			}
			return l.emit(tokenColon)
		case ',':
			return l.emit(tokenComma)
		case '"':
			return l.scanString()
		case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			return l.scanNumber()
		default:
			if isAlpha(c) {
				return l.scanIdentifier()
			}
			return l.emit(tokenEOF)
		}
	}
}

func (l *lexer) next() rune {
	if l.pos >= len(l.input) {
		l.width = 0
		return rune(-1)
	}
	r := rune(l.input[l.pos])
	l.width = 1
	if r >= 0x80 {
		l.width = utf8.RuneLen(r)
		if l.width < 0 {
			l.width = 1
		} else if l.pos+l.width > len(l.input) {
			l.width = len(l.input) - l.pos
		}
	}
	l.pos += l.width
	return r
}

func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

func (l *lexer) backup() {
	l.pos -= l.width
}

func (l *lexer) emit(typ tokenType) token {
	tok := token{
		typ:   typ,
		value: l.input[l.start:l.pos],
		pos:   l.start,
	}
	return tok
}

func (l *lexer) skipWhitespace() {
	for {
		r := l.next()
		if r == rune(-1) || !unicode.IsSpace(r) {
			l.backup()
			break
		}
	}
}

func (l *lexer) scanString() token {
	for {
		r := l.next()
		if r == '"' || r == rune(-1) {
			break
		}
		if r == '\\' {
			l.next()
		}
	}
	return l.emit(tokenString)
}

func (l *lexer) scanNumber() token {
	if l.input[l.pos-1] == '-' {
		l.next()
	}

	for {
		if l.pos >= len(l.input) {
			break
		}
		c := l.input[l.pos]
		if c >= '0' && c <= '9' {
			l.pos++
			continue
		}
		if c == '.' {
			l.pos++
			for l.pos < len(l.input) && l.input[l.pos] >= '0' && l.input[l.pos] <= '9' {
				l.pos++
			}
		}
		if c == 'e' || c == 'E' {
			l.pos++
			if l.pos < len(l.input) && (l.input[l.pos] == '+' || l.input[l.pos] == '-') {
				l.pos++
			}
			for l.pos < len(l.input) && l.input[l.pos] >= '0' && l.input[l.pos] <= '9' {
				l.pos++
			}
		}
		break
	}

	return l.emit(tokenNumber)
}

func (l *lexer) scanIdentifier() token {
	for {
		r := l.next()
		if !isAlphaNumeric(r) {
			l.backup()
			break
		}
	}

	val := l.input[l.start:l.pos]
	switch val {
	case "true", "false":
		return l.emit(tokenBoolean)
	case "null":
		return l.emit(tokenNull)
	default:
		return l.emit(tokenIdentifier)
	}
}

func isAlpha(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func isAlphaNumeric(c rune) bool {
	return isAlpha(c) || (c >= '0' && c <= '9')
}

// parser 语法分析器
type parser struct {
	lexer *lexer
	cur   token
	peek  token
}

func newParser(lexer *lexer) *parser {
	p := &parser{lexer: lexer}
	p.nextToken()
	p.nextToken()
	return p
}

func (p *parser) nextToken() {
	p.cur = p.peek
	p.peek = p.lexer.nextToken()
}

func (p *parser) parse() (interface{}, error) {
	return p.parseValue()
}

func (p *parser) parseValue() (interface{}, error) {
	switch p.cur.typ {
	case tokenLBrace:
		return p.parseObject()
	case tokenLBracket:
		return p.parseArray()
	case tokenString:
		val, err := strconv.Unquote(p.cur.value)
		if err != nil {
			val = p.cur.value
		}
		p.nextToken()
		return val, nil
	case tokenNumber:
		if strings.Contains(p.cur.value, ".") || strings.Contains(p.cur.value, "e") || strings.Contains(p.cur.value, "E") {
			val, _ := strconv.ParseFloat(p.cur.value, 64)
			p.nextToken()
			return val, nil
		}
		val, _ := strconv.ParseInt(p.cur.value, 10, 64)
		p.nextToken()
		return val, nil
	case tokenBoolean:
		val := p.cur.value == "true"
		p.nextToken()
		return val, nil
	case tokenNull:
		p.nextToken()
		return nil, nil
	case tokenIdentifier:
		val := p.cur.value
		p.nextToken()
		return val, nil
	default:
		return nil, fmt.Errorf("unexpected token %v at position %d", p.cur.typ, p.cur.pos)
	}
}

func (p *parser) parseObject() (map[string]interface{}, error) {
	obj := make(map[string]interface{})
	p.nextToken() // 跳过 {

	for p.cur.typ != tokenRBrace && p.cur.typ != tokenEOF {
		if p.cur.typ != tokenIdentifier && p.cur.typ != tokenString {
			return nil, fmt.Errorf("expected key at position %d", p.cur.pos)
		}

		key := p.cur.value
		if p.cur.typ == tokenString {
			key, _ = strconv.Unquote(key)
		}
		p.nextToken()

		if p.cur.typ != tokenColon {
			return nil, fmt.Errorf("expected colon at position %d", p.cur.pos)
		}
		p.nextToken()

		value, err := p.parseValue()
		if err != nil {
			return nil, err
		}

		obj[key] = value

		if p.cur.typ == tokenComma {
			p.nextToken()
		}
	}

	if p.cur.typ != tokenRBrace {
		return nil, fmt.Errorf("unclosed object")
	}
	p.nextToken() // 跳过 }

	return obj, nil
}

func (p *parser) parseArray() (interface{}, error) {
	var arr []interface{}
	p.nextToken() // 跳过 [

	// 检查是否是表格格式
	var tableKeys []string
	if p.cur.typ == tokenNumber {
		// 跳过数组大小
		p.nextToken()
	}

	if p.cur.typ == tokenLBrace {
		p.nextToken() // 跳过 {
		tableKeys = p.parseTableKeys()
		if p.cur.typ == tokenRBrace {
			p.nextToken() // 跳过 }
		}
		if p.cur.typ == tokenColon {
			p.nextToken() // 跳过 :
		}
	}

	if len(tableKeys) > 0 {
		// 解析表格格式数据
		return p.parseTableFormat(tableKeys)
	}

	// 解析普通数组
	for p.cur.typ != tokenRBracket && p.cur.typ != tokenEOF {
		value, err := p.parseValue()
		if err != nil {
			return nil, err
		}

		arr = append(arr, value)

		if p.cur.typ == tokenComma {
			p.nextToken()
		}
	}

	if p.cur.typ != tokenRBracket {
		return nil, fmt.Errorf("unclosed array")
	}
	p.nextToken() // 跳过 ]

	return arr, nil
}

func (p *parser) parseTableKeys() []string {
	var keys []string

	for p.cur.typ != tokenRBrace && p.cur.typ != tokenEOF {
		if p.cur.typ == tokenIdentifier {
			keys = append(keys, p.cur.value)
			p.nextToken()
		} else if p.cur.typ == tokenString {
			key, _ := strconv.Unquote(p.cur.value)
			keys = append(keys, key)
			p.nextToken()
		}

		if p.cur.typ == tokenComma {
			p.nextToken()
		}
	}

	return keys
}

func (p *parser) parseTableFormat(keys []string) ([]interface{}, error) {
	var arr []interface{}

	for p.cur.typ != tokenRBracket && p.cur.typ != tokenEOF {
		if p.cur.typ == tokenEOF {
			break
		}

		// 跳过缩进
		if p.cur.typ == tokenIdentifier && isAllWhitespace(p.cur.value) {
			p.nextToken()
			continue
		}

		obj := make(map[string]interface{})
		for i, key := range keys {
			if p.cur.typ == tokenEOF {
				break
			}

			value, err := p.parseValue()
			if err != nil {
				return nil, err
			}

			obj[key] = value

			if i < len(keys)-1 && p.cur.typ == tokenComma {
				p.nextToken()
			}
		}

		if len(obj) > 0 {
			arr = append(arr, obj)
		}
	}

	// 找到数组结束符
	for p.cur.typ != tokenRBracket && p.cur.typ != tokenEOF {
		p.nextToken()
	}
	if p.cur.typ == tokenRBracket {
		p.nextToken() // 跳过 ]
	}

	return arr, nil
}

func isAllWhitespace(s string) bool {
	for _, c := range s {
		if !unicode.IsSpace(c) {
			return false
		}
	}
	return true
}
