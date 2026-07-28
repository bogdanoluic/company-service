package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/bogdanoluic/company-service/internal/api"
	"github.com/bogdanoluic/company-service/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	router := api.NewRouter()

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	log.Printf("Starting server on %s", addr)

	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
