package main

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/mr55p-dev/app-utils/lib/manager"
	"github.com/mr55p-dev/app-utils/lib/portainer"
)

func (h *Handler) portainerPublish(c echo.Context) error {
	app := c.Get("app").(*manager.App)

	if app.PortainerId == 0 {
		return c.String(http.StatusNotFound, "Application is not managed via portainer")
	}

	composeReader := bytes.NewReader(app.ComposeFile)
	env, err := portainer.ReadEnvironment(bytes.NewReader(app.EnvFile))
	if err != nil {
		return c.String(http.StatusOK, "Failed to parse env file, check syntax")
	}
	res, err := h.portainer.UpdateStack(app.PortainerId, composeReader, env)
	if err != nil {
		return c.String(http.StatusOK, fmt.Sprintf("Operation failed with message: %s", err))
	}
	return c.String(http.StatusOK, fmt.Sprintf("Operation completed with message: %s", res.Message))

}
