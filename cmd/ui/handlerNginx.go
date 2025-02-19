package main

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/mr55p-dev/app-utils/lib/manager"
)

func (h *Handler) nginxEnable(c echo.Context) error {
	app := c.Get("app").(*manager.App)
	err := h.nginx.CreateAndInstallUnits(app.ID, app.AppYaml.Nginx)
	if err != nil {
		c.Logger().Debug("Failed to crate unit", err)
		return c.String(http.StatusOK, "Failed to create unit")
	}
	return c.String(http.StatusOK, "Success!")
}

func (h *Handler) nginxDisable(c echo.Context) error {
	app := c.Get("app").(*manager.App)
	err := h.nginx.RemoveUnit(app.ID)
	if err != nil {
		c.Logger().Debug("Error removing unit", err)
		return c.String(http.StatusOK, "Failed to remove unit")
	}
	return c.String(http.StatusOK, "Success!")
}

func (h *Handler) nginxReload(c echo.Context) error {
	if err := h.nginx.Reload(); err != nil {
		c.Logger().Debug("Failed to reload nginx", err)
		return c.String(http.StatusOK, fmt.Sprintf("Failed to reload nginx: %s", err))
	}

	return c.String(http.StatusOK, "Reloaded nginx!")
}
