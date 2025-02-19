package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/mr55p-dev/app-utils/lib/compose"
	"github.com/mr55p-dev/app-utils/lib/manager"
	"github.com/mr55p-dev/app-utils/lib/nginx"
	"github.com/mr55p-dev/app-utils/lib/portainer"
	"gopkg.in/yaml.v3"
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
	c.Logger().Debug("Rendering index with apps", "apps", stacks)
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

func (h *Handler) viewApp(c echo.Context) error {
	app := c.Get("app").(*manager.App)
	return c.Render(http.StatusOK, "app.html", map[string]any{
		"Name":           app.ID,
		"Path":           app.Path,
		"AppYaml":        app.AppYaml,
		"RawAppYaml":     string(app.RawAppYaml),
		"RawComposeYaml": string(app.ComposeFile),
		"PortainerId":    app.PortainerId,
		"NginxStatus":    h.nginx.Status(app.ID),
	})
}

func (h *Handler) configApp(c echo.Context) error {
	app := c.Get("app").(*manager.App)
	appYaml := []byte(c.FormValue("app"))

	iface := make(map[string]any)
	err := yaml.Unmarshal(appYaml, &iface)
	if err != nil {
		return c.String(http.StatusOK, "Failed to update resource: invalid yaml")
	}

	err = h.apps.Update(app.ID, appYaml)
	if err != nil {
		c.Logger().Debug("Could not update yaml", err)
		return c.String(http.StatusInternalServerError, "Could not update app")
	}
	c.Logger().Info("Updated yaml content", "app")
	return c.Render(http.StatusOK, "alert.html", map[string]string{
		"Message": "Succesfully updated app.yml",
	})
}

func (h *Handler) configCompose(c echo.Context) error {
	app := c.Get("app").(*manager.App)
	composeYaml := []byte(c.FormValue("compose"))

	iface := make(map[string]any)
	err := yaml.Unmarshal(composeYaml, &iface)
	if err != nil {
		return c.String(http.StatusOK, "Failed to update resource: invalid yaml")
	}

	err = h.apps.UpdateCompose(app.ID, composeYaml)
	if err != nil {
		c.Logger().Debug("Could not update yaml", err)
		return c.String(http.StatusInternalServerError, "Could not update app")
	}
	c.Logger().Info("Updated yaml content", "app", app.ID)
	return c.Render(http.StatusOK, "alert.html", map[string]string{
		"Message": "Succesfully updated docker-compose.yml",
	})
}

func (h *Handler) composeRestart(c echo.Context) error {
	app := c.Get("app").(*manager.App)
	c.Logger().Info("Restarting docker compose")
	err := h.compose.Up(app.Path)
	if err != nil {
		c.Logger().Debug("Could not update stack", err)
		return c.String(http.StatusOK, "Could not update stack")
	}

	return c.String(http.StatusOK, "Succesfully restarted the containers")
}
