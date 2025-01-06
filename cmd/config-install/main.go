package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"arr-setup/config"

	"github.com/mr55p-dev/gonk"
)

var targetDir = flag.String("target", "/etc/nginx/conf.d", "Destination directory to link to")
var uninstall = flag.Bool("uninstall", false, "Should uninstall the named app")

type Config struct {
	appConfig config.AppConfig
	baseDir   string
}

func main() {
	flag.Parse()
	for _, path := range flag.Args() {
		log.Println("Generating configs for", path)
		cfg := MustNewConfig(path)
		log.Println("Found app", cfg.appConfig.App)
		var err error
		if *uninstall {
			err = cfg.ExecUninstall()
		} else {
			err = cfg.ExecInstall()
		}
		if err != nil {
			panic(err)
		}
	}
}

func MustNewConfig(path string) *Config {
	// Load the config object
	cfg := new(Config)
	cfg.baseDir = path
	loader, err := gonk.NewYamlLoader(filepath.Join(path, "app.yml"))
	if err != nil {
		log.Panicf("Error creating loader: %s", err)
	}
	if err := gonk.LoadConfig(&cfg.appConfig, loader); err != nil {
		log.Panicf("Error loading config file: %s", err)
	}
	return cfg
}

func (cfg *Config) ExecInstall() error {
	src := filepath.Join(cfg.baseDir, "nginx.conf")
	tgt := filepath.Join(*targetDir, fmt.Sprintf("%s.conf", cfg.appConfig.App))
	if _, err := os.Stat(tgt); err == nil {
		err := os.Remove(tgt)
		if err != nil {
			return fmt.Errorf("Failed to unlink existing file in destination: %w", err)
		}
	}
	err := os.Link(src, tgt)
	if err != nil {
		return fmt.Errorf("Failed to link %s to %s: %w", src, tgt, err)
	}
	return nil
}

func (cfg *Config) ExecUninstall() error {
	tgt := filepath.Join(*targetDir, fmt.Sprintf("%s.conf", cfg.appConfig.App))
	if _, err := os.Stat(tgt); err == nil {
		err := os.Remove(tgt)
		if err != nil {
			return fmt.Errorf("Failed to unlink existing file in destination: %w", err)
		}
	}
	return nil
}
