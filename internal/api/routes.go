package api

import (
	"net/http"
)

func SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/ready", readyness)
	mux.HandleFunc("/status", liveness)
	mux.HandleFunc("/se/", Middleware(sendSEDates))
	mux.HandleFunc("/nu/", Middleware(sendNUDates))
	mux.HandleFunc("/sedomains/", Middleware(sendSERows))
	mux.HandleFunc("/nudomains/", Middleware(sendNURows))
	mux.HandleFunc("/search/", Middleware(domainSearch))
	mux.HandleFunc("/stats/", Middleware(domainStats))
	mux.HandleFunc("/seappearance/", Middleware(seDomainFirstAppearance))
	mux.HandleFunc("/nuappearance/", Middleware(nuDomainFirstAppearance))

	return mux
}
