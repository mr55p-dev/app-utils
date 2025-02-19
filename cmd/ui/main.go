package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"

	"embed"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/mr55p-dev/app-utils/lib/compose"
	"github.com/mr55p-dev/app-utils/lib/manager"
	"github.com/mr55p-dev/app-utils/lib/nginx"
	"github.com/mr55p-dev/app-utils/lib/portainer"
)

var (
	AppsDir        = flag.String("apps", "/etc/gold/apps", "Path to apps directory")
	NginxDir       = flag.String("nginx", "/etc/nginx/sites-enabled", "Path to nginx dir")
	SSLCertPath    = flag.String("ssl-cert", "", "Path to ssl cert")
	SSLCertKeyPath = flag.String("ssl-key", "", "Path to ssl cert key")
	SSLDHParamPath = flag.String("ssl-dhparam", "", "Path to dhparams.txt file")
	host           = flag.String("host", "", "Host to listen on")
	port           = flag.Int("port", 8080, "Port to listen on")
	logLevel       = flag.Bool("v", false, "Sets verbose mode")
)

//go:embed html/*
var embeddedFS embed.FS
var templateFS, _ = fs.Sub(embeddedFS, "html")

func main() {
	flag.Parse()
	t := NewTemplates(
		"layout.html",
		"components/alert.html",
		"components/composeForm.html",
		"components/configForm.html",
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
	if *logLevel {
		e.Logger.SetLevel(log.DEBUG)
	} else {
		e.Logger.SetLevel(log.INFO)
	}
	e.Use(
		middleware.RemoveTrailingSlash(),
		middleware.Recover(),
		middleware.Logger(),
	)

	apps, err := manager.New(*AppsDir)
	if err != nil {
		panic(err)
	}

	compose, err := compose.New(*AppsDir)
	if err != nil {
		panic(err)
	}

	nginxArgs := []nginx.ConfigFn{nginx.WithDir(*NginxDir)}
	sslEnabled := *SSLCertPath != "" && *SSLCertKeyPath != ""
	if sslEnabled {
		nginxArgs = append(nginxArgs, nginx.WithSSL(*SSLCertPath, *SSLCertKeyPath))
	}
	if sslEnabled && *SSLDHParamPath != "" {
		nginxArgs = append(nginxArgs, nginx.WithDHParams(*SSLDHParamPath))
	}

	handler := &Handler{
		apps:    apps,
		compose: compose,
		nginx:   nginx.New(nginxArgs...),
		portainer: &portainer.Client{
			Scheme:     os.Getenv("PORTAINER_SCHEME"),
			Host:       os.Getenv("PORTAINER_HOST"),
			ApiKey:     os.Getenv("PORTAINER_KEY"),
			EndpointId: os.Getenv("PORTAINER_ENDPOINT_ID"),
		},
	}

	e.GET("", handler.root)
	e.GET("/extensions", handler.extensions)
	e.POST("/server/nginx/reload", handler.nginxReload)

	app := e.Group("/app/:id", func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			id := c.Param("id")
			if id == "" {
				return c.String(http.StatusBadRequest, "App id is required")
			}
			app, err := handler.apps.Get(id)
			if err != nil {
				return c.String(http.StatusNotFound, fmt.Sprintf("App not found: %s", err))
			}

			c.Set("app", app)
			return next(c)
		}
	})
	app.GET("", handler.viewApp)
	app.GET("/create", handler.create)
	app.POST("/create", func(c echo.Context) error { return c.NoContent(http.StatusNotImplemented) })

	// app yaml config
	app.POST("/config", handler.configApp)

	// nginx units
	app.POST("/nginx/enable", handler.nginxEnable)
	app.POST("/nginx/disable", handler.nginxDisable)

	// compose file changes
	app.POST("/compose", handler.configCompose)
	app.POST("/compose/reload", handler.composeRestart)

	// publushing
	app.POST("/portainer", handler.portainerPublish)

	if err := e.Start(fmt.Sprintf("%s:%d", *host, *port)); err != nil {
		slog.Error("Failed to start server", "error", err)
	}
}
