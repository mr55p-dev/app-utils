package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log/slog"

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
var host = flag.String("host", "", "Host to listen on")
var port = flag.Int("port", 8080, "Port to listen on")

//go:embed html/*
var embeddedFS embed.FS
var templateFS, _ = fs.Sub(embeddedFS, "html")

func main() {
	flag.Parse()
	t := NewTemplates(
		"layout.html",
		"components/containersTable.html",
		"components/configForm.html",
		"components/composeForm.html",
	)
	t.LoadPage(
		"views/list.html",
		"views/app.html",
		"views/create.html",
		"views/extensions.html",
	)

	e := echo.New()
	e.Renderer = t
	e.Logger.SetLevel(log.DEBUG)
	e.Use(middleware.Recover(), middleware.Logger())

	apps, err := manager.New(*AppsDir)
	if err != nil {
		panic(err)
	}

	compose, err := compose.New(*AppsDir)
	if err != nil {
		panic(err)
	}

	handler := &Handler{
		apps:    apps,
		compose: compose,
		nginx:   nginx.New(*NginxDir),
	}

	e.GET("/", handler.root)
	e.GET("/create", handler.create)
	e.GET("/extensions", handler.extensions)
	e.GET("/app/:id", handler.app)
	e.POST("/app/:id/config", handler.appConfig)
	e.POST("/app/:id/compose", handler.composeConfig)
	e.POST("/app/:id/nginx/enable", handler.nginxEnable)
	e.POST("/app/:id/nginx/disable", handler.nginxDisable)

	if err := e.Start(fmt.Sprintf("%s:%d", *host, *port)); err != nil {
		slog.Error("Failed to start server", "error", err)
	}
}
