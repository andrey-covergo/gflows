package lib

import (
	"fmt"

	"github.com/spf13/afero"
)

func newTestContext() (*afero.Afero, *JFlowsContext) {
	fs := CreateMemFs()
	context := &JFlowsContext{
		Dir:       ".jflows",
		GitHubDir: ".github/",
	}
	return fs, context
}

const invalidTemplate = `
local workflow = {
  on: {
    push: {
      branches: [ "develop" ],
    },
  }
};
std.manifestYamlDoc(workflow)
`

const invalidWorkflow = `# File generated by jflows, do not modify
# Source: .jflows/workflows/test.jsonnet
"on":
  "push":
    "branches":
    - "develop"
`

const exampleTemplate = `
local workflow = {
  on: {
    push: {
      branches: [ "develop" ],
    },
  },
	jobs: {
		test: {
			"runs-on": "ubuntu-latest",
			steps: [
			  { run: "echo Hello, World!" }
      ]
    }
  }
};
std.manifestYamlDoc(workflow)
`

const exampleWorkflow = `# File generated by jflows, do not modify
# Source: .jflows/workflows/test.jsonnet
"jobs":
  "test":
    "runs-on": "ubuntu-latest"
    "steps":
    - "run": "echo Hello, World!"
"on":
  "push":
    "branches":
    - "develop"
`

func newTestWorkflowDefinition(name string, content string) *WorkflowDefinition {
	return &WorkflowDefinition{
		Name:        name,
		Source:      fmt.Sprintf(".jflows/workflows/%s.jsonnet", name),
		Destination: fmt.Sprintf(".github/workflows/%s.yml", name),
		Content:     content,
	}
}