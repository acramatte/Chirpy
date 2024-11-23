package main

import (
	"log"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func main() {
	fs := http.FileServer(http.Dir("."))
	apiCfg := apiConfig{}
	apiCfg.fileserverHits.Store(0)

	serveMux := http.NewServeMux()
	fsHandler := apiCfg.middlewareMetricsInc(http.StripPrefix("/app", fs))
	serveMux.Handle("/app/", fsHandler)

	serveMux.HandleFunc("GET /api/healthz", handlerReadiness)
	serveMux.HandleFunc("POST /admin/reset", apiCfg.metricsReset)
	serveMux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetrics)

	server := &http.Server{
		Addr:    ":8080",
		Handler: serveMux,
	}
	log.Println("Starting server on :8080")
	log.Fatal(server.ListenAndServe())
}
