package services

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

type NotificationService struct {
	settingsService *SettingsService
}

func NewNotificationService(ss *SettingsService) *NotificationService {
	return &NotificationService{
		settingsService: ss,
	}
}

func (n *NotificationService) IsEnabled() bool {
	return n.settingsService.IsNotificationsEnabled()
}

func (n *NotificationService) SetEnabled(enabled bool) {
	n.settingsService.SetNotificationsEnabled(enabled)
}

func (n *NotificationService) Notify(stationName, songTitle string) {
	if !n.IsEnabled() || strings.TrimSpace(songTitle) == "" {
		return
	}

	switch runtime.GOOS {
	case "darwin":
		n.sendMac(stationName, songTitle)
	case "linux":
		n.sendLinux(stationName, songTitle)
	}
}

func (n *NotificationService) sendMac(stationName, songTitle string) {
	safeStation := n.escape(stationName)
	safeSong := n.escape(songTitle)
	script := fmt.Sprintf("display notification \"%s\" with title \"%s\"", safeSong, safeStation)
	exec.Command("osascript", "-e", script).Run()
}

func (n *NotificationService) sendLinux(stationName, songTitle string) {
	exec.Command("notify-send", stationName, songTitle).Run()
}

func (n *NotificationService) escape(value string) string {
	val := strings.ReplaceAll(value, "\\", "\\\\")
	val = strings.ReplaceAll(val, "\"", "\\\"")
	return val
}
