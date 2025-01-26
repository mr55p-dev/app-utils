package compose

import (
	"bytes"
	"fmt"
	"os/exec"
)

var composeArgs = []string{"compose"}

func command(path string, args ...string) ([]byte, error) {
	outBytes := new(bytes.Buffer)
	errBytes := new(bytes.Buffer)
	a := make([]string, len(args)+len(composeArgs))
	copy(a[:len(composeArgs)], composeArgs)
	copy(a[len(composeArgs):], args)
	cmd := exec.Command("docker", a...)
	cmd.Dir = path
	cmd.Stdout = outBytes
	cmd.Stderr = errBytes
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("Error (%s): %s", err, errBytes.String())
	}
	return outBytes.Bytes(), nil
}

