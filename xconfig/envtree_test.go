package xconfig_test

import (
	"testing"

	"github.com/raphoester/x/xconfig"
	"github.com/stretchr/testify/assert"
)

type testStruct struct {
	TestString    string            `yaml:"test_string"`
	TestInt       int               `yaml:"test_int"`
	TestStruct    subTestStruct     `yaml:"test_struct"`
	TestByteArray xconfig.ByteArray `yaml:"test_byte_array"`
}

type subTestStruct struct {
	String string `yaml:"sub_string"`
}

func TestApply(t *testing.T) {
	tests := []struct {
		name      string
		envVars   map[string]string
		input     interface{}
		expected  interface{}
		expectErr bool
	}{
		{
			name: "basic native top-level fields",
			envVars: map[string]string{
				"TEST_STRING": "hello",
				"TEST_INT":    "42",
			},
			input: &testStruct{},
			expected: &testStruct{
				TestString: "hello",
				TestInt:    42,
			},
			expectErr: false,
		}, {
			name: "invalid int value",
			envVars: map[string]string{
				"TEST_INT":    "not an int",
				"TEST_STRING": "hello",
			},
			input: &testStruct{},
			expected: &testStruct{
				TestString: "hello",
			},
			expectErr: true,
		}, {
			name: "basic native nested fields",
			envVars: map[string]string{
				"TEST_STRUCT_SUB_STRING": "hello world",
			},
			input: &testStruct{},
			expected: &testStruct{
				TestStruct: subTestStruct{
					String: "hello world",
				},
			},
			expectErr: false,
		}, {
			name: "byte array decoded as base64",
			envVars: map[string]string{
				"TEST_BYTE_ARRAY": "aGVsbG8gd29ybGQ=",
			},
			input: &testStruct{},
			expected: &testStruct{
				TestByteArray: []byte("hello world"),
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			err := xconfig.ApplyEnv(tt.input)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Equal(t, tt.expected, tt.input)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, tt.input)
			}
		})
	}
}
