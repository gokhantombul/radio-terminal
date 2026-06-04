package player

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"radio-shell/internal/config"
	"radio-shell/internal/models"
	"radio-shell/internal/services"
	"regexp"
	"strings"
	"sync"
	"time"
)

type AudioPlayer struct {
	config              *config.RadioConfig
	notificationService *services.NotificationService
	process             *exec.Cmd
	recordProcess       *exec.Cmd
	currentRecordPath   string
	currentStation      *models.RadioStation
	currentSong         string
	volume              int
	muted               bool
	playbackStartTime   time.Time
	codec               string
	sampleRate          string
	channels            string
	bitrate             string
	mu                  sync.RWMutex
	stopChan            chan struct{}
	onSongChange        func(string)
}

func NewAudioPlayer(cfg *config.RadioConfig, ns *services.NotificationService) *AudioPlayer {
	return &AudioPlayer{
		config:              cfg,
		notificationService: ns,
		volume:              100,
	}
}

func (p *AudioPlayer) Play(station models.RadioStation, initialVolume int, muted bool) {
	p.Stop()

	p.mu.Lock()
	p.currentStation = &station
	p.volume = initialVolume
	p.muted = muted
	p.currentSong = ""
	p.playbackStartTime = time.Now()
	p.stopChan = make(chan struct{})
	p.mu.Unlock()

	p.startFFplay()

	go p.watchdogLoop()
}

func (p *AudioPlayer) startFFplay() {
	p.mu.RLock()
	station := p.currentStation
	if station == nil {
		p.mu.RUnlock()
		return
	}
	p.mu.RUnlock()

	effectiveVol := p.GetEffectiveVolume()
	args := []string{"-nodisp", "-hide_banner", "-loglevel", "info", "-autoexit", "-volume", fmt.Sprintf("%d", effectiveVol), station.URL}

	cmd := exec.Command(p.config.Player.Command, args...)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return
	}

	if err := cmd.Start(); err != nil {
		return
	}

	p.mu.Lock()
	p.process = cmd
	p.mu.Unlock()

	go p.monitorOutput(stderr)
}

func (p *AudioPlayer) Stop() {
	p.mu.Lock()
	if p.stopChan != nil {
		close(p.stopChan)
		p.stopChan = nil
	}

	if p.process != nil && p.process.Process != nil {
		p.process.Process.Signal(os.Interrupt)
		// Wait or kill
		done := make(chan error, 1)
		go func() { done <- p.process.Wait() }()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			p.process.Process.Kill()
		}
		p.process = nil
	}

	p.currentStation = nil
	p.currentSong = ""
	p.codec = ""
	p.sampleRate = ""
	p.channels = ""
	p.bitrate = ""
	p.playbackStartTime = time.Time{}
	p.mu.Unlock()

	p.StopRecording()
}

func (p *AudioPlayer) GetEffectiveVolume() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.muted {
		return 0
	}
	return p.volume
}

func (p *AudioPlayer) SetVolume(volume int, unmute bool) {
	p.mu.Lock()
	p.volume = volume
	if unmute && volume > 0 {
		p.muted = false
	}
	p.mu.Unlock()

	if p.IsPlaying() {
		p.restartFFplay()
	}
}

func (p *AudioPlayer) SetMuted(muted bool) {
	p.mu.Lock()
	p.muted = muted
	p.mu.Unlock()

	if p.IsPlaying() {
		p.restartFFplay()
	}
}

func (p *AudioPlayer) IsPlaying() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.process != nil && p.process.Process != nil && p.process.ProcessState == nil
}

func (p *AudioPlayer) restartFFplay() {
	p.mu.Lock()
	if p.process != nil && p.process.Process != nil {
		p.process.Process.Kill()
		p.process.Wait()
		p.process = nil
	}
	p.mu.Unlock()
	p.startFFplay()
}

func (p *AudioPlayer) watchdogLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	retries := 0
	maxRetries := 3

	for {
		select {
		case <-p.stopChan:
			return
		case <-ticker.C:
			if !p.IsPlaying() {
				if retries < maxRetries {
					retries++
					time.Sleep(1 * time.Second)
					p.startFFplay()
				} else {
					return
				}
			}
		}
	}
}

func (p *AudioPlayer) monitorOutput(stderr interface{}) {
	scanner := bufio.NewScanner(stderr.(interface {
		Read(p []byte) (n int, err error)
	}))

	icyPattern := regexp.MustCompile(`StreamTitle\s*:\s*([^;]+)`)
	audioPattern := regexp.MustCompile(`Stream #.*Audio: ([^,]+), ([^,]+), ([^,]+)`)
	bitratePattern := regexp.MustCompile(`, ([0-9]+ [kK]b/s)`)

	for scanner.Scan() {
		line := scanner.Text()

		// ICY Metadata
		if match := icyPattern.FindStringSubmatch(line); len(match) > 1 {
			title := strings.TrimSpace(match[1])
			if title != "" && title != "Unknown" && title != "null" {
				p.mu.Lock()
				if title != p.currentSong {
					p.currentSong = title
					stationName := ""
					if p.currentStation != nil {
						stationName = p.currentStation.Name
					}
					p.mu.Unlock()

					p.notificationService.Notify(stationName, title)
					if p.onSongChange != nil {
						p.onSongChange(title)
					}
				} else {
					p.mu.Unlock()
				}
			}
			continue
		}

		// Audio stream info
		if match := audioPattern.FindStringSubmatch(line); len(match) > 3 {
			p.mu.Lock()
			p.codec = strings.ToUpper(strings.Fields(match[1])[0])
			if strings.Contains(p.codec, "AAC") {
				p.codec = "AAC"
			}
			p.sampleRate = strings.TrimSpace(match[2])
			p.channels = strings.TrimSpace(match[3])

			if matchB := bitratePattern.FindStringSubmatch(line); len(matchB) > 1 {
				p.bitrate = strings.TrimSpace(matchB[1])
			}
			p.mu.Unlock()
		}
	}
}

func (p *AudioPlayer) StartRecording() (string, error) {
	if !p.IsPlaying() {
		return "", fmt.Errorf("not playing")
	}

	p.mu.RLock()
	station := p.currentStation
	p.mu.RUnlock()

	if p.IsRecording() {
		return "", fmt.Errorf("already recording")
	}

	p.config.EnsureDirs()
	safeName := ""
	for _, r := range strings.ToLower(station.Name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			safeName += string(r)
		} else {
			safeName += "_"
		}
	}
	timestamp := time.Now().Format("20060102_150405")
	fileName := fmt.Sprintf("%s_%s.mp3", safeName, timestamp)
	filePath := filepath.Join(p.config.RecordingsDir, fileName)

	args := []string{
		"-y",
		"-user_agent", "VLC/3.0.16 LibVLC/3.0.16",
		"-reconnect", "1", "-reconnect_at_eof", "1", "-reconnect_streamed", "1", "-reconnect_delay_max", "5",
		"-i", station.URL,
		"-c:a", "libmp3lame", "-b:a", "128k",
		filePath,
	}

	cmd := exec.Command("ffmpeg", args...)
	if err := cmd.Start(); err != nil {
		return "", err
	}

	p.mu.Lock()
	p.recordProcess = cmd
	p.currentRecordPath = filePath
	p.mu.Unlock()

	return fileName, nil
}

func (p *AudioPlayer) StopRecording() string {
	if !p.IsRecording() {
		return ""
	}

	p.mu.Lock()
	if p.recordProcess != nil && p.recordProcess.Process != nil {
		p.recordProcess.Process.Signal(os.Interrupt)
		p.recordProcess.Wait()
		p.recordProcess = nil
	}
	path := p.currentRecordPath
	p.currentRecordPath = ""
	p.mu.Unlock()

	return path
}

func (p *AudioPlayer) IsRecording() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.recordProcess != nil && p.recordProcess.Process != nil && p.recordProcess.ProcessState == nil
}

func (p *AudioPlayer) GetStatus() (station *models.RadioStation, song string, vol int, muted bool, playing bool, recording bool, elapsed int) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	station = p.currentStation
	song = p.currentSong
	vol = p.volume
	muted = p.muted
	playing = p.process != nil && p.process.Process != nil && p.process.ProcessState == nil
	recording = p.recordProcess != nil && p.recordProcess.Process != nil && p.recordProcess.ProcessState == nil
	if !p.playbackStartTime.IsZero() {
		elapsed = int(time.Since(p.playbackStartTime).Seconds())
	}
	return
}

func (p *AudioPlayer) GetCodecInfo() (codec, bitrate, sampleRate string) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.codec, p.bitrate, p.sampleRate
}

func (p *AudioPlayer) SetOnSongChange(f func(string)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onSongChange = f
}
