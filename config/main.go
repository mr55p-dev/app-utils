package config

type NginxBlock struct {
	ExternalHost string
	Protocol     string `config:"protocol,optional"`
	IPv4         string `config:"ipv4"`
	Port         int
	Protected    bool `config:"protected,optional"`
}

type AppConfig struct {
	App     string
	Nginx   []NginxBlock `config:"nginx,optional"`
	Runtime struct {
		EnvExtensions []string       `config:"env-extensions,optional"`
		Env           map[string]any `config:"env,optional"`
	} `config:"runtime,optional"`
}
