package handler

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	mw "github.com/voidwg/control/internal/interfaces/http/middleware"
)

// SystemHandler — системные операции: версия, обновление.
type SystemHandler struct {
	installDir string
}

func NewSystem(installDir string) *SystemHandler {
	if installDir == "" {
		installDir = "/opt/void-wg"
	}
	return &SystemHandler{installDir: installDir}
}

// Version — GET /api/v1/admin/system/version
// Возвращает текущий git-коммит, дату сборки и uptime.
func (h *SystemHandler) Version(w http.ResponseWriter, r *http.Request) {
	commit := h.gitHead()
	branch := h.gitBranch()

	writeJSON(w, http.StatusOK, map[string]any{
		"commit":      commit,
		"branch":      branch,
		"install_dir": h.installDir,
		"go_version":  runtime.Version(),
		"uptime_s":    int64(time.Since(startTime).Seconds()),
	})
}

// Update — POST /api/v1/admin/system/update
// Запускает scripts/update.sh асинхронно и возвращает статус немедленно.
// Логи обновления идут в /var/log/void-wg-update.log на хосте.
func (h *SystemHandler) Update(w http.ResponseWriter, r *http.Request) {
	_ = mw.UserIDFromCtx(r.Context())

	scriptPath := filepath.Join(h.installDir, "scripts", "update.sh")
	if _, err := os.Stat(scriptPath); err != nil {
		// Fallback: write trigger file and let host-side cron pick it up
		trigger := filepath.Join(h.installDir, "runtime", "update.trigger")
		_ = os.WriteFile(trigger, []byte(time.Now().Format(time.RFC3339)+"\n"), 0644)
		writeJSON(w, http.StatusAccepted, map[string]any{
			"status":  "triggered",
			"message": "trigger file written, update will start shortly",
			"trigger": trigger,
		})
		return
	}

	// Try to run the script directly (works if running on the host, not in container)
	cmd := exec.Command("bash", scriptPath)
	cmd.Env = append(os.Environ(),
		"INSTALL_DIR="+h.installDir,
		"REPO_BRANCH=main",
		"LOG_FILE=/var/log/void-wg-update.log",
	)

	if err := cmd.Start(); err != nil {
		// Running inside container — write trigger file instead
		trigger := filepath.Join(h.installDir, "runtime", "update.trigger")
		_ = os.WriteFile(trigger, []byte(time.Now().Format(time.RFC3339)+"\n"), 0644)
		writeJSON(w, http.StatusAccepted, map[string]any{
			"status":  "triggered",
			"message": "update queued via trigger file",
			"trigger": trigger,
		})
		return
	}

	// Don't wait — update runs in background
	go func() { _ = cmd.Wait() }()

	writeJSON(w, http.StatusAccepted, map[string]any{
		"status":  "started",
		"message": "update.sh started in background. Panel will restart when done.",
		"log":     "/var/log/void-wg-update.log",
	})
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
