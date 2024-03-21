package main

import (
	"html/template"
	"net"
	"net/http"
	"net/url"
	"time"
)

var healthCheckHttpClient = &http.Client{
	Timeout: 5 * time.Second,
	Transport: &http.Transport{
		Proxy:             http.ProxyFromEnvironment,
		DisableKeepAlives: true,
		ForceAttemptHTTP2: true,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 5 * time.Second,
		}).DialContext,
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
		palog.Errorf("%s not found", r.URL.Path)
		http.NotFound(w, r)
		return
	}
}

func panelIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tmpl, err := template.ParseFiles(templatePath)

	if err != nil {
		palog.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type allRoutes struct {
		HTTPRoutes   HTTPRoutes
		StreamRoutes StreamRoutes
	}

	err = tmpl.Execute(w, allRoutes{
		HTTPRoutes:   httpRoutes,
		StreamRoutes: streamRoutes,
	})
	if err != nil {
		palog.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func panelCheckTargetHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	targetUrl := r.URL.Query().Get("target")

	if targetUrl == "" {
		http.Error(w, "target is required", http.StatusBadRequest)
		return
	}

	url, err := url.Parse(targetUrl)
	if err != nil {
		palog.Infof("failed to parse url %q, error: %v", targetUrl, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	scheme := url.Scheme

	if isStreamScheme(scheme) {
		err = utils.healthCheckStream(scheme, url.Host)
	} else {
		err = utils.healthCheckHttp(targetUrl)
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}
