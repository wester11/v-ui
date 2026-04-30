package handler

import (
	"net/http"

	"github.com/voidwg/control/internal/application/usecase"
	"github.com/voidwg/control/internal/interfaces/http/dto"
)

type StatsHandler struct {
	svc *usecase.StatsService
}

func NewStats(s *usecase.StatsService) *StatsHandler {
	return &StatsHandler{svc: s}
}

func (h *StatsHandler) Get(w http.ResponseWriter, r *http.Request) {
	st, err := h.svc.Get(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.StatsResponse{
		Users:         st.Users,
		Peers:         st.Peers,
		Servers:       st.Servers,
		ServersOnline: st.ServersOnline,
		BytesRxTotal:  st.BytesRxTotal,
		BytesTxTotal:  st.BytesTxTotal,
	})
}
