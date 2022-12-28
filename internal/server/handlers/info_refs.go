package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Misnaged/annales/logger"
	"github.com/go-chi/chi/v5"
)

func (h *Handlers) InfoRef() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		log.Printf("httpInfoRefs %s %s", r.Method, r.URL)

		service := r.URL.Query().Get("service")
		if service != "git-upload-pack" && service != "git-receive-pack" {
			http.Error(rw, "only smart git", 403)
			return
		}

		rw.Header().Set("content-type", fmt.Sprintf("application/x-%s-advertisement", service))

		resp, err := h.srv.InfoRef(r.Context(), chi.URLParam(r, repoNamePath), service)
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
