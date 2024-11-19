package main

import (
	"go-axfr-backend/internal/api"
	"net/http"
)

func main() {
	router := api.SetupRoutes()
	http.ListenAndServe(":8080", router)
}
