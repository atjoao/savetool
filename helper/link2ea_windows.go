package helper

import (
	"fmt"
	"net/url"
	"os"
	"regexp"

	"golang.org/x/sys/windows/registry"
)

func ParseLinkToEA(url *url.URL) (executablePath string, args []string) {
	k, err := registry.OpenKey(registry.CLASSES_ROOT, "link2ea\\shell\\open\\command", registry.QUERY_VALUE)
	if err != nil {
		fmt.Println("Error opening registry key:", err)
		os.Exit(1)
	}
	defer k.Close()
	executable, _, err := k.GetStringValue("")
	if err != nil {
		fmt.Println("Error reading registry value:", err)
		os.Exit(1)
	}

	regexReplace := regexp.MustCompile(`^"?(.*?\.exe)"?.*$`)
	match := regexReplace.FindStringSubmatch(executable)
	if len(match) > 1 {
		executable = match[1]
	}
	fmt.Println("Resolved Link2EA executable:", executable)

	executablePath = executable
	args = append([]string{url.String()}, args...)

	return executablePath, args
}
