package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"
)

func main() {
	port := os.Getenv("PORT")
	proxy := httputil.NewSingleHostReverseProxy(&url.URL{
		Scheme: "https",
		Host:   "credhub.service.cf.internal:8844",
	})
	proxy.FlushInterval = 100 * time.Millisecond
	http.ListenAndServe(fmt.Sprintf(":%s", port), proxy)
}
