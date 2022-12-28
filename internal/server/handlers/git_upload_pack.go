package handlers

import (
	"log"
	"net/http"

	"github.com/Misnaged/annales/logger"
	"github.com/go-chi/chi/v5"
)

func (h *Handlers) GitUploadPack() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("content-type", "application/x-git-upload-pack-result")

		resp, err := h.srv.UploadPack(r.Context(), r.Body, chi.URLParam(r, repoNamePath))
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

		return
	}
}
