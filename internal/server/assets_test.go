package server

import (
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	webembed "caddy-configurator/web"
)

func fsSub() (fs.FS, error) {
	return fs.Sub(webembed.DistFS, "dist")
}

// Requires `npm run build` output in web/dist (task 1.2 verification).
func TestFrontendServesBuiltIndex(t *testing.T) {
	h, err := NewFrontendHandler("")
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	res := rec.Result()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET / = %d, want 200", res.StatusCode)
	}
	if !strings.Contains(string(body), `id="root"`) {
		t.Fatalf("index does not contain vite root div; not the built SPA?")
	}
}

func TestFrontendDotfilesHidden(t *testing.T) {
	h, err := NewFrontendHandler("")
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/.gitkeep", nil))
	if rec.Result().StatusCode != http.StatusNotFound {
		t.Fatalf("GET /.gitkeep = %d, want 404", rec.Result().StatusCode)
	}
}

func TestFrontendSpaFallback(t *testing.T) {
	h, err := NewFrontendHandler("")
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sites/abc", nil))
	if rec.Result().StatusCode != http.StatusOK {
		t.Fatalf("GET /sites/abc = %d, want 200 (index fallback)", rec.Result().StatusCode)
	}
}

func TestFrontendBundleIsApp(t *testing.T) {
	// Guards against shipping the placeholder: the production bundle must
	// contain API calls and UI strings from tasks 6.1-6.3.
	sub, err := fsSub()
	if err != nil {
		t.Fatal(err)
	}
	entries, err := fs.ReadDir(sub, "assets")
	if err != nil {
		t.Fatal(err)
	}
	var js []byte
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".js") {
			js, err = fs.ReadFile(sub, "assets/"+e.Name())
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if len(js) == 0 {
		t.Fatal("no JS bundle in web/dist/assets")
	}
	for _, marker := range []string{"api/v1/preview", "New site", "Caddy snippet", "Changed on disk"} {
		if !strings.Contains(string(js), marker) {
			t.Fatalf("bundle lacks %q: UI from 6.x not embedded?", marker)
		}
	}
}

func TestFrontendDevProxy(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("vite-dev:" + r.URL.Path))
	}))
	defer backend.Close()
	h, err := NewFrontendHandler(backend.URL)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/src/main.tsx", nil))
	body, _ := io.ReadAll(rec.Result().Body)
	if string(body) != "vite-dev:/src/main.tsx" {
		t.Fatalf("dev proxy body = %q", body)
	}
}
