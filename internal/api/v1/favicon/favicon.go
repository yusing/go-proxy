package favicon

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/vincent-petithory/dataurl"
	U "github.com/yusing/go-proxy/internal/api/v1/utils"
	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/homepage"
	"github.com/yusing/go-proxy/internal/logging"
	gphttp "github.com/yusing/go-proxy/internal/net/http"
	"github.com/yusing/go-proxy/internal/route/routes"
	route "github.com/yusing/go-proxy/internal/route/types"
	"github.com/yusing/go-proxy/internal/task"
	"github.com/yusing/go-proxy/internal/utils"
)

type content struct {
	header http.Header
	data   []byte
	status int
}

func newContent() *content {
	return &content{
		header: make(http.Header),
	}
}

func (c *content) Header() http.Header {
	return c.header
}

func (c *content) Write(data []byte) (int, error) {
	c.data = append(c.data, data...)
	return len(data), nil
}

func (c *content) WriteHeader(statusCode int) {
	c.status = statusCode
}

func (c *content) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, errors.New("not supported")
}

// GetFavIcon returns the favicon of the route
//
// Returns:
//   - 200 OK: if icon found
//   - 400 Bad Request: if alias is empty or route is not HTTPRoute
//   - 404 Not Found: if route or icon not found
//   - 500 Internal Server Error: if internal error
//   - others: depends on route handler response
func GetFavIcon(w http.ResponseWriter, req *http.Request) {
	url, alias := req.FormValue("url"), req.FormValue("alias")
	if url == "" && alias == "" {
		U.RespondError(w, U.ErrMissingKey("url or alias"), http.StatusBadRequest)
		return
	}
	if url != "" && alias != "" {
		U.RespondError(w, U.ErrInvalidKey("url and alias are mutually exclusive"), http.StatusBadRequest)
		return
	}

	// try with url
	if url != "" {
		var iconURL homepage.IconURL
		if err := iconURL.Parse(url); err != nil {
			U.RespondError(w, err, http.StatusBadRequest)
			return
		}
		icon, status, errMsg := getFavIconFromURL(&iconURL)
		if icon == nil {
			http.Error(w, errMsg, status)
			return
		}
		U.WriteBody(w, icon)
		return
	}

	// try with route.Homepage.Icon
	r, ok := routes.GetHTTPRoute(alias)
	if !ok {
		U.RespondError(w, errors.New("no such route"), http.StatusNotFound)
		return
	}
	var icon []byte
	var status int
	var errMsg string

	hp := r.RawEntry().Homepage.GetOverride()
	if !hp.IsEmpty() && hp.Icon != nil {
		switch hp.Icon.IconSource {
		case homepage.IconSourceRelative:
			icon, status, errMsg = findIcon(r, req, hp.Icon.Value)
		default:
			icon, status, errMsg = getFavIconFromURL(hp.Icon)
		}
	} else {
		// try extract from "link[rel=icon]"
		icon, status, errMsg = findIcon(r, req, "/")
	}
	if status != http.StatusOK {
		http.Error(w, errMsg, status)
		return
	}
	U.WriteBody(w, icon)
}

func getFavIconFromURL(iconURL *homepage.IconURL) ([]byte, int, string) {
	switch iconURL.IconSource {
	case homepage.IconSourceAbsolute:
		return fetchIconAbsolute(iconURL.URL())
	case homepage.IconSourceRelative:
		return nil, http.StatusBadRequest, "unexpected relative icon"
	case homepage.IconSourceWalkXCode, homepage.IconSourceSelfhSt:
		return fetchKnownIcon(iconURL)
	}
	return nil, http.StatusBadRequest, "invalid icon source"
}

// cache key can be absolute url or route name.
var (
	iconCache   = make(map[string][]byte)
	iconCacheMu sync.RWMutex
)

func InitIconCache() {
	err := utils.LoadJSONIfExist(common.IconCachePath, &iconCache)
	if err != nil {
		logging.Error().Err(err).Msg("failed to load icon cache")
	} else {
		logging.Info().Msgf("icon cache loaded (%d icons)", len(iconCache))
	}

	task.OnProgramExit("save_favicon_cache", func() {
		iconCacheMu.Lock()
		defer iconCacheMu.Unlock()

		if err := utils.SaveJSON(common.IconCachePath, &iconCache, 0o644); err != nil {
			logging.Error().Err(err).Msg("failed to save icon cache")
		}
	})
}

func ResetIconCache(route route.HTTPRoute) {
	iconCacheMu.Lock()
	defer iconCacheMu.Unlock()
	delete(iconCache, route.TargetName())
}

func loadIconCache(key string) (icon []byte, ok bool) {
	iconCacheMu.RLock()
	defer iconCacheMu.RUnlock()
	icon, ok = iconCache[key]
	if ok {
		logging.Debug().
			Str("key", key).
			Msg("icon found in cache")
	}
	return
}

func storeIconCache(key string, icon []byte) {
	iconCacheMu.Lock()
	defer iconCacheMu.Unlock()
	iconCache[key] = icon
}

func fetchIconAbsolute(url string) ([]byte, int, string) {
	icon, ok := loadIconCache(url)
	if ok {
		return icon, http.StatusOK, ""
	}

	resp, err := U.Get(url)
	if err != nil || resp.StatusCode != http.StatusOK {
		if err == nil {
			err = errors.New(resp.Status)
		}
		logging.Error().Err(err).
			Str("url", url).
			Msg("failed to get icon")
		return nil, http.StatusBadGateway, "connection error"
	}

	defer resp.Body.Close()
	icon, err = io.ReadAll(resp.Body)
	if err != nil {
		logging.Error().Err(err).
			Str("url", url).
			Msg("failed to read icon")
		return nil, http.StatusInternalServerError, "internal error"
	}

	storeIconCache(url, icon)
	return icon, http.StatusOK, ""
}

var nameSanitizer = strings.NewReplacer(
	"_", "-",
	" ", "-",
	"(", "",
	")", "",
)

func sanitizeName(name string) string {
	return strings.ToLower(nameSanitizer.Replace(name))
}

func fetchKnownIcon(url *homepage.IconURL) ([]byte, int, string) {
	// if icon isn't in the list, no need to fetch
	if !url.HasIcon() {
		logging.Debug().
			Str("value", url.String()).
			Str("url", url.URL()).
			Msg("no such icon")
		return nil, http.StatusNotFound, "no such icon"
	}

	return fetchIconAbsolute(url.URL())
}

func fetchIcon(filetype, filename string) (icon []byte, status int, errMsg string) {
	icon, status, errMsg = fetchKnownIcon(homepage.NewSelfhStIconURL(filename, filetype))
	if icon != nil {
		return
	}
	icon, status, errMsg = fetchKnownIcon(homepage.NewWalkXCodeIconURL(filename, filetype))
	return
}

func findIcon(r route.HTTPRoute, req *http.Request, uri string) (icon []byte, status int, errMsg string) {
	key := r.RawEntry().Provider + ":" + r.TargetName()
	icon, ok := loadIconCache(key)
	if ok {
		if icon == nil {
			return nil, http.StatusNotFound, "icon not found"
		}
		return icon, http.StatusOK, ""
	}

	icon, status, errMsg = fetchIcon("png", sanitizeName(r.TargetName()))
	cont := r.RawEntry().Container
	if icon == nil && cont != nil {
		icon, status, errMsg = fetchIcon("png", sanitizeName(cont.ImageName))
	}
	if icon == nil {
		// fallback to parse html
		icon, status, errMsg = findIconSlow(r, req, uri)
	}
	if icon != nil {
		storeIconCache(key, icon)
	}
	return
}

func findIconSlow(r route.HTTPRoute, req *http.Request, uri string) (icon []byte, status int, errMsg string) {
	ctx, cancel := context.WithTimeoutCause(req.Context(), 3*time.Second, errors.New("favicon request timeout"))
	defer cancel()
	newReq := req.WithContext(ctx)
	newReq.Header.Set("Accept-Encoding", "identity") // disable compression
	if !strings.HasPrefix(uri, "/") {
		uri = "/" + uri
	}
	u, err := url.ParseRequestURI(uri)
	if err != nil {
		logging.Error().Err(err).
			Str("route", r.TargetName()).
			Str("path", uri).
			Msg("failed to parse uri")
		return nil, http.StatusInternalServerError, "cannot parse uri"
	}
	newReq.URL.Path = u.Path
	newReq.URL.RawPath = u.RawPath
	newReq.URL.RawQuery = u.RawQuery
	newReq.RequestURI = u.String()

	c := newContent()
	r.ServeHTTP(c, newReq)
	if c.status != http.StatusOK {
		switch c.status {
		case 0:
			return nil, http.StatusBadGateway, "connection error"
		default:
			if loc := c.Header().Get("Location"); loc != "" {
				loc = path.Clean(loc)
				if !strings.HasPrefix(loc, "/") {
					loc = "/" + loc
				}
				if loc == newReq.URL.Path {
					return nil, http.StatusBadGateway, "circular redirect"
				}
				return findIconSlow(r, req, loc)
			}
		}
		return nil, c.status, "upstream error: " + http.StatusText(c.status)
	}
	// return icon data
	if !gphttp.GetContentType(c.header).IsHTML() {
		return c.data, http.StatusOK, ""
	}
	// try extract from "link[rel=icon]" from path "/"
	doc, err := goquery.NewDocumentFromReader(bytes.NewBuffer(c.data))
	if err != nil {
		logging.Error().Err(err).
			Str("route", r.TargetName()).
			Msg("failed to parse html")
		return nil, http.StatusInternalServerError, "internal error"
	}
	ele := doc.Find("head > link[rel=icon]").First()
	if ele.Length() == 0 {
		return nil, http.StatusNotFound, "icon element not found"
	}
	href := ele.AttrOr("href", "")
	if href == "" {
		return nil, http.StatusNotFound, "icon href not found"
	}
	// https://en.wikipedia.org/wiki/Data_URI_scheme
	if strings.HasPrefix(href, "data:image/") {
		dataURI, err := dataurl.DecodeString(href)
		if err != nil {
			logging.Error().Err(err).
				Str("route", r.TargetName()).
				Msg("failed to decode favicon")
			return nil, http.StatusInternalServerError, "internal error"
		}
		return dataURI.Data, http.StatusOK, ""
	}
	switch {
	case strings.HasPrefix(href, "http://"), strings.HasPrefix(href, "https://"):
		return fetchIconAbsolute(href)
	default:
		return findIconSlow(r, req, path.Clean(href))
	}
}
