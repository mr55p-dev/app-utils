package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/mr55p-dev/app-utils/config"
	"github.com/mr55p-dev/app-utils/embed"

	"github.com/mr55p-dev/gonk"
	"gopkg.in/yaml.v3"
)

type Config struct {
	appConfig  config.AppConfig
	templates  *template.Template
	extensions map[string]map[string]any
	baseDir    string
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
	loader, err := gonk.NewYamlLoader(filepath.Join(path, "app.yml"))
	if err != nil {
		log.Panicf("Error creating loader: %s", err)
	}
	if err := gonk.LoadConfig(&cfg.appConfig, loader); err != nil {
		log.Panicf("Error loading config file: %s", err)
	}
	cfg.baseDir = path
	cfg.templates = template.Must(template.New("nginx").Parse(embed.NginxTemplate))
	extData, err := os.ReadFile(*extensionsFile)
	if err != nil {
		log.Panicf("Failed to read extensions file: %s", err)
	}
	if err := yaml.Unmarshal(extData, &cfg.extensions); err != nil {
		log.Println("Failed to load extensions file", err)
	}
	return cfg
}

func (cfg *Config) doNginx() error {
	buf := new(bytes.Buffer)
	outPath := filepath.Join(cfg.baseDir, "nginx.conf")
	nginxTemplate := cfg.templates.Lookup("nginx")
	if nginxTemplate == nil {
		return errors.New("Template is nil")
	}

	for _, block := range cfg.appConfig.Nginx {
		if block.Protocol == "" {
			block.Protocol = "http"
		}
		err := nginxTemplate.Execute(buf, block)
		if err != nil {
			return fmt.Errorf("Error executing nginx template: %s", err)
		}
		fmt.Fprint(buf, "\n\n")
	}

	err := os.WriteFile(outPath, buf.Bytes(), 0o644)
	if err != nil {
		return fmt.Errorf("Failed to write nginx file: %w", err)
	}

	return nil
}

func (cfg *Config) doEnvFile() error {
	stackEnvData := new(bytes.Buffer)
	for _, nginx := range cfg.appConfig.Nginx {
		fmt.Fprintf(stackEnvData, "CFG_IPV4_%s=%s\n", sanitizeHost(nginx.ExternalHost), nginx.IPv4)
	}
	for _, extensionName := range cfg.appConfig.Runtime.EnvExtensions {
		ext, ok := cfg.extensions[extensionName]
		if !ok {
			return fmt.Errorf("Failed to load env extension %s: not found", extensionName)
		}
		for key, val := range ext {
			fmt.Fprintf(stackEnvData, "%s=%v\n", key, val)
		}
	}
	for key, val := range cfg.appConfig.Runtime.Env {
		fmt.Fprintf(stackEnvData, "%s=%v\n", key, val)
	}

	b := stackEnvData.Bytes()
	err := os.WriteFile(filepath.Join(cfg.baseDir, "stack.env"), b, 0644)
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

func sanitizeHost(hostname string) string {
	s := hostname
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, " ", "_")
	return strings.ToUpper(s)
}
