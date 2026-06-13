//go:build !windows

package helper

import (
	"fmt"
	"os"
)

func StartProcess(executablePath string, args []string) {
	env := os.Environ()
	proc, err := os.StartProcess(executablePath, append([]string{executablePath}, args...), &os.ProcAttr{
		Env: env,
		Files: []*os.File{
			os.Stdin,
			os.Stdout,
			os.Stderr,
		},
	})

	if err != nil {
		fmt.Println("Error starting process:", err)
		os.Exit(1)
	}

	proc.Wait()
}
