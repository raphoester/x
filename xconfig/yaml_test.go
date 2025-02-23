package xconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestStruct struct {
	TestString    string      `yaml:"test_string"`
	TestInt       int         `yaml:"test_int"`
	TestStruct2   TestStruct2 `yaml:"test_struct2"`
	TestByteArray ByteArray   `yaml:"test_byte_array"`
}

type TestStruct2 struct {
	TestString2 string `yaml:"test_string2"`
	TestInt2    int    `yaml:"test_int2"`
}

func TestLoadYamlFilesInConfig(t *testing.T) {
	testCases := []struct {
		name                string
		filesContents       map[string]string
		fileNames           []string
		dest                any
		expected            any
		expectErr           bool
		expectedErrContains string
	}{
		{
			name:      "basic native fields",
			fileNames: []string{"test.yaml"},
			filesContents: map[string]string{"test.yaml": `
test_string: hello
test_int: 42
test_struct2:
  test_string2: world
  test_int2: 24
`,
			},
			dest: &TestStruct{},
			expected: &TestStruct{
				TestString: "hello",
				TestInt:    42,
				TestStruct2: TestStruct2{
					TestString2: "world",
					TestInt2:    24,
				},
			},
			expectErr: false,
		}, {
			name:      "files conflicting each other",
			fileNames: []string{"test.yaml", "test2.yaml"},
			filesContents: map[string]string{
				"test.yaml": `
test_string: hello
test_int: 42`,
				"test2.yaml": `
test_string: world`,
			},
			dest: &TestStruct{},
			expected: &TestStruct{
				TestString: "world", // we expect the last file to overwrite the first one
				TestInt:    42,      // the non-conflicting value should not be overwritten
			},
			expectErr: false,
		}, {
			name:      "missing file",
			fileNames: []string{"missing.yaml", "test2.yaml"},
			filesContents: map[string]string{
				"test2.yaml": `
test_string: world`,
			},
			dest: &TestStruct{},
			expected: &TestStruct{
				TestString: "world", // we still expect the second file to be loaded
			},
			expectErr:           true,
			expectedErrContains: "unable to find config file",
		}, {
			name:      "invalid yaml format",
			fileNames: []string{"test.yaml", "test2.yaml"},
			filesContents: map[string]string{
				"test.yaml": `123456this is not a valid yaml content`,
				"test2.yaml": `
test_string: world`,
			},
			dest: &TestStruct{},
			expected: &TestStruct{
				TestString: "world", // we still expect the second file to be loaded
			},
			expectErr: true,
			// yamlNode.Decode will fail, but interestingly enough, not yaml.Unmarshal
			expectedErrContains: "unable to decode yaml config file",
		}, {
			name:      "invalid field type",
			fileNames: []string{"test.yaml"},
			filesContents: map[string]string{"test.yaml": `
test_string: hello
test_int: not_an_int`,
			},
			dest: &TestStruct{},
			expected: &TestStruct{
				TestString: "hello",
			},
			expectErr:           true,
			expectedErrContains: "unable to decode yaml config file",
		}, {
			name:      "byte array decoded as base64",
			fileNames: []string{"test.yaml"},
			dest:      &TestStruct{},
			filesContents: map[string]string{"test.yaml": `
test_byte_array: aGVsbG8gd29ybGQ=`,
			},

			expected: &TestStruct{
				TestByteArray: []byte("hello world"),
			},
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name,
			func(t *testing.T) {

				err := loadRawYamlContents(tc.dest, tc.filesContents, tc.fileNames...)

				if tc.expectErr {
					require.Error(t, err)
					assert.Contains(t, err.Error(), tc.expectedErrContains)
				} else {
					require.NoError(t, err)
				}

				assert.Equal(t, tc.expected, tc.dest)
			},
		)
	}
}
