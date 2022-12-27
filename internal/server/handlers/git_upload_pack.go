package handlers

import (
	"log"
	"net/http"

	"github.com/Misnaged/annales/logger"
)

func (h *Handlers) GitUploadPack() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		log.Printf("httpGitUploadPack %s %s", r.Method, r.URL)

		rw.Header().Set("content-type", "application/x-git-upload-pack-result")

		resp, err := h.srv.UploadPack(r.Context(), r.Body, "")
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
