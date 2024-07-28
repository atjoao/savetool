package main

import (
	"flag"
	"fmt"
	"os"
	"savetool/config"
	"savetool/services/catbox"
	"strings"
)

func main() {
	// Define flags
	saves := flag.String("saves", "", "Path to the saves folder")
	service := flag.String("service", "", "Service name")
	keepSaves := flag.Bool("kp", true, "Keep saves stored in game directory - game_dir/saves/<timestamp>-<albumId>.zip")
	catboxPtr := flag.String("catbox", "", "Catbox configuration || mandatory depending on the service chosen")

	// Parse flags
	flag.Parse()

	// Get the executable and arguments from os.Args
	executableArgs := parseExecutableArgs(os.Args)

	if executableArgs.executable == "" {
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

	fmt.Println("Executable:", executableArgs.executable)

	switch *service {
	case "catbox":
		handleCatboxService(catboxPtr, saves, keepSaves)
	default:
		fmt.Println("Service not supported:", *service)
		fmt.Println("Supported services: catbox")
		os.Exit(1)
	}

	executablePath, args := executableArgs.executable, executableArgs.args

	fmt.Println("Starting process:", executablePath)
	fmt.Println("Arguments:", args)

	startProcess(executablePath, args)

	// Compress and then upload
	switch *service {
	case "catbox":
		catbox.CompressAndUpload()
		fmt.Println("Done")
		catbox.UploadLastFile("true")
	default:
		fmt.Println("Service not supported:", *service)
	}
}

func handleCatboxService(catboxPtr, saves *string, keepSaves *bool) {
	config := config.CatboxConfig{}
	fmt.Println("Service: catbox")
	catboxConfig := strings.Split(*catboxPtr, "+")
	if len(catboxConfig) != 2 {
		fmt.Println("Invalid catbox configuration")
		fmt.Println("Example configuration: --catbox=userhash+albumId")
		os.Exit(1)
	}

	config.Userhash = catboxConfig[0]
	config.AlbumID = catboxConfig[1]
	config.SavePath = *saves
	config.KeepSaves = *keepSaves
	// 0 = error
	// 1 = new files
	catbox.Retrieve(&config)
}

type executableArgs struct {
	executable string
	args       []string
}

func parseExecutableArgs(args []string) executableArgs {
	var execArgs executableArgs
	for i, arg := range args {
		if arg == "--" {
			if i+1 < len(args) {
				execArgs.executable = args[i+1]
				execArgs.args = args[i+2:]
			}
			return execArgs
		}
	}
	return execArgs
}

func startProcess(executablePath string, args []string) {
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
