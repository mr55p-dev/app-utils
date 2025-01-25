package manager

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/mr55p-dev/app-utils/config"
	"github.com/mr55p-dev/app-utils/lib/portainer"
)

type FSClient struct {
	dir string
}

type App struct {
	ID        string
	Path        string
	ComposeFile []byte
	EnvFile     []byte
	PortainerId string
	AppYaml     *config.AppConfig
	RawAppYaml  []byte
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
			// path := filepath.Join(cli.dir, dir.Name())
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

	composeFile, err := os.ReadFile(filepath.Join(app.Path, "compose.yml"))
	if err == nil {
		app.ComposeFile = composeFile
	}

	envFile, err := os.ReadFile(filepath.Join(app.Path, "stack.env"))
	if err == nil {
		app.EnvFile = envFile
	}

	portainerId, err := portainer.GetStackId(app.Path)
	if err == nil {
		app.PortainerId = strconv.Itoa(portainerId)
	}

	rawConfig, err := os.ReadFile(filepath.Join(app.Path, "app.yml"))
	if err == nil {
		app.RawAppYaml = rawConfig
	}

	conf, err := config.NewAppConfig(app.Path)
	if err == nil {
		app.AppYaml = conf
	}

	return app, nil
}

func (cli *FSClient) Update(info *App) error {
	return nil
}

func (cli *FSClient) Delete(name string) error {
	return nil
}
