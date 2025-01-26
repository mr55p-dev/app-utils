package generate

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/mr55p-dev/app-utils/config"
)

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
