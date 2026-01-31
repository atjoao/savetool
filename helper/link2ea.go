//go:build !windows

package helper

import "net/url"

func ParseLinkToEA(url *url.URL) (executablePath string, args []string) {
	return "", nil
}
