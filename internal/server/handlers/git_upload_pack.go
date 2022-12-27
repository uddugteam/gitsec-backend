package handlers

import (
	"log"
	"net/http"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/server"
)

func (h *Handlers) GitUploadPack() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		log.Printf("httpGitUploadPack %s %s", r.Method, r.URL)

		rw.Header().Set("content-type", "application/x-git-upload-pack-result")

		upr := packp.NewUploadPackRequest()
		err := upr.Decode(r.Body)
		if err != nil {
			http.Error(rw, err.Error(), 500)
			log.Println(err)
			return
		}

		ep, err := transport.NewEndpoint("/")
		if err != nil {
			http.Error(rw, err.Error(), 500)
			log.Println(err)
			return
		}
		bfs := osfs.New(h.dir)
		ld := server.NewFilesystemLoader(bfs)
		svr := server.NewServer(ld)
		sess, err := svr.NewUploadPackSession(ep, nil)
		if err != nil {
			http.Error(rw, err.Error(), 500)
			log.Println(err)
			return
		}
		res, err := sess.UploadPack(r.Context(), upr)
		if err != nil {
			http.Error(rw, err.Error(), 500)
			log.Println(err)
			return
		}

		err = res.Encode(rw)
		if err != nil {
			http.Error(rw, err.Error(), 500)
			log.Println(err)
			return
		}
	}
}
