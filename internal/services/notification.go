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
	case "windows":
		n.sendWindows(stationName, songTitle)
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

func (n *NotificationService) sendWindows(stationName, songTitle string) {
	xmlEsc := func(s string) string {
		s = strings.ReplaceAll(s, "&", "&amp;")
		s = strings.ReplaceAll(s, "<", "&lt;")
		s = strings.ReplaceAll(s, ">", "&gt;")
		s = strings.ReplaceAll(s, "\"", "&quot;")
		s = strings.ReplaceAll(s, "'", "&apos;")
		return s
	}
	xml := fmt.Sprintf(
		`<toast><visual><binding template="ToastGeneric"><text>%s</text><text>%s</text></binding></visual></toast>`,
		xmlEsc(stationName), xmlEsc(songTitle),
	)
	script := `[void][Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime];` +
		`[void][Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime];` +
		`$x = New-Object Windows.Data.Xml.Dom.XmlDocument;` +
		`$x.LoadXml('` + xml + `');` +
		`$t = [Windows.UI.Notifications.ToastNotification]::new($x);` +
		`[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('Radio Terminal').Show($t)`
	exec.Command("powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", script).Run()
}

func (n *NotificationService) escape(value string) string {
	val := strings.ReplaceAll(value, "\\", "\\\\")
	val = strings.ReplaceAll(val, "\"", "\\\"")
	return val
}
