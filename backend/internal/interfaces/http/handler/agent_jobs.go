package handler

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/voidwg/control/internal/application/port"
	"github.com/voidwg/control/internal/application/usecase"
	"github.com/voidwg/control/internal/domain"
)

// AgentJobsHandler — pull-mode endpoints для агентов.
// Phase 8: вместо push'а от control-plane'а агент самостоятельно тянет
// pending-job'ы. Авторизация — X-Node-ID + X-Node-Secret (не bearer).
type AgentJobsHandler struct {
	jobs    *usecase.JobService
	servers port.ServerRepository
}

func NewAgentJobs(j *usecase.JobService, s port.ServerRepository) *AgentJobsHandler {
	return &AgentJobsHandler{jobs: j, servers: s}
}

// authNode — резолвит сервер по node_id и сравнивает secret в constant-time.
// Возвращает server или nil + http-status, который handler должен отдать.
func (h *AgentJobsHandler) authNode(r *http.Request) (*domain.Server, int) {
	nodeID := strings.TrimSpace(r.Header.Get("X-Node-ID"))
	secret := strings.TrimSpace(r.Header.Get("X-Node-Secret"))
	if nodeID == "" || secret == "" {
		return nil, http.StatusUnauthorized
	}
	id, err := uuid.Parse(nodeID)
	if err != nil {
		return nil, http.StatusUnauthorized
	}
	srv, err := h.servers.GetByNodeID(r.Context(), id)
	if err != nil {
		return nil, http.StatusUnauthorized
	}
	if subtle.ConstantTimeCompare([]byte(srv.NodeSecret), []byte(secret)) != 1 {
		return nil, http.StatusUnauthorized
	}
	return srv, http.StatusOK
}

// Pull — GET /api/v1/agent/jobs
// Агент вызывает каждые ~10 секунд. Возвращает массив job'ов (lease).
func (h *AgentJobsHandler) Pull(w http.ResponseWriter, r *http.Request) {
	srv, code := h.authNode(r)
	if code != http.StatusOK {
		w.WriteHeader(code)
		return
	}
	jobs, err := h.jobs.Lease(r.Context(), srv.ID)
	if err != nil {
		writeErr(w, err)
		return
	}
	out := make([]map[string]any, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, map[string]any{
			"id":           j.ID,
			"type":         j.Type,
			"payload":      j.Payload,
			"attempts":     j.Attempts,
			"max_attempts": j.MaxAttempts,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// Submit — POST /api/v1/agent/jobs/{id}
// Body: { "status": "success"|"failed", "result": {...}, "error": "..." }
func (h *AgentJobsHandler) Submit(w http.ResponseWriter, r *http.Request) {
	if _, code := h.authNode(r); code != http.StatusOK {
		w.WriteHeader(code)
		return
	}
	jobID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, domain.ErrValidation)
		return
	}
	var req struct {
		Status string          `json:"status"`
		Result json.RawMessage `json:"result"`
		Error  string          `json:"error"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, err)
		return
	}
	st := domain.JobStatus(req.Status)
	if st != domain.JobSuccess && st != domain.JobFailed {
		writeErr(w, domain.ErrValidation)
		return
	}
	if err := h.jobs.Submit(r.Context(), jobID, st, req.Result, req.Error); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
