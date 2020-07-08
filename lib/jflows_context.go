package lib

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// JFlowsContext - current command context
type JFlowsContext struct {
	Dir        string
	ConfigPath string
	GitHubDir  string
	Config     *JFlowsConfig
}

var contextCache map[*cobra.Command]*JFlowsContext

// NewContext - returns a context for the given config
func NewContext(fs *afero.Afero, configPath string, dryRun bool) (*JFlowsContext, error) {
	contextDir := filepath.Dir(configPath)

	config, err := GetContextConfig(fs, configPath)
	if err != nil {
		return nil, err
	}

	githubDir := config.Workflows.GitHubDir
	if githubDir == "" {
		githubDir = ".github/"
	}
	if !filepath.IsAbs(githubDir) {
		githubDir = filepath.Join(filepath.Dir(contextDir), githubDir)
	}

	context := &JFlowsContext{
		Config:     config,
		ConfigPath: configPath,
		GitHubDir:  githubDir,
		Dir:        contextDir,
	}

	return context, nil
}

// GetContext - returns the current command context
func GetContext(fs *afero.Afero, cmd *cobra.Command) (*JFlowsContext, error) {
	context := contextCache[cmd]
	if context != nil {
		return context, nil
	}

	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		panic(err)
	}

	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		panic(err)
	}

	if configPath == "" {
		configPath = os.Getenv("JFLOWS_CONFIG")
	}
	if configPath == "" {
		configPath = ".jflow/config.yml"
	}

	return NewContext(fs, configPath, dryRun)
}

// LoadServiceManifest - finds and returns the JFlowsService for the given service
func (context *JFlowsContext) LoadServiceManifest(name string) (JFlowsService, error) {
	serviceContext := context.Config.Services[name]

	data, err := ioutil.ReadFile(serviceContext.Manifest)

	if err != nil {
		return JFlowsService{}, err
	}

	service := JFlowsService{}
	err = yaml.Unmarshal(data, &service)
	if err != nil {
		panic(err)
	}

	return service, nil
}

func init() {
	contextCache = make(map[*cobra.Command]*JFlowsContext)
}
