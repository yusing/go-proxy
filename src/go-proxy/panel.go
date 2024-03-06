package main

import (
	"html/template"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/golang/glog"
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

	err = tmpl.Execute(w, &routes)
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

	url, err := url.Parse(targetUrl)
	if err != nil {
		glog.Infof("[Panel] failed to parse %s, error: %v", targetUrl, err)
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
