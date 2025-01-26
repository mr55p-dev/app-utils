package main

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/mr55p-dev/app-utils/lib/compose"
	"github.com/mr55p-dev/app-utils/lib/manager"
	"github.com/mr55p-dev/app-utils/lib/nginx"
	"github.com/mr55p-dev/app-utils/lib/portainer"
)

type Handler struct {
	apps      *manager.FSClient
	compose   *compose.Client
	nginx     *nginx.Client
	portainer *portainer.Client
}

func (h *Handler) root(c echo.Context) error {
	stacks, err := h.apps.List()
	if err != nil {
		c.Logger().Error("Failed to get stacks", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to get stacks")
	}
	return c.Render(http.StatusOK, "list.html", stacks)
}

func (h *Handler) create(c echo.Context) error {
	return c.Render(http.StatusOK, "create.html", nil)
}

func (h *Handler) extensions(c echo.Context) error {
	extensions, err := h.apps.Extensions()
	if err != nil {
		return c.String(http.StatusInternalServerError, "failed to load extensions")
	}
	type env struct {
		Key string
		Val string
	}
	type ext struct {
		Name string
		Vals []env
	}
	vals := make([]ext, len(extensions))
	for key, val := range extensions {
		parsed := make([]env, len(val))
		for k2, v2 := range val {
			parsed = append(parsed, env{k2, v2})
		}
		vals = append(vals, ext{key, parsed})
	}
	return c.Render(http.StatusOK, "extensions.html", vals)
}

func (h *Handler) app(c echo.Context) error {
	id := c.Param("id")
	app, err := h.apps.Get(id)
	if err != nil {
		return c.String(http.StatusBadRequest, "Failed to load stack")
	}

	d := map[string]any{
		"Name":           app.ID,
		"Path":           app.Path,
		"AppYaml":        app.AppYaml,
		"RawAppYaml":     string(app.RawAppYaml),
		"RawComposeYaml": string(app.ComposeFile),
		"PortainerId":    app.PortainerId,
		"NginxStatus":    h.nginx.Status(app.ID),
	}
	containers, err := h.compose.Ps(app.Path)
	if err == nil {
		d["Containers"] = containers
	}
	return c.Render(http.StatusOK, "app.html", d)
}

func (h *Handler) nginxEnable(c echo.Context) error {
	id := c.Param("id")
	app, err := h.apps.Get(id)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app id")
	}

	err = h.nginx.CreateAndInstallUnits(id, app.AppYaml.Nginx)
	if err != nil {
		c.Logger().Debug("Failed to crate unit", err)
		return c.String(http.StatusInternalServerError, "Failed to create unit")
	}

	err = h.nginx.Reload()
	if err != nil {
		c.Logger().Debug("Error reloading nginx", err)
		return c.String(http.StatusInternalServerError, "Failed to reload nginx")
	}
	return c.String(http.StatusOK, "Success!")
}

func (h *Handler) nginxDisable(c echo.Context) error {
	id := c.Param("id")
	app, err := h.apps.Get(id)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app id")
	}
	err = h.nginx.RemoveUnit(app.ID)
	if err != nil {
		c.Logger().Debug("Error removing unit", err)
		return c.String(http.StatusInternalServerError, "Failed to remove unit")
	}
	err = h.nginx.Reload()
	if err != nil {
		c.Logger().Debug("Error reloading nginx", err)
		return c.String(http.StatusInternalServerError, "Failed to reload nginx")
	}
	return c.String(http.StatusOK, "Success!")
}

func (h *Handler) appConfig(c echo.Context) error {
	id := c.Param("id")
	app, err := h.apps.Get(id)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app id")
	}

	appYaml := c.FormValue("app")
	fmt.Printf("string(appYaml): %v\n", string(appYaml))
	err = h.apps.Update(id, []byte(appYaml))
	if err != nil {
		c.Logger().Debug("Could not update yaml", err)
		return c.String(http.StatusInternalServerError, "Could not update")
	}
	app, err = h.apps.Get(id)
	if err != nil {
		c.Logger().Debug("Failed to parse updated config", err)
		return c.String(http.StatusBadRequest, "Failed to parse updated config")
	}

	if app.PortainerId != 0 {
		c.Logger().Info("Updating portainer")
		composeReader := bytes.NewReader(app.ComposeFile)
		env, err := portainer.ReadEnvironment(bytes.NewReader(app.RawStackEnv))
		_, err = h.portainer.UpdateStack(app.PortainerId, composeReader, env)
		if err != nil {
			c.Logger().Debug("Could not update stack", err)
			return c.String(http.StatusInternalServerError, "Could not update stack")
		}
		c.Logger().Info("Stack updated")
	} else {
		c.Logger().Info("Restarting docker compose")
		err = h.compose.Up(app.Path)
		if err != nil {
			c.Logger().Debug("Could not update stack", err)
			return c.String(http.StatusInternalServerError, "Could not update stack")
		}
	}

	if h.nginx.Status(id) == nginx.StatusEnabled {
		c.Logger().Info("Recreating nginx units")
		err = h.nginx.CreateAndInstallUnits(id, app.AppYaml.Nginx)
		if err != nil {
			c.Logger().Debug("Failed to create nginx units", err)
			return c.String(http.StatusInternalServerError, "Failed to update nginx")
		}
		err = h.nginx.Reload()
		if err != nil {
			c.Logger().Debug("Failed to reload nginx", err)
			return c.String(http.StatusInternalServerError, "Failed to reload nginx")
		}
	}

	return c.Render(http.StatusOK, "configForm.html", map[string]any{
		"Name":       app,
		"RawAppYaml": appYaml,
		"Flash":      "Success!",
	})
}

func (h *Handler) composeConfig(c echo.Context) error {
	id := c.Param("id")
	app, err := h.apps.Get(id)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app id")
	}

	composeYaml := c.FormValue("compose")
	err = h.apps.UpdateCompose(id, []byte(composeYaml))
	if err != nil {
		c.Logger().Debug("Could not update yaml", err)
		return c.String(http.StatusInternalServerError, "Could not update")
	}
	app, err = h.apps.Get(id)
	if err != nil {
		c.Logger().Debug("Failed to parse updated config", err)
		return c.String(http.StatusBadRequest, "Failed to parse updated config")
	}

	c.Logger().Info("Restarting docker compose")
	err = h.compose.Up(app.Path)
	if err != nil {
		c.Logger().Debug("Could not update stack", err)
		return c.String(http.StatusInternalServerError, "Could not update stack")
	}

	if h.nginx.Status(id) == nginx.StatusEnabled {
		c.Logger().Info("Recreating nginx units")
		err = h.nginx.CreateAndInstallUnits(id, app.AppYaml.Nginx)
		if err != nil {
			c.Logger().Debug("Failed to create nginx units", err)
			return c.String(http.StatusInternalServerError, "Failed to update nginx")
		}
		err = h.nginx.Reload()
		if err != nil {
			c.Logger().Debug("Failed to reload nginx", err)
			return c.String(http.StatusInternalServerError, "Failed to reload nginx")
		}
	}

	return c.Render(http.StatusOK, "composeForm.html", map[string]any{
		"Name":           app,
		"RawComposeYaml": composeYaml,
		"Flash":          "Success!",
	})
}
