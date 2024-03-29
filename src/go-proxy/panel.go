package main

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"path"
)

var panelHandler = panelRouter()

func panelRouter() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", panelServeFile)
	mux.HandleFunc("GET /{file}", panelServeFile)
	mux.HandleFunc("GET /panel/", panelPage)
	mux.HandleFunc("GET /panel/{file}", panelServeFile)
	mux.HandleFunc("HEAD /checkhealth", panelCheckTargetHealth)
	mux.HandleFunc("GET /config_editor/", panelConfigEditor)
	mux.HandleFunc("GET /config_editor/{file}", panelServeFile)
	mux.HandleFunc("GET /config/{file}", panelConfigGet)
	mux.HandleFunc("PUT /config/{file}", panelConfigUpdate)
	mux.HandleFunc("POST /reload", configReload)
	mux.HandleFunc("GET /codemirror/", panelServeFile)
	return mux
}

func panelPage(w http.ResponseWriter, r *http.Request) {
	resp := struct {
		HTTPRoutes   HTTPRoutes
		StreamRoutes StreamRoutes
	}{httpRoutes, streamRoutes}

	panelRenderFile(w, r, panelTemplatePath, resp)
}

func panelCheckTargetHealth(w http.ResponseWriter, r *http.Request) {
	targetUrl := r.URL.Query().Get("target")

	if targetUrl == "" {
		panelHandleErr(w, r, errors.New("target is required"), http.StatusBadRequest)
		return
	}

	url, err := url.Parse(targetUrl)
	if err != nil {
		err = NewNestedError("failed to parse url").Subject(targetUrl).With(err)
		panelHandleErr(w, r, err, http.StatusBadRequest)
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

func panelConfigEditor(w http.ResponseWriter, r *http.Request) {
	cfgFiles := make([]string, 0)
	cfgFiles = append(cfgFiles, path.Base(configPath))
	for _, p := range cfg.Value().Providers {
		if p.Kind != ProviderKind_File {
			continue
		}
		cfgFiles = append(cfgFiles, p.Value)
	}

	panelRenderFile(w, r, configEditorTemplatePath, cfgFiles)
}

func panelConfigGet(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, path.Join(configBasePath, r.PathValue("file")))
}

func panelConfigUpdate(w http.ResponseWriter, r *http.Request) {
	p := r.PathValue("file")
	content := make([]byte, r.ContentLength)
	_, err := r.Body.Read(content)
	if err != nil {
		panelHandleErr(w, r, NewNestedError("unable to read request body").Subject(p).With(err))
		return
	}
	if p == path.Base(configPath) {
		err = ValidateConfig(content)
	} else {
		_, err = ValidateFileContent(content)
	}
	if err != nil {
		panelHandleErr(w, r, err)
		return
	}
	p = path.Join(configBasePath, p)
	_, err = os.Stat(p)
	exists := !errors.Is(err, os.ErrNotExist)
	err = os.WriteFile(p, content, 0644)
	if err != nil {
		panelHandleErr(w, r, NewNestedError("unable to write config file").With(err))
		return
	}
	w.WriteHeader(http.StatusOK)
	if !exists {
		w.Write([]byte(fmt.Sprintf("Config file %s created, remember to add it to config.yml!", p)))
		return
	}
	w.Write([]byte(fmt.Sprintf("Config file %s updated", p)))
}

func panelServeFile(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, path.Join(templatesBasePath, r.URL.Path))
}

func panelRenderFile(w http.ResponseWriter, r *http.Request, f string, data any) {
	tmpl, err := template.ParseFiles(f)
	if err != nil {
		panelHandleErr(w, r, NewNestedError("unable to parse template").With(err))
		return
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		panelHandleErr(w, r, NewNestedError("unable to render template").With(err))
	}
}

func configReload(w http.ResponseWriter, r *http.Request) {
	err := cfg.Reload()
	if err != nil {
		panelHandleErr(w, r, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func panelHandleErr(w http.ResponseWriter, r *http.Request, err error, code ...int) {
	err = NewNestedErrorFrom(err).Subjectf("%s %s", r.Method, r.URL)
	palog.Error(err)
	if len(code) > 0 {
		http.Error(w, err.Error(), code[0])
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
