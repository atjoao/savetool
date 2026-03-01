package helper

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func getMachineID() (string, error) {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Cryptography`, registry.QUERY_VALUE)
	if err != nil {
		fmt.Println("Could not get unique indentifier, resorting to hostname")
		str, err := os.Hostname()
		return str, err
	}
	defer key.Close()

	machineGUID, _, err := key.GetStringValue("MachineGuid")
	if err != nil {
		return "", fmt.Errorf("error reading MachineGuid: %w", err)
	}

	cleaned := strings.TrimSpace(strings.ReplaceAll(strings.ToLower(machineGUID), "-", ""))
	if len(cleaned) > 8 {
		cleaned = cleaned[:8]
	}
	return cleaned, nil
}
