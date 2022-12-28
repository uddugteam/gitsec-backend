package handlers

import (
	"fmt"
	"net/http"

	"github.com/Misnaged/annales/logger"
	"github.com/go-chi/chi/v5"

	"gitsec-backend/internal/models"
)

func (h *Handlers) InfoRef() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		infoRefRequestType, err := models.GitSessionTypeFromString(r.URL.Query().Get("service"))
		if err != nil {
			http.Error(rw, "only smart git", 403)
			return
		}

		rw.Header().Set("content-type", fmt.Sprintf("application/x-%s-advertisement", infoRefRequestType.String()))

		resp, err := h.srv.InfoRef(r.Context(), chi.URLParam(r, repoNamePath), infoRefRequestType)
		if err != nil {
			http.Error(rw, err.Error(), 500)
			logger.Log().Error(err)
			return
		}

		if err = resp.Encode(rw); err != nil {
			http.Error(rw, err.Error(), 500)
			logger.Log().Error(err)
			return
		}
	}
}
