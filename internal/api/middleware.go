package api

import (
	"net/http"
)

func Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.MaxBytesReader(w, r.Body, 1048576)
		w.Header().Set("content-type", "application/json")
		next(w, r)
	}
}
