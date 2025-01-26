package main

import (
	"html/template"
	"io"
	"path/filepath"

	"github.com/labstack/echo/v4"
)

type Template struct {
	templates map[string]*template.Template
	base      *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates[name].Execute(w, data)
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
	for _, component := range components {
		name := filepath.Base(component)
		templateMap[name] = template.Must(
			template.ParseFS(templateFS, component),
		)
	}

	arg := []string{layout}
	arg = append(arg, components...)
	return &Template{
		templates: templateMap,
		base:      template.Must(template.ParseFS(templateFS, arg...)),
	}
}
