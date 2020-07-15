package lib

import (
	"archive/zip"
	"bytes"
	"fmt"
	"net/http"

	statikFs "github.com/rakyll/statik/fs"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

func newTestCommand() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().String("config", "", "")
	return cmd
}

func newTestContext(cmd *cobra.Command, config string) (*afero.Afero, *GFlowsContext) {
	fs := CreateMemFs()
	fs.WriteFile(".gflows/config.yml", []byte(config), 0644)
	context, _ := GetContext(fs, cmd)
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

const invalidWorkflow = `# File generated by gflows, do not modify
# Source: .gflows/workflows/test.jsonnet
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

const exampleWorkflow = `# File generated by gflows, do not modify
# Source: .gflows/workflows/test.jsonnet
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
		Source:      fmt.Sprintf(".gflows/workflows/%s.jsonnet", name),
		Destination: fmt.Sprintf(".github/workflows/%s.yml", name),
		Content:     content,
	}
}

type file struct {
	Name string
	Body string
}

func createTestFileSystem(files []file, assetNamespace string) http.FileSystem {
	out := new(bytes.Buffer)
	writer := zip.NewWriter(out)
	for _, file := range files {
		f, err := writer.Create(file.Name)
		if err != nil {
			panic(err)
		}
		_, err = f.Write([]byte(file.Body))
		if err != nil {
			panic(err)
		}
	}
	err := writer.Close()
	if err != nil {
		panic(err)
	}
	asset := out.String()
	statikFs.RegisterWithNamespace(assetNamespace, asset)
	sourceFs, err := statikFs.NewWithNamespace(assetNamespace)
	if err != nil {
		panic(err)
	}
	return sourceFs
}
