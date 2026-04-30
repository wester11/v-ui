package handler

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	mw "github.com/voidwg/control/internal/interfaces/http/middleware"
)

// SystemHandler — системные операции: версия, обновление.
type SystemHandler struct {
	installDir string
	updateMu   sync.Mutex // один update за раз
}

func NewSystem(installDir string) *SystemHandler {
	if installDir == "" {
		installDir = "/opt/void-wg"
	}
	return &SystemHandler{installDir: installDir}
}

// Version — GET /api/v1/admin/system/version
func (h *SystemHandler) Version(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"commit":     h.gitHead(),
		"branch":     h.gitBranch(),
		"go_version": runtime.Version(),
		"uptime_seconds": int64(time.Since(startTime).Seconds()),
		"built_at":   "unknown",
	})
}

// Update — POST /api/v1/admin/system/update
// Запускает update.sh асинхронно.
func (h *SystemHandler) Update(w http.ResponseWriter, r *http.Request) {
	_ = mw.UserIDFromCtx(r.Context())

	scriptPath := filepath.Join(h.installDir, "scripts", "update.sh")
	if _, err := os.Stat(scriptPath); err != nil {
		trigger := filepath.Join(h.installDir, "runtime", "update.trigger")
		_ = os.WriteFile(trigger, []byte(time.Now().Format(time.RFC3339)+"\n"), 0644)
		writeJSON(w, http.StatusAccepted, map[string]any{
			"status":  "triggered",
			"message": "trigger file written, update will start shortly",
		})
		return
	}

	cmd := exec.Command("bash", scriptPath)
	cmd.Env = append(os.Environ(),
		"INSTALL_DIR="+h.installDir,
		"REPO_BRANCH=main",
		"LOG_FILE=/var/log/void-wg-update.log",
	)
	if err := cmd.Start(); err != nil {
		trigger := filepath.Join(h.installDir, "runtime", "update.trigger")
		_ = os.WriteFile(trigger, []byte(time.Now().Format(time.RFC3339)+"\n"), 0644)
		writeJSON(w, http.StatusAccepted, map[string]any{
			"status":  "triggered",
			"message": "update queued via trigger file",
		})
		return
	}
	go func() { _ = cmd.Wait() }()
	writeJSON(w, http.StatusAccepted, map[string]any{
		"status":  "started",
		"message": "update.sh started in background",
		"log":     "/var/log/void-wg-update.log",
	})
}

// UpdateStream — GET /api/v1/admin/system/update/stream
// SSE: запускает update.sh и стримит stdout+stderr построчно.
// Отправляет "__DONE__" когда завершится, "__ERROR__:..." при ошибке.
func (h *SystemHandler) UpdateStream(w http.ResponseWriter, r *http.Request) {
	_ = mw.UserIDFromCtx(r.Context())

	// Только один update за раз
	if !h.updateMu.TryLock() {
		http.Error(w, "update already in progress", http.StatusConflict)
		return
	}
	defer h.updateMu.Unlock()

	// SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // nginx: disable buffering

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	send := func(line string) {
		// Escape newlines inside the data field
		line = strings.ReplaceAll(line, "\n", " ")
		fmt.Fprintf(w, "data: %s\n\n", line)
		flusher.Flush()
	}

	scriptPath := filepath.Join(h.installDir, "scripts", "update.sh")
	if _, err := os.Stat(scriptPath); err != nil {
		send("__ERROR__:update.sh not found at " + scriptPath)
		return
	}

	cmd := exec.CommandContext(r.Context(), "bash", scriptPath)
	cmd.Env = append(os.Environ(),
		"INSTALL_DIR="+h.installDir,
		"REPO_BRANCH=main",
		"LOG_FILE=/var/log/void-wg-update.log",
		"TERM=dumb",  // no ANSI escapes
	)

	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw

	if err := cmd.Start(); err != nil {
		send("__ERROR__:" + err.Error())
		return
	}

	// Wait in goroutine so pw is closed when process exits,
	// which causes the scanner loop to terminate naturally.
	waitErr := make(chan error, 1)
	go func() {
		waitErr <- cmd.Wait()
		pw.Close()
	}()

	scanner := bufio.NewScanner(pr)
	for scanner.Scan() {
		select {
		case <-r.Context().Done():
			return
		default:
			send(scanner.Text())
		}
	}

	if err := <-waitErr; err != nil {
		send("__ERROR__:exit: " + err.Error())
	} else {
		send("__DONE__")
	}
}

var startTime = time.Now()

func (h *SystemHandler) gitHead() string {
	out, err := exec.Command("git", "-C", h.installDir, "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func (h *SystemHandler) gitBranch() string {
	out, err := exec.Command("git", "-C", h.installDir, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "main"
	}
	return strings.TrimSpace(string(out))
}
