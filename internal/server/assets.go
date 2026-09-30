package server

import (
	"io/fs"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"

	webembed "caddy-configurator/web"
)

// NewFrontendHandler serves the embedded production SPA, or proxies every
// request to a vite dev server when devURL is non-empty (serve --dev-frontend).
func NewFrontendHandler(devURL string) (http.Handler, error) {
	if devURL != "" {
		target, err := url.Parse(devURL)
		if err != nil {
			return nil, err
		}
		return httputil.NewSingleHostReverseProxy(target), nil
	}
	sub, err := fs.Sub(webembed.DistFS, "dist")
	if err != nil {
		return nil, err
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if upath == "" || upath == "." {
			upath = "index.html"
		}
		if strings.HasPrefix(path.Base(upath), ".") {
			http.NotFound(w, r)
			return
		}
		if _, err := fs.Stat(sub, upath); err != nil {
			if strings.Contains(path.Base(upath), ".") {
				http.NotFound(w, r)
				return
			}
			// SPA route: fall back to index.html.
			upath = "index.html"
		}
		http.ServeFileFS(w, r, sub, upath)
	}), nil
}
