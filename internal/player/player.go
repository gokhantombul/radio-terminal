package player

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"radio-shell/internal/config"
	"radio-shell/internal/models"
	"radio-shell/internal/services"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

type AudioPlayer struct {
	config              *config.RadioConfig
	notificationService *services.NotificationService
	process             *exec.Cmd
	processDone         chan struct{}
	recordProcess       *exec.Cmd
	recordDone          chan struct{}
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
	// startMu serializes ffplay start/restart so the watchdog and volume
	// changes can never spawn two players at once.
	startMu     sync.Mutex
	stopChan    chan struct{}
	songHistory []string
	historyMu   sync.RWMutex
}

const maxSongHistory = 50

func NewAudioPlayer(cfg *config.RadioConfig, ns *services.NotificationService) *AudioPlayer {
	return &AudioPlayer{
		config:              cfg,
		notificationService: ns,
		volume:              100,
	}
}

// processAlive reports whether cmd was started and has not been reaped yet.
// done is closed by the waiter goroutine once the process exits.
func processAlive(cmd *exec.Cmd, done chan struct{}) bool {
	if cmd == nil || cmd.Process == nil || done == nil {
		return false
	}
	select {
	case <-done:
		return false
	default:
		return true
	}
}

func (p *AudioPlayer) Play(station models.RadioStation, initialVolume int, muted bool) {
	p.Stop()

	stopChan := make(chan struct{})
	p.mu.Lock()
	p.currentStation = &station
	p.volume = initialVolume
	p.muted = muted
	p.currentSong = ""
	p.playbackStartTime = time.Now()
	p.stopChan = stopChan
	p.mu.Unlock()

	p.startFFplay()

	go p.watchdogLoop(stopChan)
}

func (p *AudioPlayer) startFFplay() {
	p.startMu.Lock()
	defer p.startMu.Unlock()

	p.mu.RLock()
	station := p.currentStation
	stopChan := p.stopChan
	alive := processAlive(p.process, p.processDone)
	p.mu.RUnlock()
	if station == nil || stopChan == nil || alive {
		return
	}

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

	// Single owner of Wait: reap the process when it exits (naturally or
	// killed) so it never lingers as a zombie and liveness checks stay honest.
	done := make(chan struct{})
	go func() {
		p.monitorOutput(stderr)
		cmd.Wait()
		close(done)
	}()

	p.mu.Lock()
	if p.stopChan != stopChan {
		// Stop (or a new Play) happened while we were starting; this
		// process belongs to a dead session.
		p.mu.Unlock()
		cmd.Process.Kill()
		<-done
		return
	}
	p.process = cmd
	p.processDone = done
	p.mu.Unlock()
}

func (p *AudioPlayer) Stop() {
	p.mu.Lock()
	if p.stopChan != nil {
		close(p.stopChan)
		p.stopChan = nil
	}
	proc := p.process
	done := p.processDone
	p.process = nil
	p.processDone = nil
	p.currentStation = nil
	p.currentSong = ""
	p.codec = ""
	p.sampleRate = ""
	p.channels = ""
	p.bitrate = ""
	p.playbackStartTime = time.Time{}
	p.mu.Unlock()

	terminateProcess(proc, done, 2*time.Second)
	p.StopRecording()
}

// terminateProcess asks cmd to exit and waits until the waiter goroutine has
// reaped it, escalating to Kill after timeout. Runs without holding p.mu so
// status queries stay responsive while we wait.
func terminateProcess(cmd *exec.Cmd, done chan struct{}, timeout time.Duration) {
	if !processAlive(cmd, done) {
		return
	}
	if runtime.GOOS == "windows" {
		// os.Interrupt is not implemented on Windows; kill directly.
		cmd.Process.Kill()
		<-done
		return
	}
	cmd.Process.Signal(os.Interrupt)
	select {
	case <-done:
	case <-time.After(timeout):
		cmd.Process.Kill()
		<-done
	}
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
	return processAlive(p.process, p.processDone)
}

func (p *AudioPlayer) restartFFplay() {
	p.startMu.Lock()
	p.mu.Lock()
	proc := p.process
	done := p.processDone
	p.process = nil
	p.processDone = nil
	p.mu.Unlock()

	if processAlive(proc, done) {
		proc.Process.Kill()
		<-done
	}
	p.startMu.Unlock()

	p.startFFplay()
}

func (p *AudioPlayer) watchdogLoop(stopChan chan struct{}) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	retries := 0
	maxRetries := 3

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			if p.IsPlaying() {
				retries = 0
				continue
			}
			if retries >= maxRetries {
				return
			}
			retries++
			select {
			case <-stopChan:
				return
			case <-time.After(1 * time.Second):
			}
			p.startFFplay()
		}
	}
}

func (p *AudioPlayer) monitorOutput(stderr io.Reader) {
	scanner := bufio.NewScanner(stderr)

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

					p.historyMu.Lock()
					p.songHistory = append(p.songHistory, title)
					if len(p.songHistory) > maxSongHistory {
						p.songHistory = p.songHistory[len(p.songHistory)-maxSongHistory:]
					}
					p.historyMu.Unlock()

					p.notificationService.Notify(stationName, title)
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
	p.mu.Lock()
	defer p.mu.Unlock()

	if !processAlive(p.process, p.processDone) || p.currentStation == nil {
		return "", fmt.Errorf("not playing")
	}
	if processAlive(p.recordProcess, p.recordDone) {
		return "", fmt.Errorf("already recording")
	}
	station := *p.currentStation

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

	done := make(chan struct{})
	go func() {
		cmd.Wait()
		close(done)
	}()

	p.recordProcess = cmd
	p.recordDone = done
	p.currentRecordPath = filePath

	return fileName, nil
}

func (p *AudioPlayer) StopRecording() string {
	p.mu.Lock()
	proc := p.recordProcess
	done := p.recordDone
	path := p.currentRecordPath
	p.recordProcess = nil
	p.recordDone = nil
	p.currentRecordPath = ""
	p.mu.Unlock()

	if proc == nil || proc.Process == nil || done == nil {
		return ""
	}
	// SIGINT lets ffmpeg finalize the MP3; give it time before killing.
	terminateProcess(proc, done, 5*time.Second)
	return path
}

func (p *AudioPlayer) IsRecording() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return processAlive(p.recordProcess, p.recordDone)
}

func (p *AudioPlayer) GetStatus() (station *models.RadioStation, song string, vol int, muted bool, playing bool, recording bool, elapsed int) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	station = p.currentStation
	song = p.currentSong
	vol = p.volume
	muted = p.muted
	playing = processAlive(p.process, p.processDone)
	recording = processAlive(p.recordProcess, p.recordDone)
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

func (p *AudioPlayer) GetSongHistory() []string {
	p.historyMu.RLock()
	defer p.historyMu.RUnlock()
	result := make([]string, len(p.songHistory))
	copy(result, p.songHistory)
	return result
}
