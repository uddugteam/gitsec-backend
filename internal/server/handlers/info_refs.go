package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5/plumbing/format/pktline"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/server"
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

		ep, err := transport.NewEndpoint("/")
		if err != nil {
			http.Error(rw, err.Error(), 500)
			log.Println(err)
			return
		}
		bfs := osfs.New(h.dir)
		ld := server.NewFilesystemLoader(bfs)
		svr := server.NewServer(ld)

		var sess transport.Session

		if service == "git-upload-pack" {
			sess, err = svr.NewUploadPackSession(ep, nil)
			if err != nil {
				http.Error(rw, err.Error(), 500)
				log.Println(err)
				return
			}
		} else {
			sess, err = svr.NewReceivePackSession(ep, nil)
			if err != nil {
				http.Error(rw, err.Error(), 500)
				log.Println(err)
				return
			}
		}

		ar, err := sess.AdvertisedReferencesContext(r.Context())
		if err != nil {
			http.Error(rw, err.Error(), 500)
			log.Println(err)
			return
		}
		ar.Prefix = [][]byte{
			[]byte(fmt.Sprintf("# service=%s", service)),
			pktline.Flush,
		}
		err = ar.Encode(rw)
		if err != nil {
			http.Error(rw, err.Error(), 500)
			log.Println(err)
			return
		}
	}
}
