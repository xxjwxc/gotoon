package gotoon

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestEncode(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		options  Options
		expected string
	}{
		{
			name: "simple map",
			input: map[string]interface{}{
				"name": "Alice",
				"age":  30,
			},
			options: DefaultOptions(),
			expected: `{
  age: 30,
  name: "Alice"
}`,
		},
		{
			name: "array with tabular format",
			input: []interface{}{
				map[string]interface{}{"id": 1, "name": "Alice"},
				map[string]interface{}{"id": 2, "name": "Bob"},
			},
			options: DefaultOptions(),
			expected: `[ 2 {id, name}:
  1, "Alice"
  2, "Bob"
]`,
		},
		{
			name:     "nil input",
			input:    nil,
			options:  DefaultOptions(),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Encode(tt.input, tt.options)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected:\n%s\nGot:\n%s", tt.expected, result)
			}
		})
	}
}

func TestEncodeJSON(t *testing.T) {
	jsonStr := `{"users": [{"id": 1, "name": "Alice"}, {"id": 2, "name": "Bob"}]}`
	expected := `{
  users: [ 2 {id, name}:
    1, "Alice"
    2, "Bob"
  ]
}`

	result, err := EncodeJSON(jsonStr, DefaultOptions())
	if err != nil {
		t.Fatalf("EncodeJSON failed: %v", err)
	}
	if result != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestDecode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "simple object",
			input: `{
  name: "Alice",
  age: 30
}`,
			expected: `{
  "age": 30,
  "name": "Alice"
}`,
		},
		{
			name: "tabular array",
			input: `[ 2 {id, name}:
  1, "Alice"
  2, "Bob"
]`,
			expected: `[
  {
    "id": 1,
    "name": "Alice"
  },
  {
    "id": 2,
    "name": "Bob"
  }
]`,
		},
		{
			name: "nested structure",
			input: `{
  metadata: {
    version: "1.0",
    timestamp: 1735689600
  },
  users: [ 2 {id, name, role}:
    1, "Alice", "admin"
    2, "Bob", "user"
  ]
}`,
			expected: `{
  "metadata": {
    "timestamp": 1735689600,
    "version": "1.0"
  },
  "users": [
    {
      "id": 1,
      "name": "Alice",
      "role": "admin"
    },
    {
      "id": 2,
      "name": "Bob",
      "role": "user"
    }
  ]
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DecodeJSON(tt.input)
			if err != nil {
				t.Fatalf("Decode failed: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected:\n%s\nGot:\n%s", tt.expected, result)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	jsonStr := `{
  "name": "Test",
  "values": [1, 2, 3],
  "data": [
    {"id": 1, "value": "A"},
    {"id": 2, "value": "B"}
  ]
}`

	// JSON -> TOON
	toonStr, err := EncodeJSON(jsonStr, DefaultOptions())
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	// TOON -> JSON
	resultJSON, err := DecodeJSON(toonStr)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// 比较结果
	var expected, result interface{}
	json.Unmarshal([]byte(jsonStr), &expected)
	json.Unmarshal([]byte(resultJSON), &result)

	if !reflect.DeepEqual(expected, result) {
		t.Errorf("Round trip failed\nExpected:\n%v\nGot:\n%v", expected, result)
	}
}
