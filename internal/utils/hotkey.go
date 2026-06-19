package utils

import (
	"strings"
	"syscall"
	"time"
)

var (
	user32Hotkey   = syscall.NewLazyDLL("user32.dll")
	procKeybdEvent = user32Hotkey.NewProc("keybd_event")
)

const (
	KEYEVENTF_KEYUP = 0x0002
	VK_LWIN         = 0x5B
	VK_LMENU        = 0xA4
	VK_LSHIFT       = 0xA0
	VK_CONTROL      = 0x11
)

// SendHotkey 发送快捷键到前台窗口
// hotkey 格式如 "Win+Shift+A"
func SendHotkey(hotkey string) {
	keys := strings.Split(strings.ToLower(hotkey), "+")
	var vks []uintptr

	for _, key := range keys {
		switch strings.TrimSpace(key) {
		case "win", "windows", "lwin":
			vks = append(vks, VK_LWIN)
		case "ctrl", "control", "lctrl":
			vks = append(vks, VK_CONTROL)
		case "alt", "lalt", "menu":
			vks = append(vks, VK_LMENU)
		case "shift", "lshift":
			vks = append(vks, VK_LSHIFT)
		case "a":
			vks = append(vks, 0x41)
		case "b":
			vks = append(vks, 0x42)
		case "c":
			vks = append(vks, 0x43)
		case "d":
			vks = append(vks, 0x44)
		case "e":
			vks = append(vks, 0x45)
		case "f":
			vks = append(vks, 0x46)
		case "g":
			vks = append(vks, 0x47)
		case "h":
			vks = append(vks, 0x48)
		case "i":
			vks = append(vks, 0x49)
		case "j":
			vks = append(vks, 0x4A)
		case "k":
			vks = append(vks, 0x4B)
		case "l":
			vks = append(vks, 0x4C)
		case "m":
			vks = append(vks, 0x4D)
		case "n":
			vks = append(vks, 0x4E)
		case "o":
			vks = append(vks, 0x4F)
		case "p":
			vks = append(vks, 0x50)
		case "q":
			vks = append(vks, 0x51)
		case "r":
			vks = append(vks, 0x52)
		case "s":
			vks = append(vks, 0x53)
		case "t":
			vks = append(vks, 0x54)
		case "u":
			vks = append(vks, 0x55)
		case "v":
			vks = append(vks, 0x56)
		case "w":
			vks = append(vks, 0x57)
		case "x":
			vks = append(vks, 0x58)
		case "y":
			vks = append(vks, 0x59)
		case "z":
			vks = append(vks, 0x5A)
		case "f1":
			vks = append(vks, 0x70)
		case "f2":
			vks = append(vks, 0x71)
		case "f3":
			vks = append(vks, 0x72)
		case "f4":
			vks = append(vks, 0x73)
		case "f5":
			vks = append(vks, 0x74)
		case "f6":
			vks = append(vks, 0x75)
		case "f7":
			vks = append(vks, 0x76)
		case "f8":
			vks = append(vks, 0x77)
		case "f9":
			vks = append(vks, 0x78)
		case "f10":
			vks = append(vks, 0x79)
		case "f11":
			vks = append(vks, 0x7A)
		case "f12":
			vks = append(vks, 0x7B)
		case "0", "num0":
			vks = append(vks, 0x30)
		case "1", "num1":
			vks = append(vks, 0x31)
		case "2", "num2":
			vks = append(vks, 0x32)
		case "3", "num3":
			vks = append(vks, 0x33)
		case "4", "num4":
			vks = append(vks, 0x34)
		case "5", "num5":
			vks = append(vks, 0x35)
		case "6", "num6":
			vks = append(vks, 0x36)
		case "7", "num7":
			vks = append(vks, 0x37)
		case "8", "num8":
			vks = append(vks, 0x38)
		case "9", "num9":
			vks = append(vks, 0x39)
		case "space":
			vks = append(vks, 0x20)
		case "enter":
			vks = append(vks, 0x0D)
		case "tab":
			vks = append(vks, 0x09)
		case "backspace":
			vks = append(vks, 0x08)
		case "delete":
			vks = append(vks, 0x2E)
		case "esc", "escape":
			vks = append(vks, 0x1B)
		case "up":
			vks = append(vks, 0x26)
		case "down":
			vks = append(vks, 0x28)
		case "left":
			vks = append(vks, 0x25)
		case "right":
			vks = append(vks, 0x27)
		case "home":
			vks = append(vks, 0x24)
		case "end":
			vks = append(vks, 0x23)
		case "pageup":
			vks = append(vks, 0x21)
		case "pagedown":
			vks = append(vks, 0x22)
		case "insert":
			vks = append(vks, 0x2D)
		}
	}

	// 按下所有按键
	for _, vk := range vks {
		procKeybdEvent.Call(vk, 0, 0, 0)
		time.Sleep(50 * time.Millisecond)
	}

	// 释放所有按键（逆序）
	for i := len(vks) - 1; i >= 0; i-- {
		procKeybdEvent.Call(vks[i], 0, KEYEVENTF_KEYUP, 0)
		time.Sleep(50 * time.Millisecond)
	}
}
