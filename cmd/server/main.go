package main

import (
	"go-axfr-backend/internal/api"
	"log"
	"net/http"
)

func main() {
	api.InitRedis()
	mux := api.SetupRoutes()
	log.Fatal(http.ListenAndServe(":8080", mux))
}
