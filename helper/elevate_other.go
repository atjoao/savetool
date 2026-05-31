//go:build !windows

package helper

func CheckAndElevate() {
	// Elevation is only supported on Windows
}
