# gotoon
golang库将json数据结构转成‌TOON（Token-Oriented Object Notation）输出


# 安装
```bash
go get github.com/xxjwxc/gotoon
```

# 使用
```go
package main

import (
	"encoding/json"
	"fmt"
	"github.com/xxjwxc/toon"
)

func main() {
	jsonData := `{"users": [{"id": 1, "name": "Alice"}]}`
	var data interface{}
	json.Unmarshal([]byte(jsonData), &data)
	
	toonOutput, _ := toon.Encode(data, toon.DefaultOptions())
	fmt.Println(toonOutput)
}
```

# 说明
## 主要 API

``` go
// 核心编码函数
func Encode(data interface{}, options Options) (string, error)

// 直接从JSON字符串编码
func EncodeJSON(jsonStr string, options Options) (string, error)

// 将TOON格式字符串解码为Go数据结构
func Decode(toonStr string) (interface{}, error) 
// 将TOON格式字符串解码为JSON字符串
func DecodeJSON(toonStr string) (string, error) 

// 配置选项
type Options struct {
    IndentSize     int    // 缩进大小
    Delimiter      string // 列分隔符
    UseTabular     bool   // 启用表格格式
    KeyFolding     bool   // 启用键折叠
    ShowArraySizes bool   // 显示数组大小
}
```