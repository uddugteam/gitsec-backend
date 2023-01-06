package handlers

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/misnaged/annales/logger"
)

// GitReceivePack is an HTTP handler that processes a
// "git-receive-pack" request.
func (h *Handlers) GitReceivePack() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("content-type", "application/x-git-receive-pack-result")

		resp, err := h.srv.ReceivePack(r.Context(), r.Body, chi.URLParam(r, repoNamePath))
		if err != nil {
			http.Error(rw, err.Error(), 500)
			logger.Log().Error(err)
			return
		}

		if err = resp.Encode(rw); err != nil {
			http.Error(rw, err.Error(), 500)
			log.Println(err)
			return
		}
	}
}
