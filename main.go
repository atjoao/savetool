package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"savetool/config"
	"savetool/helper"
	"savetool/services/catbox"
	"savetool/services/github"
	"strings"
)

func main() {
	workDir, _ := os.Getwd()
	logFile := filepath.Join(workDir, "savetool.log")

	// Define flags
	saves := flag.String("saves", "", "Path to the saves folder")
	service := flag.String("service", "", "Service name (github / catbox)")
	keepSaves := flag.Bool("kp", true, "Keep saves stored in game directory - game_dir/saves/<timestamp>.zip")
	catboxPtr := flag.String("catbox", "", "Catbox configuration || mandatory depending on the service chosen")
	githubPtr := flag.String("github", "", "GitHub configuration: token+repo+branch (branch optional) || mandatory depending on the service chosen")
	gamePtr := flag.String("game", "", "Game name identifier (required for github)")

	// Parse flags
	flag.Parse()

	// Get the executable and arguments from os.Args.
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

	f, err := os.Create(logFile)
	if err == nil {
		defer f.Close()
		os.Stdout = f
		os.Stderr = f
	}

	fmt.Println("Executable:", executableArgs.executable)

	switch *service {
	case "catbox":
		handleCatboxService(catboxPtr, saves, keepSaves)
	case "github":
		handleGithubService(githubPtr, saves, keepSaves, gamePtr)
	default:
		fmt.Println("Service not supported:", *service)
		fmt.Println("Supported services: catbox, github")
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
	case "github":
		github.CompressAndUpload()
		fmt.Println("Done")
		github.UploadLastFile("true")
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

	catbox.Retrieve(&config)
}

func handleGithubService(githubPtr, saves *string, keepSaves *bool, gamePtr *string) {
	config := config.GitHubConfig{}
	fmt.Println("Service: GitHub")
	githubConfig := strings.Split(*githubPtr, "+")
	if *gamePtr == "" {
		osEnvGameId := os.Getenv("SteamGameId")
		if osEnvGameId == "" {
			osEnvGameId = os.Getenv("SteamAppId")
		}
		if osEnvGameId == "" {
			fmt.Println("Game name is required (-game=\"\")")
			os.Exit(1)
		}
		fmt.Println("Using gameid from environment variable:", osEnvGameId)
		*gamePtr = osEnvGameId
	}
	if len(githubConfig) < 2 || len(githubConfig) > 3 {
		fmt.Println("Invalid GitHub configuration")
		fmt.Println("Example configuration: --github=token+username/repo+branch")
		fmt.Println("Branch is optional, defaults to 'main'")
		os.Exit(1)
	}

	config.Token = githubConfig[0]
	config.Repo = githubConfig[1]
	if len(githubConfig) == 3 {
		config.Branch = githubConfig[2]
	} else {
		config.Branch = "main"
	}
	config.GameName = *gamePtr
	config.SavePath = *saves
	config.KeepSaves = *keepSaves

	github.Retrieve(&config)
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
	url, _ := url.Parse(executablePath)
	if url.Scheme == "link2ea" {
		fmt.Println("Link2EA protocol")
		if runtime.GOOS == "windows" {
			executablePath, args = helper.ParseLinkToEA(url)
			fmt.Println("Resolved Link2EA executable path:", executablePath)
			fmt.Println("Arguments:", args)
		} else {
			fmt.Println("Link2EA protocol is only supported on Windows")
			os.Exit(1)
		}
	}

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
