package workflow

import (
	"strings"
	"testing"

	"github.com/jbrunton/gflows/fixtures"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func setupValidator(workflowContent string, config string) (*afero.Afero, *Validator, *Definition) {
	container, context, _ := fixtures.NewTestContext(config)
	fs := container.FileSystem()
	Definition := newTestWorkflowDefinition("test", workflowContent)
	validator := NewValidator(fs, context)
	return fs, validator, Definition
}

func TestValidateContent(t *testing.T) {
	workflowContent := fixtures.ExampleWorkflow("test.jsonnet")
	fs, validator, definition := setupValidator(workflowContent, "")

	fs.WriteFile(definition.Destination, []byte(workflowContent), 0644)
	result := validator.ValidateContent(definition)

	assert.True(t, result.Valid)
	assert.Equal(t, []string{}, result.Errors)
}

func TestValidateContentMissing(t *testing.T) {
	_, validator, definition := setupValidator(fixtures.ExampleWorkflow("test.jsonnet"), "")

	result := validator.ValidateContent(definition)

	assert.False(t, result.Valid)
	assert.Equal(t, []string{"Workflow missing for \"test\" (expected workflow at .github/workflows/test.yml)"}, result.Errors)
}

func TestValidateContentOutOfDate(t *testing.T) {
	fs, validator, definition := setupValidator(fixtures.ExampleWorkflow("test.jsonnet"), "")

	fs.WriteFile(definition.Destination, []byte("incorrect content"), 0644)
	result := validator.ValidateContent(definition)

	assert.False(t, result.Valid)
	assert.Equal(t, []string{"Content is out of date for \"test\" (.github/workflows/test.yml)"}, result.Errors)
}

func TestValidateSchema(t *testing.T) {
	_, validator, definition := setupValidator(fixtures.ExampleWorkflow("test.jsonnet"), "")

	result := validator.ValidateSchema(definition)

	assert.True(t, result.Valid)
	assert.Equal(t, []string{}, result.Errors)
}

func TestValidateSchemaMissingField(t *testing.T) {
	_, validator, definition := setupValidator(fixtures.InvalidJsonnetWorkflow, "")

	result := validator.ValidateSchema(definition)

	assert.False(t, result.Valid)
	assert.Equal(t, []string{"(root): jobs is required"}, result.Errors)
}

func TestValidateContentEnabledFlags(t *testing.T) {
	scenarios := []struct {
		config         string
		workflow       string
		expectedResult ValidationResult
	}{
		{
			config: strings.Join([]string{
				"templates:",
				"  engine: ytt",
				"workflows:",
				"  defaults:",
				"    checks:",
				"      content:",
				"        enabled: false",
			}, "\n"),
			workflow: "",
			expectedResult: ValidationResult{
				Valid:  true,
				Errors: []string{"Content checks disabled for test, skipping"},
			},
		},
		{
			config: strings.Join([]string{
				"templates:",
				"  engine: ytt",
				"workflows:",
				"  defaults:",
				"    checks:",
				"      content:",
				"        enabled: true",
			}, "\n"),
			workflow: "",
			expectedResult: ValidationResult{
				Valid:  false,
				Errors: []string{`Workflow missing for "test" (expected workflow at .github/workflows/test.yml)`},
			},
		},
	}

	for _, scenario := range scenarios {
		_, validator, definition := setupValidator(fixtures.ExampleWorkflow("test.jsonnet"), scenario.config)
		result := validator.ValidateContent(definition)
		assert.Equal(t, scenario.expectedResult, result)
	}
}

func TestValidateSchemaEnabledFlags(t *testing.T) {
	scenarios := []struct {
		config         string
		workflow       string
		expectedResult ValidationResult
	}{
		{
			config: strings.Join([]string{
				"templates:",
				"  engine: ytt",
				"workflows:",
				"  defaults:",
				"    checks:",
				"      schema:",
				"        enabled: false",
			}, "\n"),
			workflow: "",
			expectedResult: ValidationResult{
				Valid:  true,
				Errors: []string{"Schema checks disabled for test, skipping"},
			},
		},
		{
			config: strings.Join([]string{
				"templates:",
				"  engine: ytt",
				"workflows:",
				"  defaults:",
				"    checks:",
				"      schema:",
				"        enabled: true",
				"  overrides:",
				"    test:",
				"      checks:",
				"        schema:",
				"          enabled: false",
			}, "\n"),
			workflow: "",
			expectedResult: ValidationResult{
				Valid:  true,
				Errors: []string{"Schema checks disabled for test, skipping"},
			},
		},
		{
			config: strings.Join([]string{
				"templates:",
				"  engine: ytt",
				"workflows:",
				"  defaults:",
				"    checks:",
				"      schema:",
				"        enabled: true",
			}, "\n"),
			workflow: "",
			expectedResult: ValidationResult{
				Valid:  false,
				Errors: []string{"(root): jobs is required"},
			},
		},
	}

	for _, scenario := range scenarios {
		_, validator, definition := setupValidator(fixtures.InvalidJsonnetWorkflow, scenario.config)
		result := validator.ValidateSchema(definition)
		assert.Equal(t, scenario.expectedResult, result)
	}
}
