package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jbrunton/gflows/io/pkg"

	"github.com/jbrunton/gflows/config"
	"github.com/jbrunton/gflows/env"
	"github.com/jbrunton/gflows/io/content"
	"github.com/jbrunton/gflows/workflow"
	"github.com/jbrunton/gflows/yamlutil"
	cmdcore "github.com/k14s/ytt/pkg/cmd/core"
	cmdtpl "github.com/k14s/ytt/pkg/cmd/template"
	"github.com/k14s/ytt/pkg/files"
	"github.com/k14s/ytt/pkg/workspace"
	"github.com/spf13/afero"
	"github.com/thoas/go-funk"
)

type YttTemplateEngine struct {
	fs            *afero.Afero
	context       *config.GFlowsContext
	contentWriter *content.Writer
	env           *env.GFlowsEnv
}

func NewYttTemplateEngine(fs *afero.Afero, context *config.GFlowsContext, contentWriter *content.Writer, env *env.GFlowsEnv) *YttTemplateEngine {
	return &YttTemplateEngine{
		fs:            fs,
		context:       context,
		contentWriter: contentWriter,
		env:           env,
	}
}

func (engine *YttTemplateEngine) GetObservableSourcesInDir(dir string) []string {
	files := []string{}
	err := engine.fs.Walk(dir, func(path string, f os.FileInfo, err error) error {
		// TODO: should this check apply to all package workflow dirs? (or is it even still needed?)
		if filepath.Dir(path) == engine.context.WorkflowsDir() {
			// ignore files in the top level workflows dir, as we need them to be in a nested directory to infer the template name
			return nil
		}
		ext := filepath.Ext(path)
		if ext == ".yml" || ext == ".yaml" || ext == ".txt" {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		panic(err)
	}

	return files
}

func (engine *YttTemplateEngine) GetObservableSources() []string {
	return engine.GetObservableSourcesInDir(engine.context.WorkflowsDir())
}

func (engine *YttTemplateEngine) GetWorkflowTemplates() []*pkg.PathInfo {
	templates := []*pkg.PathInfo{}
	packages, err := engine.env.GetPackages()
	if err != nil {
		panic(err)
	}
	for _, pkg := range packages {
		paths, err := afero.Glob(engine.fs, filepath.Join(pkg.WorkflowsDir(), "/*"))
		if err != nil {
			panic(err)
		}
		for _, path := range paths {
			isDir, err := engine.fs.IsDir(path)
			if err != nil {
				panic(err)
			}
			if !isDir || engine.isLib(path) {
				continue
			}
			sources := engine.GetObservableSourcesInDir(path)
			if len(sources) > 0 {
				// only add directories with genuine source files
				pathInfo, err := pkg.GetPathInfo(path)
				if err != nil {
					panic(err)
				}
				templates = append(templates, pathInfo)
			}
		}
	}
	return templates
}

type FileSource struct {
	fs   *afero.Afero
	path string
	dir  string
}

func NewFileSource(fs *afero.Afero, path, dir string) FileSource { return FileSource{fs, path, dir} }

func (s FileSource) Description() string { return fmt.Sprintf("file '%s'", s.path) }

func (s FileSource) RelativePath() (string, error) {
	if s.dir == "" {
		return filepath.Base(s.path), nil
	}

	cleanPath, err := filepath.Abs(filepath.Clean(s.path))
	if err != nil {
		return "", err
	}

	cleanDir, err := filepath.Abs(filepath.Clean(s.dir))
	if err != nil {
		return "", err
	}

	if strings.HasPrefix(cleanPath, cleanDir) {
		result := strings.TrimPrefix(cleanPath, cleanDir)
		result = strings.TrimPrefix(result, string(os.PathSeparator))
		return result, nil
	}

	return "", fmt.Errorf("unknown relative path for %s", s.path)
}

func (s FileSource) Bytes() ([]byte, error) { return s.fs.ReadFile(s.path) }

func (engine *YttTemplateEngine) getInput(workflowName string, templateDir string) (*cmdtpl.TemplateInput, error) {
	var in cmdtpl.TemplateInput
	for _, sourcePath := range engine.GetObservableSourcesInDir(templateDir) {
		source := NewFileSource(engine.fs, sourcePath, filepath.Dir(sourcePath))
		file, err := files.NewFileFromSource(source)
		if err != nil {
			panic(err)
		}
		in.Files = append(in.Files, file)
	}
	paths, err := engine.getYttLibs(workflowName)
	if err != nil {
		return nil, err
	}
	libs, err := files.NewSortedFilesFromPaths(paths, files.SymlinkAllowOpts{})
	if err != nil {
		return nil, err
	}
	in.Files = append(in.Files, libs...)
	return &in, nil
}

func (engine *YttTemplateEngine) apply(workflowName string, templateDir string) (string, error) {
	ui := cmdcore.NewPlainUI(false)
	in, err := engine.getInput(workflowName, templateDir)
	if err != nil {
		return "", err
	}
	rootLibrary := workspace.NewRootLibrary(in.Files)

	libraryExecutionFactory := workspace.NewLibraryExecutionFactory(ui, workspace.TemplateLoaderOpts{
		IgnoreUnknownComments: true,
		StrictYAML:            false,
	})

	libraryCtx := workspace.LibraryExecutionContext{Current: rootLibrary, Root: rootLibrary}
	libraryLoader := libraryExecutionFactory.New(libraryCtx)

	values, libraryValues, err := libraryLoader.Values([]*workspace.DataValues{})
	if err != nil {
		return "", err
	}

	result, err := libraryLoader.Eval(values, libraryValues)
	if err != nil {
		return "", err
	}

	workflowContent := ""

	for _, file := range result.Files {
		workflowContent = workflowContent + string(file.Bytes())
	}

	return workflowContent, nil
}

// GetWorkflowDefinitions - get workflow definitions for the given context
func (engine *YttTemplateEngine) GetWorkflowDefinitions() ([]*workflow.Definition, error) {
	templates := engine.GetWorkflowTemplates()
	definitions := []*workflow.Definition{}
	for _, template := range templates {
		workflowName := filepath.Base(template.LocalPath)
		destinationPath := filepath.Join(engine.context.GitHubDir, "workflows/", workflowName+".yml")
		definition := &workflow.Definition{
			Name:        workflowName,
			Source:      template.LocalPath,
			Destination: destinationPath,
			Status:      workflow.ValidationResult{Valid: true},
		}

		workflow, err := engine.apply(workflowName, template.LocalPath)

		if err != nil {
			definition.Status.Valid = false
			definition.Status.Errors = []string{strings.Trim(err.Error(), " \n\r")}
		} else {
			definition.SetContent(workflow, template)
		}

		definitions = append(definitions, definition)
	}

	return definitions, nil
}

func (engine *YttTemplateEngine) ImportWorkflow(workflow *workflow.GitHubWorkflow) (string, error) {
	workflowContent, err := engine.fs.ReadFile(workflow.Path)
	if err != nil {
		return "", err
	}

	templateContent, err := yamlutil.NormalizeWorkflow(string(workflowContent))
	if err != nil {
		return "", err
	}

	_, filename := filepath.Split(workflow.Path)
	templateName := strings.TrimSuffix(filename, filepath.Ext(filename))
	templatePath := filepath.Join(engine.context.WorkflowsDir(), templateName, templateName+".yml")
	engine.contentWriter.SafelyWriteFile(templatePath, string(templateContent))

	return templatePath, nil
}

func (engine *YttTemplateEngine) WorkflowGenerator(templateVars map[string]string) content.WorkflowGenerator {
	return content.WorkflowGenerator{
		Name:         "gflows",
		TemplateVars: templateVars,
		Sources: []content.WorkflowSource{
			content.NewWorkflowSource("/ytt/workflows/common/steps.lib.yml", "/workflows/common/steps.lib.yml"),
			content.NewWorkflowSource("/ytt/workflows/common/workflows.lib.yml", "/workflows/common/workflows.lib.yml"),
			content.NewWorkflowSource("/ytt/workflows/common/values.yml", "/workflows/common/values.yml"),
			content.NewWorkflowSource("/ytt/workflows/gflows/gflows.yml", "/workflows/$WORKFLOW_NAME/$WORKFLOW_NAME.yml"),
			content.NewWorkflowSource("/ytt/config.yml", "/config.yml"),
		},
	}
}

func (engine *YttTemplateEngine) getWorkflowName(workflowsDir string, filename string) string {
	_, templateFileName := filepath.Split(filename)
	return strings.TrimSuffix(templateFileName, filepath.Ext(templateFileName))
}

func (engine *YttTemplateEngine) getAllYttLibs() []string {
	libs := engine.context.Config.Templates.Defaults.Libs

	for _, override := range engine.context.Config.Templates.Overrides {
		libs = append(libs, override.Libs...)
	}

	return engine.context.ResolvePaths(libs)
}

func (engine *YttTemplateEngine) getYttLibs(workflowName string) ([]string, error) {
	var paths []string
	for _, path := range engine.context.Config.GetTemplateLibs(workflowName) {
		if strings.HasSuffix(path, ".gflowslib") {
			lib, err := engine.env.LoadLib(path)
			if err != nil {
				return []string{}, err
			}
			paths = append(paths, lib.LibsDir())
		} else {
			paths = append(paths, path)
		}
	}
	return engine.context.ResolvePaths(paths), nil
}

func (engine *YttTemplateEngine) isLib(path string) bool {
	_, isLib := funk.FindString(engine.getAllYttLibs(), func(lib string) bool {
		return filepath.Clean(lib) == filepath.Clean(path)
	})
	return isLib
}
