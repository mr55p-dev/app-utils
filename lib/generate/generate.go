package generate

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"text/template"

	"github.com/mr55p-dev/app-utils/config"
)

func Nginx(nginxTemplate *template.Template, blocks []config.NginxBlock) (io.Reader, error) {
	buf := new(bytes.Buffer)
	if nginxTemplate == nil {
		return nil, errors.New("Template is nil")
	}

	for _, block := range blocks {
		err := nginxTemplate.Execute(buf, block)
		if err != nil {
			slog.Error(
				"Failed to write nginx config",
				"error", err,
				"host", block.ExternalHost,
			)
			return nil, errors.New("Could not execute on template")
		}
		fmt.Fprint(buf, "\n\n")
	}

	return buf, nil
}

func Environment(appConfig config.AppConfig, extensions config.Extensions) (io.Reader, error) {
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

func sanitizeHost(hostname string) string {
	s := hostname
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, " ", "_")
	return strings.ToUpper(s)
}
