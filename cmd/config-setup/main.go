package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"text/template"

	"github.com/mr55p-dev/app-utils/config"
	"github.com/mr55p-dev/app-utils/embed"
	"github.com/mr55p-dev/app-utils/lib/generate"
)

type Config struct {
	baseDir    string
	appConfig  config.AppConfig
	extensions config.Extensions
	templates  *template.Template
}

var includeNignx = flag.Bool("include-nginx", true, "Generates NGINX config")
var includeEnv = flag.Bool("include-env", true, "Generate environment files")
var extensionsFile = flag.String("extensions", "./env-extensions.yml", "Path to extensions file")

func main() {
	flag.Parse()

	for _, path := range flag.Args() {
		log.Println("Generating configs for", path)
		cfg := MustNewConfig(path)
		log.Println("Found app", cfg.appConfig.App)
		cfg.Exec()
	}
}

func MustNewConfig(path string) *Config {
	// Load the config object
	cfg := new(Config)
	appConfig, err := config.NewFromFile(path)
	if err != nil {
		panic(err)
	}

	extFile, err := os.Open(*extensionsFile)
	if err != nil {
		panic(err)
	}
	defer extFile.Close()
	extensions, err := config.NewExtensions(extFile)
	if err != nil {
		panic(err)
	}

	cfg.appConfig = *appConfig
	cfg.baseDir = path
	cfg.templates = template.Must(template.New("nginx").Parse(embed.NginxTemplate))
	cfg.extensions = extensions
	return cfg
}

func (cfg *Config) doNginx() error {
	outPath := filepath.Join(cfg.baseDir, "nginx.conf")
	nginxTemplate := cfg.templates.Lookup("nginx")
	if nginxTemplate == nil {
		return errors.New("Template is nil")
	}

	nginxData, err := generate.Nginx(nginxTemplate, cfg.appConfig.Nginx)
	if err != nil {
		return fmt.Errorf("Error when executing nginx template: %w", err)
	}
	data, err := io.ReadAll(nginxData)
	if err != nil {
		return fmt.Errorf("Failed to read from nginx source: %w", err)
	}

	err = os.WriteFile(outPath, data, 0o644)
	if err != nil {
		return fmt.Errorf("Failed to write nginx file: %w", err)
	}

	return nil
}

func (cfg *Config) doEnvFile() error {
	envData, err := generate.Environment(cfg.appConfig, cfg.extensions)
	if err != nil {
		return fmt.Errorf("Failed to load env data: %w", err)
	}
	b, err := io.ReadAll(envData)
	if err != nil {
		return fmt.Errorf("Failed to read from environment reader: %w", err)
	}
	err = os.WriteFile(filepath.Join(cfg.baseDir, "stack.env"), b, 0644)
	if err != nil {
		return fmt.Errorf("Failed to write stack.env: %w", err)
	}
	err = os.WriteFile(filepath.Join(cfg.baseDir, ".env"), b, 0644)
	if err != nil {
		return fmt.Errorf("Failed to write .env: %w", err)
	}

	return nil
}

func (cfg *Config) Exec() {
	doMethod(*includeNignx, cfg.doNginx, "nginx")
	doMethod(*includeEnv, cfg.doEnvFile, "Environment files")
}

func doMethod(predicate bool, fn func() error, name string) {
	if predicate {
		log.Println("Starting task", name)
		if err := fn(); err != nil {
			log.Printf("Failed task %s with reason: %s", name, err)
		} else {
			log.Printf("Completed task %s", name)
		}
	} else {
		log.Println("Skipping task", name)
	}
}
