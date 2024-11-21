package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}
func (cfg *apiConfig) metricsReset(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("Hits reset to 0"))
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Println("Fail to reset metrics:", err)
	}
}
func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte(fmt.Sprintf("Hits: %v", cfg.fileserverHits.Load())))
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Println("Fail to write metrics response:", err)
	}
}

func main() {
	serveMux := http.NewServeMux()
	fs := http.FileServer(http.Dir("."))
	apiCfg := apiConfig{}
	apiCfg.fileserverHits.Store(0)

	serveMux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", fs)))
	serveMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Println("Fail to write healthz response:", err)
		}
	})
	serveMux.HandleFunc("/metrics", apiCfg.handlerMetrics)
	serveMux.HandleFunc("/reset", apiCfg.metricsReset)

	server := &http.Server{
		Addr:    ":8080",
		Handler: serveMux,
	}
	log.Println("Starting server on :8080")
	log.Fatal(server.ListenAndServe())
}
