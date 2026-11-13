package platform

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	advapi32 = syscall.NewLazyDLL("advapi32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")

	procQueryFullProcessImageName = kernel32.NewProc("QueryFullProcessImageNameW")
	procAttachConsole             = kernel32.NewProc("AttachConsole")
	procAllocConsole              = kernel32.NewProc("AllocConsole")
	procSetConsoleOutputCP        = kernel32.NewProc("SetConsoleOutputCP")
	procRegCreateKeyEx            = advapi32.NewProc("RegCreateKeyExW")
	procRegOpenKeyEx              = advapi32.NewProc("RegOpenKeyExW")
	procRegSetValueEx             = advapi32.NewProc("RegSetValueExW")
	procRegQueryValueEx           = advapi32.NewProc("RegQueryValueExW")
	procRegDeleteValue            = advapi32.NewProc("RegDeleteValueW")
	procRegCloseKey               = advapi32.NewProc("RegCloseKey")
	procShellExecute              = shell32.NewProc("ShellExecuteW")
)

const (
	runKey            = `Software\Microsoft\Windows\CurrentVersion\Run`
	runValue          = "ForeverFriendsCompanion"
	hkeyCurrentUser   = 0x80000001
	hkeyLocalMachine  = 0x80000002
	keySetValue       = 0x0002
	keyQueryValue     = 0x0001
	keyWow6432        = 0x0200 // KEY_WOW64_32KEY: the 32-bit view the game's installer writes to
	regSZ             = 1
	processQueryLimit = 0x1000
	errorFileNotFound = syscall.Errno(2)
	// CreationFlags for a background process with no window of its own.
	createNewProcessGroup = 0x00000200
	detachedProcess       = 0x00000008
	createNoWindow        = 0x08000000
	attachParentProcess   = 0xFFFFFFFF
	swShowNormal          = 1
)

// ConfigDir is %LOCALAPPDATA%\ForeverFriends. Local, not roaming: a roaming profile would copy
// the private key to every PC of the account and give two devices one key.
func ConfigDir() (string, error) {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		return "", errors.New("LOCALAPPDATA is not set")
	}
	return filepath.Join(base, "ForeverFriends"), nil
}

// ProgramDir is where setup keeps its copy of the program, a per-user location that needs no
// administrator rights.
func ProgramDir() string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		return ""
	}
	return filepath.Join(base, "Programs", "ForeverFriends")
}

// CaseFoldPaths reports whether file paths ignore case.
func CaseFoldPaths() bool { return true }

// GameCandidates lists the usual install folders, and the one the Blizzard installer recorded
// in the registry when that key exists (unverified: a failed lookup only means no hint).
func GameCandidates() []string {
	var out []string
	if p := regString(hkeyLocalMachine, `SOFTWARE\Blizzard Entertainment\World of Warcraft`, "InstallPath", keyWow6432); p != "" {
		out = append(out, filepath.Join(p, "_forever_"), p)
	}
	for _, env := range []string{"ProgramFiles(x86)", "ProgramFiles"} {
		if base := os.Getenv(env); base != "" {
			out = append(out, filepath.Join(base, "World of Warcraft", "_forever_"))
		}
	}
	return out
}

func utf16(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }

func regString(root uintptr, sub, name string, view uintptr) string {
	var h syscall.Handle
	r, _, _ := procRegOpenKeyEx.Call(root, uintptr(unsafe.Pointer(utf16(sub))), 0, keyQueryValue|view, uintptr(unsafe.Pointer(&h)))
	if r != 0 {
		return ""
	}
	defer procRegCloseKey.Call(uintptr(h))
	buf := make([]uint16, 1024)
	size := uint32(len(buf) * 2)
	var typ uint32
	r, _, _ = procRegQueryValueEx.Call(uintptr(h), uintptr(unsafe.Pointer(utf16(name))), 0, uintptr(unsafe.Pointer(&typ)), uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if r != 0 || typ != regSZ {
		return ""
	}
	return syscall.UTF16ToString(buf)
}

// ListImages returns the full executable path of every process it may open, through the
// toolhelp snapshot and QueryFullProcessImageNameW: no child process, no PowerShell.
func ListImages() ([]string, error) {
	snap, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer syscall.CloseHandle(snap)
	var e syscall.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	if err := syscall.Process32First(snap, &e); err != nil {
		return nil, err
	}
	var out []string
	for {
		if p := imagePath(e.ProcessID); p != "" {
			out = append(out, p)
		}
		if err := syscall.Process32Next(snap, &e); err != nil {
			return out, nil
		}
	}
}

func imagePath(pid uint32) string {
	h, err := syscall.OpenProcess(processQueryLimit, false, pid)
	if err != nil {
		return ""
	}
	defer syscall.CloseHandle(h)
	buf := make([]uint16, 1024)
	size := uint32(len(buf))
	if r, _, _ := procQueryFullProcessImageName.Call(uintptr(h), 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size))); r == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:size])
}

// OpenURL opens an http(s) address with the shell, as a double-click on a link would.
func OpenURL(url string) error {
	if len(url) < 7 || (url[:7] != "http://" && (len(url) < 8 || url[:8] != "https://")) {
		return errors.New("not a web address")
	}
	r, _, _ := procShellExecute.Call(0, uintptr(unsafe.Pointer(utf16("open"))), uintptr(unsafe.Pointer(utf16(url))), 0, 0, swShowNormal)
	if r <= 32 {
		return errors.New("the browser could not be opened")
	}
	return nil
}

// SetAutostart registers `"exe" run` under HKCU Run, which starts it at this user's logon.
func SetAutostart(exe string) error {
	var h syscall.Handle
	r, _, _ := procRegCreateKeyEx.Call(hkeyCurrentUser, uintptr(unsafe.Pointer(utf16(runKey))), 0, 0, 0, keySetValue, 0, uintptr(unsafe.Pointer(&h)), 0)
	if r != 0 {
		return syscall.Errno(r)
	}
	defer procRegCloseKey.Call(uintptr(h))
	data, _ := syscall.UTF16FromString(`"` + exe + `" run`)
	r, _, _ = procRegSetValueEx.Call(uintptr(h), uintptr(unsafe.Pointer(utf16(runValue))), 0, regSZ, uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)*2))
	if r != 0 {
		return syscall.Errno(r)
	}
	return nil
}

// RemoveAutostart deletes the Run value; a missing value is fine.
func RemoveAutostart() error {
	var h syscall.Handle
	r, _, _ := procRegOpenKeyEx.Call(hkeyCurrentUser, uintptr(unsafe.Pointer(utf16(runKey))), 0, keySetValue, uintptr(unsafe.Pointer(&h)))
	if r != 0 {
		return nil
	}
	defer procRegCloseKey.Call(uintptr(h))
	r, _, _ = procRegDeleteValue.Call(uintptr(h), uintptr(unsafe.Pointer(utf16(runValue))))
	if r != 0 && syscall.Errno(r) != errorFileNotFound {
		return syscall.Errno(r)
	}
	return nil
}

// SpawnDetached starts exe with no window, outliving this process.
func SpawnDetached(exe string, args ...string) error {
	cmd := exec.Command(exe, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNewProcessGroup | detachedProcess | createNoWindow}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

// PrepareConsole gives a program built for the windowed subsystem a console for the commands a
// member types or double-clicks: the parent's when started from a terminal, a new one
// otherwise. The background `run` never calls it, so it never shows a window.
func PrepareConsole() {
	if r, _, _ := procAttachConsole.Call(attachParentProcess); r == 0 {
		if r, _, _ := procAllocConsole.Call(); r == 0 {
			return // already has a console (a console build): keep the standard handles
		}
	}
	procSetConsoleOutputCP.Call(65001)
	if out, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0); err == nil {
		os.Stdout, os.Stderr = out, out
	}
	if in, err := os.OpenFile("CONIN$", os.O_RDONLY, 0); err == nil {
		os.Stdin = in
	}
}
