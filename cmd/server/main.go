package main

import (
	"log"
	"net/http"

	"github.com/bogdanoluic/company-service/internal/api"
)

func main() {
	router := api.NewRouter()

	log.Println("Starting server on :8080")

	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatal(err)
	}
}
