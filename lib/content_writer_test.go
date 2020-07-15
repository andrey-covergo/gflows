package lib

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogErrors(t *testing.T) {
	out := new(bytes.Buffer)
	writer := NewContentWriter(CreateMemFs(), out)

	writer.LogErrors("path/to/file", "error message", []string{"error details"})

	expectedOutput := "      error path/to/file error message\n  ► error details\n"
	assert.Equal(t, expectedOutput, out.String())
}

func TestSafelyWriteFile(t *testing.T) {
	fs := CreateMemFs()
	writer := NewContentWriter(fs, new(bytes.Buffer))

	writer.SafelyWriteFile("path/to/file", "foobar")

	actualContent, _ := fs.ReadFile("path/to/file")
	assert.Equal(t, "foobar", string(actualContent))
}

func TestUpdateFileContentCreate(t *testing.T) {
	fs := CreateMemFs()
	out := new(bytes.Buffer)
	writer := NewContentWriter(fs, out)

	writer.UpdateFileContent("path/to/file", "foobar", "(baz)")

	actualContent, _ := fs.ReadFile("path/to/file")
	assert.Equal(t, "foobar", string(actualContent))
	assert.Equal(t, "     create path/to/file (baz)\n", out.String())
}

func TestUpdateFileContentUpdate(t *testing.T) {
	fs := CreateMemFs()
	out := new(bytes.Buffer)
	writer := NewContentWriter(fs, out)

	writer.SafelyWriteFile("path/to/file", "foo")
	writer.UpdateFileContent("path/to/file", "foobar", "(baz)")

	actualContent, _ := fs.ReadFile("path/to/file")
	assert.Equal(t, "foobar", string(actualContent))
	assert.Equal(t, "     update path/to/file (baz)\n", out.String())
}

func TestUpdateFileContentIdentical(t *testing.T) {
	fs := CreateMemFs()
	out := new(bytes.Buffer)
	writer := NewContentWriter(fs, out)

	writer.SafelyWriteFile("path/to/file", "foobar")
	writer.UpdateFileContent("path/to/file", "foobar", "(baz)")

	actualContent, _ := fs.ReadFile("path/to/file")
	assert.Equal(t, "foobar", string(actualContent))
	assert.Equal(t, "  identical path/to/file (baz)\n", out.String())
}
