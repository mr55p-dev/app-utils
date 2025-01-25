package nginx

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	_ "embed"

	"github.com/mr55p-dev/app-utils/config"
)

type Client struct {
	dir string
}

type Status string

var (
	StatusUnknown  Status = "Unknown"
	StatusEnabled  Status = "Enabled"
	StatusDisabled Status = "Disabled"
)

//go:embed nginx.conf.tmpl
var tmpl string

var t = template.Must(template.New("nginx.conf.tmpl").Parse(tmpl))

func New(base string) *Client {
	return &Client{dir: base}
}

func (c *Client) List() {

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

func (*Client) CreateUnit(w io.Writer, conf config.NginxBlock) error {
	err := t.Execute(w, conf)
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
	if FileExists(path) {
		return fmt.Errorf("Unit already exists")
	}

	err = os.WriteFile(path, data, 0o660)
	if err != nil {
		return fmt.Errorf("Failed writing: %w", err)
	}
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
