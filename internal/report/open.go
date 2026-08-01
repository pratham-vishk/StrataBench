package report

import (
	"os/exec"
)

func openBrowser(name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	return cmd.Start()
}
