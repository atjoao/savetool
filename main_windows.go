package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"savetool/config"
	"savetool/services/catbox"
	"strings"
	"syscall"
)

func main() {
	executable := flag.String("executable", "", "Path to the executable + arguments")
	saves := flag.String("saves", "", "Path to the saves folder")
	service := flag.String("service", "", "Service name")
	catboxPtr := flag.String("catbox", "", "Catbox configuration || mandatory depending on the service chosen")

	flag.Parse()

	if *executable == "" {
		fmt.Println("Executable path is required")
		os.Exit(1)
	}

	if *saves == "" {
		fmt.Println("Saves path is required")
		os.Exit(1)
	}

	if *service == "" {
		fmt.Println("Service is required")
		os.Exit(1)
	}

	fmt.Println("Executable:", *executable)

	switch *service {
	case "catbox":
		{
			config := config.CatboxConfig{}
			fmt.Println("Service:", *service)
			catboxConfig := strings.Split(*catboxPtr, "+")
			if len(catboxConfig) != 2 {
				fmt.Println("Invalid catbox configuration")
				fmt.Println("Example configuration: -catbox=userhash+albumId")
				return
			}

			config.Userhash = catboxConfig[0]
			config.AlbumID = catboxConfig[1]
			config.SavePath = *saves
			// 0 = error
			// 1 = new files
			catbox.Retrieve(&config)
		}
	default:
		{
			fmt.Println("Service not supported:", *service)
			fmt.Println("Supported services: catbox")
		}
	}

	env := os.Environ()

	ext := filepath.Ext(*executable)
	if ext != ".exe" && ext != ".lnk" {
		fmt.Println("Error: executable path must end with .exe or .lnk")
		os.Exit(1)
	}

	exeIndex := strings.LastIndex(*executable, ext) + len(ext)
	executablePath := (*executable)[:exeIndex]
	args := strings.Fields((*executable)[exeIndex:])

	_, handler, err := syscall.StartProcess(executablePath, args, &syscall.ProcAttr{
		Env: env,
		Files: []uintptr{
			uintptr(syscall.Stdin),
			uintptr(syscall.Stdout),
			uintptr(syscall.Stderr),
		},
	})

	if err != nil {
		fmt.Println("Error starting process:", err)
	}

	_, err = syscall.WaitForSingleObject(syscall.Handle(handler), syscall.INFINITE)
	if err != nil {
		fmt.Println("Error waiting for process to exit:", err)
	}

	// compress and then upload
	switch *service {
	case "catbox":
		{
			catbox.CompressAndUpload()
			fmt.Println("Done")
			catbox.UploadLastFile("true")
		}
	default:
		{
			fmt.Println("Service not supported:", *service)
		}
	}

}
