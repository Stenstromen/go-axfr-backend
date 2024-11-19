package api

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func SetupRoutes() *httprouter.Router {
	router := httprouter.New()

	router.GlobalOPTIONS = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header.Set("Access-Control-Allow-Methods", header.Get("Allow"))
		w.WriteHeader(http.StatusNoContent)
	})

	router.GET("/ready", readyness)
	router.GET("/status", liveness)
	router.GET("/se/:page", Middleware(sendSEDates))
	router.GET("/nu/:page", Middleware(sendNUDates))
	router.GET("/sedomains/:date/:page", Middleware(sendSERows))
	router.GET("/nudomains/:date/:page", Middleware(sendNURows))
	router.GET("/search/:tld/:query", Middleware(domainSearch))
	router.GET("/stats/:tld", Middleware(domainStats))

	return router
}
