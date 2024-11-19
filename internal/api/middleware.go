package api

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
)

func Middleware(next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		http.MaxBytesReader(w, r.Body, 1048576)
		w.Header().Set("content-type", "application/json")
		next(w, r, ps)
	}
}
