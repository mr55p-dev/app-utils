package nginx

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"text/template"

	_ "embed"

	"github.com/mr55p-dev/app-utils/config"
)

type Status string
type ConfigFn func(*Client)
type Client struct {
	dir            string
	dhParamsPath   string
	sslCertPath    string
	sslCertKeyPath string
	enabledSSL     bool
}

//go:embed nginx.conf.tmpl
var tmpl string

var (
	StatusUnknown  Status = "Unknown"
	StatusEnabled  Status = "Enabled"
	StatusDisabled Status = "Disabled"

	t = template.Must(template.New("nginx.conf.tmpl").Parse(tmpl))
)

func copyStructToMap(data any) map[string]any {
	to := make(map[string]any)
	dataType := reflect.TypeOf(data).Elem()
	dataVal := reflect.ValueOf(data).Elem()
	fields := reflect.VisibleFields(dataType)
	for _, field := range fields {
		to[field.Name] = dataVal.FieldByName(field.Name).Interface()
	}
	return to
}

func WithDir(dir string) ConfigFn {
	return func(c *Client) { c.dir = dir }
}

func WithSSL(certPath, certKeyPath string) ConfigFn {
	return func(c *Client) {
		c.sslCertPath = certPath
		c.sslCertKeyPath = certKeyPath
		c.enabledSSL = true
	}
}

func WithDHParams(paramsPath string) ConfigFn {
	return func(c *Client) { c.dhParamsPath = paramsPath }
}

func New(config ...ConfigFn) *Client {
	cli := &Client{
		dir: "/etc/nginx/sites-enabled",
	}
	for _, fn := range config {
		fn(cli)
	}
	return cli
}

func (c *Client) pathFromName(name string) string {
	return filepath.Join(c.dir, fmt.Sprintf("%s.gold.nginx.conf", name))
}

func FileExists(path string) bool {
	stat, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !stat.IsDir()

}

func (c *Client) Status(name string) Status {
	stat, err := os.Stat(c.pathFromName(name))
	if err != nil {
		if os.IsNotExist(err) {
			return StatusDisabled
		} else {
			return StatusUnknown
		}
	}
	if stat.IsDir() {
		return StatusUnknown
	}
	return StatusEnabled
}

func (c *Client) CreateUnit(w io.Writer, conf config.NginxBlock) error {
	templateData := copyStructToMap(&conf)
	if c.enabledSSL {
		templateData["SSLEnabled"] = true
		templateData["SSLCertPath"] = c.sslCertPath
		templateData["SSLCertKeyPath"] = c.sslCertKeyPath
		if c.dhParamsPath != "" {
			templateData["SSLDHParamPath"] = c.dhParamsPath
		}
	}

	err := t.Execute(w, templateData)
	if err != nil {
		return fmt.Errorf("Failed to exec template: %w", err)
	}
	return nil
}

func (c *Client) Reload() error {
	buf := new(bytes.Buffer)
	cmd := exec.Command("nginx", "-s", "reload")
	cmd.Stdout = buf
	cmd.Stderr = buf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Failed to reload nginx: %s", buf.String())
	}
	return nil
}

func (c *Client) InstallUnit(r io.Reader, name string) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("Failed to read data: %w", err)
	}

	path := c.pathFromName(name)
	err = os.WriteFile(path, data, 0o660)
	if err != nil {
		return fmt.Errorf("Failed writing: %w", err)
	}
	return nil
}

func (c *Client) CreateAndInstallUnits(id string, blocks []config.NginxBlock) error {
	buf := new(bytes.Buffer)
	fmt.Println("Will create units")
	for _, block := range blocks {
		err := c.CreateUnit(buf, block)
		if err != nil {
			return fmt.Errorf("Error creating unit: %w", err)
		}
		fmt.Fprint(buf, "\n\n")
	}
	fmt.Println("Created units, will install")
	err := c.InstallUnit(buf, id)
	if err != nil {
		return fmt.Errorf("Error installing unit: %w", err)
	}
	fmt.Println("Done with installing")
	return nil
}

func (c *Client) RemoveUnit(name string) error {
	path := c.pathFromName(name)
	if !FileExists(path) {
		return errors.New("Unit not found")
	}

	err := os.Remove(path)
	if err != nil {
		return fmt.Errorf("Failed to remove file: %w", err)
	}
	return nil
}
