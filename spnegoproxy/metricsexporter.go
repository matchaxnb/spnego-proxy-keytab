package spnegoproxy

import (
	"io"
	"net/http"
)

func ExposeMetrics(listenAddr string, events WebHDFSEventChannel) {
	srv := http.NewServeMux()

	srv.HandleFunc("/metrics", handleMetrics)
	srv.HandleFunc("/metrics/", handleMetrics)
	srv.HandleFunc("/", handleRoot)
	go serveMetrics(listenAddr, srv)
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "use /metrics")
}

func handleMetrics(w http.ResponseWriter, r *http.Request) {
	// ctx := r.Context()
	io.WriteString(w, webHDFSEvents.String())

}

func serveMetrics(addr string, server *http.ServeMux) {
	http.ListenAndServe(addr, server)
}
