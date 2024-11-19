package main

import (
	"log"
	"net/http"
)

func main() {
	serveMux := http.NewServeMux()
	serveMux.Handle("/app/", http.StripPrefix("/app/", http.FileServer(http.Dir("."))))
	serveMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			log.Fatal("Fail to write")
		}
	})
	server := &http.Server{
		Addr:    ":8080",
		Handler: serveMux,
	}
	log.Fatal(server.ListenAndServe())
}
