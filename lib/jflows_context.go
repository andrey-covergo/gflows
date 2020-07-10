package lib

import (
	"os"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// JFlowsContext - current command context
type JFlowsContext struct {
	Dir          string
	ConfigPath   string
	GitHubDir    string
	WorkflowsDir string
	Config       *JFlowsConfig
}

var contextCache map[*cobra.Command]*JFlowsContext

// NewContext - returns a context for the given config
func NewContext(fs *afero.Afero, configPath string) (*JFlowsContext, error) {
	contextDir := filepath.Dir(configPath)

	config, err := GetContextConfig(fs, configPath)
	if err != nil {
		return nil, err
	}

	githubDir := config.GithubDir
	if githubDir == "" {
		githubDir = ".github/"
	}
	if !filepath.IsAbs(githubDir) {
		githubDir = filepath.Join(filepath.Dir(contextDir), githubDir)
	}

	workflowsDir := filepath.Join(contextDir, "/workflows")

	context := &JFlowsContext{
		Config:       config,
		ConfigPath:   configPath,
		GitHubDir:    githubDir,
		WorkflowsDir: workflowsDir,
		Dir:          contextDir,
	}

	return context, nil
}

// GetContext - returns the current command context
func GetContext(fs *afero.Afero, cmd *cobra.Command) (*JFlowsContext, error) {
	context := contextCache[cmd]
	if context != nil {
		return context, nil
	}

	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		panic(err)
	}

	if configPath == "" {
		configPath = os.Getenv("JFLOWS_CONFIG")
	}
	if configPath == "" {
		configPath = ".jflows/config.yml"
	}

	return NewContext(fs, configPath)
}

func init() {
	contextCache = make(map[*cobra.Command]*JFlowsContext)
}
