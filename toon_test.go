package gotoon

import (
	"testing"
)

func TestEncode(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name: "simple map",
			input: map[string]interface{}{
				"name": "Alice",
				"age":  30,
			},
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
			expected: `[ 2 {id, name}:
  1, "Alice"
  2, "Bob"
]`,
		},
		{
			name:     "nil input",
			input:    nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Encode(tt.input)
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

	result, err := EncodeJSON(jsonStr)
	if err != nil {
		t.Fatalf("EncodeJSON failed: %v", err)
	}
	if result != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
	}
}
