//go:build windows

package helper

import (
	"fmt"
	"os"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

type SHELLEXECUTEINFOW struct {
	CbSize       uint32
	FMask        uint32
	Hwnd         windows.Handle
	LpVerb       *uint16
	LpFile       *uint16
	LpParameters *uint16
	LpDirectory  *uint16
	NShow        int32
	HInstApp     windows.Handle
	LpIDList     uintptr
	LpClass      *uint16
	HkeyClass    windows.Handle
	DwHotKey     uint32
	HIcon        windows.Handle
	HProcess     windows.Handle
}

var (
	modshell32              = windows.NewLazySystemDLL("shell32.dll")
	procShellExecuteExW     = modshell32.NewProc("ShellExecuteExW")
	SEE_MASK_NOCLOSEPROCESS = uint32(0x00000040)
	SW_NORMAL               = int32(1)
)

func shellExecuteEx(pExecInfo *SHELLEXECUTEINFOW) error {
	ret, _, err := procShellExecuteExW.Call(uintptr(unsafe.Pointer(pExecInfo)))
	if ret == 0 {
		return err
	}
	return nil
}

func CheckAndElevate() {
	if windows.GetCurrentProcessToken().IsElevated() {
		return
	}

	executable, err := os.Executable()
	if err != nil {
		fmt.Println("Failed to get executable path:", err)
		return
	}

	var args []string
	for _, arg := range os.Args[1:] {
		if arg != "-elevate" && arg != "--elevate" {
			if strings.Contains(arg, " ") {
				args = append(args, fmt.Sprintf(`"%s"`, arg))
			} else {
				args = append(args, arg)
			}
		}
	}
	argsString := strings.Join(args, " ")

	cwd, _ := os.Getwd()

	verbPtr, _ := windows.UTF16PtrFromString("runas")
	filePtr, _ := windows.UTF16PtrFromString(executable)
	paramsPtr, _ := windows.UTF16PtrFromString(argsString)
	dirPtr, _ := windows.UTF16PtrFromString(cwd)

	sei := SHELLEXECUTEINFOW{
		CbSize:       uint32(unsafe.Sizeof(SHELLEXECUTEINFOW{})),
		FMask:        SEE_MASK_NOCLOSEPROCESS,
		LpVerb:       verbPtr,
		LpFile:       filePtr,
		LpParameters: paramsPtr,
		LpDirectory:  dirPtr,
		NShow:        SW_NORMAL,
	}

	err = shellExecuteEx(&sei)
	if err != nil {
		fmt.Println("Elevation failed:", err)
		os.Exit(1)
	}

	if sei.HProcess != 0 {
		windows.WaitForSingleObject(sei.HProcess, windows.INFINITE)
		var exitCode uint32
		windows.GetExitCodeProcess(sei.HProcess, &exitCode)
		windows.CloseHandle(sei.HProcess)
		os.Exit(int(exitCode))
	}
	os.Exit(0)
}
