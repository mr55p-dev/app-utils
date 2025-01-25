package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"path/filepath"

	"embed"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/mr55p-dev/app-utils/lib/compose"
	"github.com/mr55p-dev/app-utils/lib/manager"
	"github.com/mr55p-dev/app-utils/lib/nginx"
)

var AppsDir = flag.String("apps", "/etc/gold/apps", "Path to apps directory")
var NginxDir = flag.String("nginx", "/etc/nginx/sites-enabled", "Path to nginx dir")

//go:embed html/*
var embeddedFS embed.FS
var templateFS, _ = fs.Sub(embeddedFS, "html")

type Template struct {
	templates map[string]*template.Template
	base      *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates[name].ExecuteTemplate(w, "layout.html", data)
}

func (t *Template) LoadPage(names ...string) {
	for _, name := range names {
		base := template.Must(t.base.Clone())
		parsed := template.Must(base.ParseFS(templateFS, name))
		n := filepath.Base(name)
		t.templates[n] = parsed
	}
}

func NewTemplates(layout string, components ...string) *Template {
	templateMap := make(map[string]*template.Template)
	arg := []string{layout}
	arg = append(arg, components...)
	return &Template{
		templates: templateMap,
		base:      template.Must(template.ParseFS(templateFS, arg...)),
	}
}

func main() {
	t := NewTemplates(
		"layout.html",
		"components/containersTable.html",
	)
	t.LoadPage(
		"views/list.html",
		"views/app.html",
		"views/create.html",
		"views/extensions.html",
	)

	e := echo.New()
	e.Renderer = t

	apps, err := manager.New(*AppsDir)
	if err != nil {
		panic(err)
	}

	compose, err := compose.New(*AppsDir)
	if err != nil {
		panic(err)
	}

	nginx := nginx.New(*NginxDir)

	e.Logger.SetLevel(log.DEBUG)
	e.Use(middleware.Recover(), middleware.Logger())
	e.GET("/", func(c echo.Context) error {
		stacks, err := apps.List()
		if err != nil {
			slog.Error("Failed to get stacks", "error", err)
			return c.String(http.StatusInternalServerError, "Failed to get stacks")
		}
		return c.Render(http.StatusOK, "list.html", stacks)
	})
	e.GET("/create", func(c echo.Context) error {
		return c.Render(http.StatusOK, "create.html", nil)
	})
	e.GET("/extensions", func(c echo.Context) error {
		extensions, err := apps.Extensions()
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
	})
	e.GET("/app/:id", func(c echo.Context) error {
		id := c.Param("id")
		info, err := apps.Get(id)
		if err != nil {
			return c.String(http.StatusBadRequest, "Failed to load stack")
		}

		d := map[string]any{
			"Name":        info.ID,
			"Path":        info.Path,
			"AppYaml":     info.AppYaml,
			"RawAppYaml":  string(info.RawAppYaml),
			"PortainerId": info.PortainerId,
			"NginxStatus": nginx.Status(info.ID),
		}
		containers, err := compose.Ps(id)
		if err == nil {
			d["Containers"] = containers
		}
		return c.Render(http.StatusOK, "app.html", d)
	})


	e.POST("/app/:id/nginx/enable", func(c echo.Context) error {
		id := c.Param("id")
		app, err := apps.Get(id)
		if err != nil {
			return c.String(http.StatusBadRequest, "Invalid app id")
		}
		buf := new(bytes.Buffer)
		for _, block := range app.AppYaml.Nginx {
			err := nginx.CreateUnit(buf, block)
			if err != nil {
				return c.String(
					http.StatusInternalServerError,
					fmt.Sprintf("Failed to write config block %s: %s", block.ExternalHost, err),
				)
			}
			fmt.Fprint(buf, "\n\n")
		}
		err = nginx.InstallUnit(buf, app.ID)
		if err != nil {
			c.Logger().Debug("Error installing unit", err)
			return c.String(http.StatusInternalServerError, "Failed to install unit")
		}
		err = nginx.Reload()
		if err != nil {
			c.Logger().Debug("Error reloading nginx", err)
			return c.String(http.StatusInternalServerError, "Failed to reload nginx")
		}
		return c.String(http.StatusOK, "Success!")
	})

	e.POST("/app/:id/nginx/disable", func(c echo.Context) error {
		id := c.Param("id")
		app, err := apps.Get(id)
		if err != nil {
			return c.String(http.StatusBadRequest, "Invalid app id")
		}
		err = nginx.RemoveUnit(app.ID)
		if err != nil {
			c.Logger().Debug("Error removing unit", err)
			return c.String(http.StatusInternalServerError, "Failed to remove unit")
		}
		err = nginx.Reload()
		if err != nil {
			c.Logger().Debug("Error reloading nginx", err)
			return c.String(http.StatusInternalServerError, "Failed to reload nginx")
		}
		return c.String(http.StatusOK, "Success!")
	})

	if err := e.Start(":8081"); err != nil {
		slog.Error("Failed to start server", "error", err)
	}
}

type StackManager struct{}

type Stack struct {
	Name       string
	Path       string
	Status     string
	Containers []Container
	Nginx      string
}

type Container struct {
	Name   string `json:"Name"`
	Status string `json:"Status"`
}
