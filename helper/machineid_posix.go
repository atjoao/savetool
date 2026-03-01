//go:build !windows

package helper

import (
	"fmt"
	"os"
	"strings"
)

func getMachineID() (string, error) {
	content, err := os.ReadFile("/etc/machine-id")

	if err != nil {
		content, err = os.ReadFile("/var/lib/dbus/machine-id")
		if err != nil {
			fmt.Println("Could not get unique indentifier, resorting to hostname")
			str, err := os.Hostname()
			return str, err
		}
	}

	cleaned := strings.TrimSpace(strings.ReplaceAll(strings.ToLower(string(content)), "-", ""))
	if len(cleaned) > 8 {
		cleaned = cleaned[:8]
	}

	return cleaned, nil
}
