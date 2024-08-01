package proxy

// import (
// 	"net/http"
// 	"net/url"
// 	"os"
// 	"reflect"
// 	"testing"
// 	"time"
// )

// var proxy Entry
// var proxyUrl, _ = url.Parse("http://127.0.0.1:8181")
// var proxyServer = NewServer(ServerOptions{
// 	Name:     "proxy",
// 	HTTPAddr: ":8080",
// 	Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		NewReverseProxy(proxyUrl, &http.Transport{}, &proxy).ServeHTTP(w, r)
// 	}),
// })

// var testServer = NewServer(ServerOptions{
// 	Name:     "test",
// 	HTTPAddr: ":8181",
// 	Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		h := r.Header
// 		for k, vv := range h {
// 			for _, v := range vv {
// 				w.Header().Add(k, v)
// 			}
// 		}
// 		w.WriteHeader(http.StatusOK)
// 	}),
// })

// var httpClient = http.DefaultClient

// func TestMain(m *testing.M) {
// 	proxyServer.Start()
// 	testServer.Start()
// 	time.Sleep(100 * time.Millisecond)
// 	code := m.Run()
// 	proxyServer.Stop()
// 	testServer.Stop()
// 	os.Exit(code)
// }

// func TestSetHeader(t *testing.T) {
// 	hWant := http.Header{"X-Test": []string{"foo", "bar"}, "X-Test2": []string{"baz"}}
// 	proxy = Entry{
// 		Alias:      "test",
// 		Scheme:     "http",
// 		Host:       "127.0.0.1",
// 		Port:       "8181",
// 		SetHeaders: hWant,
// 	}
// 	req, err := http.NewRequest("HEAD", "http://127.0.0.1:8080", nil)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	resp, err := httpClient.Do(req)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	hGot := resp.Header
// 	t.Log("headers: ", hGot)
// 	for k, v := range hWant {
// 		if !reflect.DeepEqual(hGot[k], v) {
// 			t.Errorf("header %s: expected %v, got %v", k, v, hGot[k])
// 		}
// 	}
// }

// func TestHideHeader(t *testing.T) {
// 	hHide := []string{"X-Test", "X-Test2"}
// 	proxy = Entry{
// 		Alias:       "test",
// 		Scheme:      "http",
// 		Host:        "127.0.0.1",
// 		Port:        "8181",
// 		HideHeaders: hHide,
// 	}
// 	req, err := http.NewRequest("HEAD", "http://127.0.0.1:8080", nil)
// 	for _, k := range hHide {
// 		req.Header.Set(k, "foo")
// 	}
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	resp, err := httpClient.Do(req)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	hGot := resp.Header
// 	t.Log("headers: ", hGot)
// 	for _, v := range hHide {
// 		_, ok := hGot[v]
// 		if ok {
// 			t.Errorf("header %s: expected hidden, got %v", v, hGot[v])
// 		}
// 	}
// }
