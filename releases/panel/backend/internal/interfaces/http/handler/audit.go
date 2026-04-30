package handler

import (
	"net/http"
	"strconv"

	"github.com/voidwg/control/internal/application/usecase"
	"github.com/voidwg/control/internal/interfaces/http/dto"
)

type AuditHandler struct{ svc *usecase.AuditService }

func NewAudit(s *usecase.AuditService) *AuditHandler { return &AuditHandler{svc: s} }

// List — admin-only журнал. Пагинация cursor-style: ?before=<id>&limit=<n>.
func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 {
		limit = 100
	}
	before, _ := strconv.ParseInt(q.Get("before"), 10, 64)
	events, err := h.svc.List(r.Context(), limit, before)
	if err != nil {
		writeErr(w, err)
		return
	}
	out := make([]dto.AuditEntry, 0, len(events))
	for _, e := range events {
		out = append(out, dto.AuditFromDomain(e))
	}
	writeJSON(w, http.StatusOK, out)
}
