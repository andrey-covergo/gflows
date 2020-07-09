package lib

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
)

type GitWorkflow struct {
	filename   string
	definition *WorkflowDefinition
}

func getWorkflows(fs *afero.Afero, context *JFlowsContext) []GitWorkflow {
	files := []string{}
	files, err := afero.Glob(fs, filepath.Join(context.GitHubDir, "workflows/*.yml"))
	if err != nil {
		panic(err)
	}

	definitions := GetWorkflowDefinitions(fs, context)

	var workflows []GitWorkflow

	for _, file := range files {
		workflow := GitWorkflow{filename: file}
		for _, definition := range definitions {
			if definition.Destination == file {
				workflow.definition = definition
				break
			}
		}
		workflows = append(workflows, workflow)
	}

	return workflows
}

func ImportWorkflows(fs *afero.Afero, context *JFlowsContext) {
	workflows := getWorkflows(fs, context)
	for _, workflow := range workflows {
		fmt.Println("Found workflow:", workflow.filename)
		if workflow.definition == nil {
			workflowContent, err := fs.ReadFile(workflow.filename)
			if err != nil {
				panic(err)
			}
			var yamlData map[interface{}]interface{}
			err = yaml.Unmarshal(workflowContent, &yamlData)
			if err != nil {
				panic(err)
			}

			jsonData, err := convertToStringKeysRecursive(yamlData, "")
			if err != nil {
				panic(err)
			}
			templateContent, err := json.Marshal(jsonData)
			if err != nil {
				panic(err)
			}
			fmt.Println(string(templateContent))
		} else {
			fmt.Println("  Source:", workflow.definition.Source)
		}
	}
}
