package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"savetool/config"
	"savetool/services/catbox"
	"strings"
)

func main() {
	executable := flag.String("executable", "", "Path to the executable + arguments")
	saves := flag.String("saves", "", "Path to the saves folder")
	service := flag.String("service", "", "Service name")
	keepSaves := flag.Bool("kp", true, "Keep saves stored in game directory - game_dir/saves/<timestamp>-<albumId>.zip")
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
			config.KeepSaves = *keepSaves
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

	var (
		executablePath string
		args           []string
	)

	if runtime.GOOS == "windows" {
		ext := filepath.Ext(*executable)
		if ext != ".exe" && ext != ".lnk" {
			fmt.Println("Error: executable path must end with .exe or .lnk")
			os.Exit(1)
		}

		exeIndex := strings.LastIndex(*executable, ext) + len(ext)
		executablePath = (*executable)[:exeIndex]
		args = strings.Fields((*executable)[exeIndex:])
	} else {
		executablePath = *executable
		args = strings.Fields(*executable)
	}

	fmt.Println("Starting process:", executablePath)
	fmt.Println("Arguments:", args)

	proc, err := os.StartProcess(executablePath, args, &os.ProcAttr{
		Env: env,
		Files: []*os.File{
			os.Stdin,
			os.Stdout,
			os.Stderr,
		},
	})

	if err != nil {
		fmt.Println("Error starting process:", err)
	}

	proc.Wait()

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
