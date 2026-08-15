package utils

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// LnkShortcutInfo contains the resolved fields of a Windows .lnk shortcut.
type LnkShortcutInfo struct {
	TargetPath       string
	Arguments        string
	WorkingDirectory string
	WindowStyle      int
}

var (
	CLSID_ShellLink  = windows.GUID{0x00021401, 0x0000, 0x0000, [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
	IID_IShellLinkW  = windows.GUID{0x000214F9, 0x0000, 0x0000, [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
	IID_IPersistFile = windows.GUID{0x0000010b, 0x0000, 0x0000, [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
)

const (
	iShellLink_Release              = 2
	iPersistFile_Release            = 2
	iPersistFile_Load               = 5
	iShellLinkW_GetPath             = 3
	iShellLinkW_GetIDList           = 4
	iShellLinkW_GetWorkingDirectory = 8
	iShellLinkW_GetArguments        = 10
	iShellLinkW_GetShowCmd          = 14
)

const SLGP_RAWPATH = 0x4

type iUnknownPtr = *unsafe.Pointer

func callVTable(punk iUnknownPtr, index int, args ...uintptr) (uintptr, uintptr, error) {
	vtbl := *(**[0x1000]uintptr)(unsafe.Pointer(punk))
	proc := vtbl[index]
	return syscall.SyscallN(proc, append([]uintptr{uintptr(unsafe.Pointer(punk))}, args...)...)
}

func failedHRESULT(hr uintptr) bool {
	return int32(hr) < 0
}

// pidlToPath converts a PIDL (item ID list) to a filesystem path using
// SHGetPathFromIDListW. Used as fallback when GetPath returns empty.
func pidlToPath(pidl uintptr) string {
	if pidl == 0 {
		return ""
	}
	buf := make([]uint16, 4*windows.MAX_PATH)
	ret, _, _ := shGetPathFromIDListW.Call(pidl, uintptr(unsafe.Pointer(&buf[0])))
	if ret == 0 {
		return ""
	}
	return windows.UTF16ToString(buf)
}

func ResolveLnkPath(linkPath string) (LnkShortcutInfo, error) {
	var info LnkShortcutInfo

	linkPath16, err := windows.UTF16PtrFromString(linkPath)
	if err != nil {
		return info, fmt.Errorf("failed to convert lnk path to utf16: %w", err)
	}

	// CoInitializeEx (STA)
	hr, _, _ := coInitializeEx.Call(0, 0x2)
	initialized := false
	switch {
	case hr == 0, hr == 1:
		initialized = true
	case hr == 0x80010106: // RPC_E_CHANGED_MODE
	default:
		return info, fmt.Errorf("CoInitializeEx failed: 0x%X", uint32(hr))
	}
	defer func() {
		if initialized {
			coUninitialize.Call()
		}
	}()

	// CoCreateInstance(IShellLinkW)
	var punk iUnknownPtr
	hr, _, _ = coCreateInstance.Call(
		uintptr(unsafe.Pointer(&CLSID_ShellLink)),
		0,
		1|4,
		uintptr(unsafe.Pointer(&IID_IShellLinkW)),
		uintptr(unsafe.Pointer(&punk)),
	)
	if failedHRESULT(hr) {
		return info, fmt.Errorf("CoCreateInstance failed: 0x%X", uint32(hr))
	}
	defer callVTable(punk, iShellLink_Release)

	// QueryInterface(IPersistFile)
	var ppf iUnknownPtr
	hr, _, _ = callVTable(punk, 0,
		uintptr(unsafe.Pointer(&IID_IPersistFile)),
		uintptr(unsafe.Pointer(&ppf)))
	if failedHRESULT(hr) {
		return info, fmt.Errorf("QueryInterface(IPersistFile) failed: 0x%X", uint32(hr))
	}
	defer callVTable(ppf, iPersistFile_Release)

	// IPersistFile.Load
	hr, _, _ = callVTable(ppf, iPersistFile_Load,
		uintptr(unsafe.Pointer(linkPath16)),
		0x40, // STGM_READ | STGM_SHARE_DENY_NONE
	)
	fmt.Printf("[ResolveLnkPath] Load hr=0x%X\n", uint32(hr))
	if failedHRESULT(hr) {
		return info, fmt.Errorf("IPersistFile.Load failed: 0x%X", uint32(hr))
	}

	const bufSz = 4 * windows.MAX_PATH

	// GetPath with SLGP_RAWPATH
	targetBuf := make([]uint16, bufSz)
	hr, _, _ = callVTable(punk, iShellLinkW_GetPath,
		uintptr(unsafe.Pointer(&targetBuf[0])),
		uintptr(bufSz),
		0,
		SLGP_RAWPATH,
	)
	info.TargetPath = windows.UTF16ToString(targetBuf)
	fmt.Printf("[ResolveLnkPath] GetPath hr=0x%X len=%d path=%q\n", uint32(hr), len(info.TargetPath), info.TargetPath)

	// If GetPath returned empty, try GetIDList → SHGetPathFromIDListW as fallback
	if info.TargetPath == "" {
		var pidl uintptr
		hrID, _, _ := callVTable(punk, iShellLinkW_GetIDList,
			uintptr(unsafe.Pointer(&pidl)))
		fmt.Printf("[ResolveLnkPath] GetIDList hr=0x%X pidl=0x%X\n", uint32(hrID), pidl)
		if pidl != 0 {
			path := pidlToPath(pidl)
			fmt.Printf("[ResolveLnkPath] pidlToPath=%q\n", path)
			if path != "" {
				info.TargetPath = path
			}
			coTaskMemFree.Call(pidl)
		}
	}

	// Arguments
	argBuf := make([]uint16, bufSz)
	hr, _, _ = callVTable(punk, iShellLinkW_GetArguments,
		uintptr(unsafe.Pointer(&argBuf[0])),
		uintptr(bufSz),
	)
	info.Arguments = windows.UTF16ToString(argBuf)
	fmt.Printf("[ResolveLnkPath] GetArguments hr=0x%X args=%q\n", uint32(hr), info.Arguments)

	// WorkingDirectory
	wdBuf := make([]uint16, bufSz)
	hr, _, _ = callVTable(punk, iShellLinkW_GetWorkingDirectory,
		uintptr(unsafe.Pointer(&wdBuf[0])),
		uintptr(bufSz),
	)
	info.WorkingDirectory = windows.UTF16ToString(wdBuf)
	fmt.Printf("[ResolveLnkPath] GetWorkingDirectory hr=0x%X wd=%q\n", uint32(hr), info.WorkingDirectory)

	// ShowCmd
	var showCmd int32
	hr, _, _ = callVTable(punk, iShellLinkW_GetShowCmd,
		uintptr(unsafe.Pointer(&showCmd)),
	)
	info.WindowStyle = int(showCmd)
	fmt.Printf("[ResolveLnkPath] GetShowCmd hr=0x%X showCmd=%d\n", uint32(hr), showCmd)

	return info, nil
}

var (
	ole32                = windows.NewLazySystemDLL("ole32.dll")
	coInitializeEx       = ole32.NewProc("CoInitializeEx")
	coUninitialize       = ole32.NewProc("CoUninitialize")
	coCreateInstance     = ole32.NewProc("CoCreateInstance")
	coTaskMemFree        = ole32.NewProc("CoTaskMemFree")
	shell32              = windows.NewLazySystemDLL("shell32.dll")
	shGetPathFromIDListW = shell32.NewProc("SHGetPathFromIDListW")
)
