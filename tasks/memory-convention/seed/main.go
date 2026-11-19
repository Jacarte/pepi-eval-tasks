package main

import (
	"net/http"
	"os"

	"payments/internal/api"
	"payments/internal/store"
	"payments/internal/xlog"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	lg := xlog.For("main")
	lg.Info("listening", "port", port)
	if err := http.ListenAndServe(":"+port, api.NewServer(store.NewMemory())); err != nil {
		lg.Error("server stopped", "err", err.Error())
		os.Exit(1)
	}
}
