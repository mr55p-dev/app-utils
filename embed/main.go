package embed

import (
	_ "embed"
)

//go:embed templates/nginx.conf.tmpl
var NginxTemplate string
