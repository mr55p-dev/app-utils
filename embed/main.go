package embed

import (
	_ "embed"
)

//go:embed templates/nginx.conf.tmpl
var NginxTemplate string

//go:embed env-extensions.yml
var Extensions []byte
