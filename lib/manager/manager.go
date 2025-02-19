package manager

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mr55p-dev/app-utils/config"
	"github.com/mr55p-dev/app-utils/lib/portainer"
)

type FSClient struct {
	dir string
}

type App struct {
	ID          string
	Path        string
	ComposeFile []byte
	EnvFile     []byte
	PortainerId int
	AppYaml     *config.AppConfig
	RawAppYaml  []byte
	RawStackEnv []byte
}

func New(directory string) (*FSClient, error) {
	stat, err := os.Stat(directory)
	if err != nil {
		return nil, fmt.Errorf("Invalid root for FSClient: %w", err)
	}
	if !stat.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", directory)
	}
	return &FSClient{dir: directory}, nil
}

func (cli *FSClient) List() ([]string, error) {
	dirs, err := os.ReadDir(cli.dir)
	if err != nil {
		return nil, fmt.Errorf("Error opening base: %w", err)
	}
	dirList := make([]string, 0)
	for _, dir := range dirs {
		if dir.IsDir() {
			dirList = append(dirList, dir.Name())
		}
	}
	return dirList, nil
}

func (cli *FSClient) Extensions() (config.Extensions, error) {
	extFile, err := os.Open(filepath.Join(cli.dir, "env-extensions.yml"))
	if err != nil {
		return nil, fmt.Errorf("Failed to open ext file: %w", err)
	}
	ext, err := config.NewExtensions(extFile)
	if err != nil {
		return nil, fmt.Errorf("Failed to read extensions: %w", err)
	}
	return ext, nil
}

func (cli *FSClient) Get(name string) (*App, error) {
	app := new(App)
	app.ID = name
	app.Path = filepath.Join(cli.dir, name)

	composeFile, err := os.ReadFile(filepath.Join(app.Path, "docker-compose.yml"))
	if err == nil {
		app.ComposeFile = composeFile
	}

	envFile, err := os.ReadFile(filepath.Join(app.Path, "stack.env"))
	if err == nil {
		app.EnvFile = envFile
	}

	portainerId, err := portainer.GetStackId(app.Path)
	if err == nil {
		app.PortainerId = portainerId
	}

	rawConfig, err := os.ReadFile(filepath.Join(app.Path, "app.yml"))
	if err == nil {
		app.RawAppYaml = rawConfig
	}

	rawEnv, err := os.ReadFile(filepath.Join(app.Path, "stack.env"))
	if err == nil {
		app.EnvFile = rawEnv
	}

	conf, err := config.NewFromFile(app.Path)
	if err == nil {
		app.AppYaml = conf
	}

	return app, nil
}

func sanitizeHost(hostname string) string {
	s := hostname
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, " ", "_")
	return strings.ToUpper(s)
}

func doEnvironment(appConfig *config.AppConfig, extensions config.Extensions) (io.Reader, error) {
	stackEnvData := new(bytes.Buffer)
	for _, nginx := range appConfig.Nginx {
		fmt.Fprintf(stackEnvData, "CFG_IPV4_%s=%s\n", sanitizeHost(nginx.ExternalHost), nginx.IPv4)
	}
	for _, extensionName := range appConfig.Runtime.EnvExtensions {
		ext, ok := extensions[extensionName]
		if !ok {
			return nil, fmt.Errorf("Env extension %s: not found", extensionName)
		}
		for key, val := range ext {
			fmt.Fprintf(stackEnvData, "%s=%v\n", key, val)
		}
	}
	for key, val := range appConfig.Runtime.Env {
		fmt.Fprintf(stackEnvData, "%s=%v\n", key, val)
	}

	return stackEnvData, nil
}

func (cli *FSClient) Update(name string, content []byte) error {
	path := filepath.Join(cli.dir, name, "app.yml")
	err := os.WriteFile(path, content, 0o660)
	if err != nil {
		return fmt.Errorf("Failed to write to %s: %w", path, err)
	}

	appConfig, err := config.NewFromBytes(content)
	if err != nil {
		return fmt.Errorf("Failed to load new config: %w", err)
	}

	extensions, err := cli.Extensions()
	if err != nil {
		return fmt.Errorf("Failed to load extensions: %w", err)
	}

	stackEnv, err := doEnvironment(appConfig, extensions)
	stackEnvBytes, err := io.ReadAll(stackEnv)
	if err != nil {
		return fmt.Errorf("Failed to read stackEnv: %w", err)
	}

	err = os.WriteFile(filepath.Join(cli.dir, name, "stack.env"), stackEnvBytes, 0o660)
	if err != nil {
		return fmt.Errorf("Failed to write updated env: %w", err)
	}

	err = os.WriteFile(filepath.Join(cli.dir, name, ".env"), stackEnvBytes, 0o660)
	if err != nil {
		return fmt.Errorf("Failed to write updated env: %w", err)
	}

	return nil
}

func (cli *FSClient) UpdateCompose(name string, content []byte) error {
	path := filepath.Join(cli.dir, name, "docker-compose.yml")
	err := os.WriteFile(path, content, 0o660)
	if err != nil {
		return fmt.Errorf("Failed to write to %s: %w", path, err)
	}
	return nil
}

func (cli *FSClient) Delete(name string) error {
	return nil
}
