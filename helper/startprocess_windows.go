//go:build windows

package helper

import (
	"fmt"
	"os"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type JOBOBJECT_BASIC_ACCOUNTING_INFORMATION struct {
	TotalUserTime             int64
	TotalKernelTime           int64
	ThisPeriodTotalUserTime   int64
	ThisPeriodTotalKernelTime int64
	TotalPageFaultCount       uint32
	TotalProcesses            uint32
	ActiveProcesses           uint32
	TotalTerminatedProcesses  uint32
}

var (
	modkernel32                   = windows.NewLazySystemDLL("kernel32.dll")
	procQueryInformationJobObject = modkernel32.NewProc("QueryInformationJobObject")
)

const (
	JobObjectBasicAccountingInformation = 1
)

func queryJobAccountingInfo(job windows.Handle) (JOBOBJECT_BASIC_ACCOUNTING_INFORMATION, error) {
	var info JOBOBJECT_BASIC_ACCOUNTING_INFORMATION
	var returnLength uint32
	ret, _, err := procQueryInformationJobObject.Call(
		uintptr(job),
		uintptr(JobObjectBasicAccountingInformation),
		uintptr(unsafe.Pointer(&info)),
		uintptr(unsafe.Sizeof(info)),
		uintptr(unsafe.Pointer(&returnLength)),
	)
	if ret == 0 {
		return info, err
	}
	return info, nil
}

func StartProcess(executablePath string, args []string) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		fmt.Println("Failed to create job object, falling back to simple wait:", err)
		startProcessFallback(executablePath, args)
		return
	}
	defer windows.CloseHandle(job)

	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = 0
	_, err = windows.SetInformationJobObject(
		job,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	)
	if err != nil {
		fmt.Println("Failed to configure job object, falling back to simple wait:", err)
		startProcessFallback(executablePath, args)
		return
	}

	argv0, _ := windows.UTF16PtrFromString(executablePath)

	cmdLine := windows.ComposeCommandLine(append([]string{executablePath}, args...))
	cmdLinePtr, _ := windows.UTF16PtrFromString(cmdLine)

	var si windows.StartupInfo
	si.Cb = uint32(unsafe.Sizeof(si))
	var pi windows.ProcessInformation

	err = windows.CreateProcess(
		argv0,
		cmdLinePtr,
		nil,
		nil,
		true,
		windows.CREATE_SUSPENDED,
		nil,
		nil,
		&si,
		&pi,
	)
	if err != nil {
		fmt.Println("Error starting process:", err)
		os.Exit(1)
	}
	defer windows.CloseHandle(pi.Process)
	defer windows.CloseHandle(pi.Thread)

	err = windows.AssignProcessToJobObject(job, pi.Process)
	if err != nil {
		fmt.Println("Warning: failed to assign process to job object:", err)
		fmt.Println("Falling back to simple wait on the initial process")
		windows.ResumeThread(pi.Thread)
		windows.WaitForSingleObject(pi.Process, windows.INFINITE)
		return
	}

	windows.ResumeThread(pi.Thread)
	fmt.Println("Process started and assigned to job object, PID:", pi.ProcessId)

	windows.WaitForSingleObject(pi.Process, windows.INFINITE)
	fmt.Println("Initial process exited")

	for {
		accounting, err := queryJobAccountingInfo(job)
		if err != nil {
			fmt.Println("Error querying job object:", err)
			break
		}

		fmt.Printf("Job: active=%d, total=%d, terminated=%d\n",
			accounting.ActiveProcesses,
			accounting.TotalProcesses,
			accounting.TotalTerminatedProcesses)

		if accounting.ActiveProcesses == 0 {
			fmt.Println("All processes in job have exited")
			break
		}

		time.Sleep(2 * time.Second)
	}
}

func startProcessFallback(executablePath string, args []string) {
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
