package main

import (
	"github.com/IrisIris/spot-instance-advisor/handler"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"time"
)

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/spot", handler.SpotHandler)
	http.Handle("/", r)

	srv := &http.Server{
		Handler: r,
		Addr:    ":8000",
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	handler.AlarmHandler(nil, nil)
	log.Fatal(srv.ListenAndServe())
}
