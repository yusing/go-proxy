package go_proxy

import (
	"html/template"
	"net/http"
	"time"
)

const templateFile = "/app/templates/panel.html"

var healthCheckHttpClient = &http.Client{
	Timeout: 5 * time.Second,
	Transport: &http.Transport{
		DisableKeepAlives: true,
	},
}

func panelHandler(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		panelIndex(w, r)
		return
	case "/checkhealth":
		panelCheckTargetHealth(w, r)
		return
	default:
		http.NotFound(w, r)
		return
	}
}

func panelIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tmpl, err := template.ParseFiles(templateFile)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, subdomainRouteMap)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func panelCheckTargetHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	targetUrl := r.URL.Query().Get("target")

	if targetUrl == "" {
		http.Error(w, "target is required", http.StatusBadRequest)
		return
	}

	// try HEAD first
	// if HEAD is not allowed, try GET
	resp, err := healthCheckHttpClient.Head(targetUrl)
	if err != nil && resp != nil && resp.StatusCode == http.StatusMethodNotAllowed {
		_, err = healthCheckHttpClient.Get(targetUrl)
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
}
