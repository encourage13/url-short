package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/encourage13/url-short/internal/domain"
	"github.com/encourage13/url-short/internal/repository"
	"github.com/encourage13/url-short/internal/service"
)

func newTestServer(t *testing.T, adminToken string) *httptest.Server {
	t.Helper()
	svc := service.NewURLService(repository.NewInMemoryRepository(), 6)
	return serve(t, svc, adminToken)
}

var discardLog = slog.New(slog.NewTextHandler(io.Discard, nil))

func serve(t *testing.T, svc URLService, adminToken string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(NewURLHandler(svc, "https://sho.rt/", adminToken, discardLog).Routes())
	t.Cleanup(srv.Close)
	return srv
}

var noRedirect = &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}}

func shorten(t *testing.T, srv *httptest.Server, body string) *http.Response {
	t.Helper()
	resp, err := http.Post(srv.URL+"/shorten", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func TestShortenAndRedirect(t *testing.T) {
	srv := newTestServer(t, "")

	resp := shorten(t, srv, `{"url":"https://example.com/page"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var created URLResponse
	json.NewDecoder(resp.Body).Decode(&created)
	if created.ShortURL != "https://sho.rt/"+created.Code {
		t.Errorf("short_url = %q, want BASE_URL-based link", created.ShortURL)
	}

	r, err := noRedirect.Get(srv.URL + "/" + created.Code)
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if r.StatusCode != http.StatusFound || r.Header.Get("Location") != "https://example.com/page" {
		t.Errorf("redirect: status %d, location %q", r.StatusCode, r.Header.Get("Location"))
	}
}

func TestShorten_Rejects(t *testing.T) {
	srv := newTestServer(t, "")

	tests := map[string]string{
		"invalid url": `{"url":"javascript:alert(1)"}`,
		"empty url":   `{"url":""}`,
		"bad json":    `{`,
		"too large":   `{"url":"https://example.com/` + strings.Repeat("a", maxRequestBody) + `"}`,
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			if resp := shorten(t, srv, body); resp.StatusCode != http.StatusBadRequest {
				t.Errorf("status = %d, want 400", resp.StatusCode)
			}
		})
	}
}

func TestRouting(t *testing.T) {
	srv := newTestServer(t, "")

	tests := []struct {
		method, path string
		want         int
	}{
		{http.MethodPut, "/shorten", http.StatusMethodNotAllowed},
		{http.MethodPost, "/abc123", http.StatusMethodNotAllowed},
		{http.MethodGet, "/a/b", http.StatusNotFound},
		{http.MethodGet, "/nope", http.StatusNotFound},
		{http.MethodGet, "/info/nope", http.StatusNotFound},
	}
	for _, tt := range tests {
		req, _ := http.NewRequest(tt.method, srv.URL+tt.path, nil)
		resp, err := noRedirect.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != tt.want {
			t.Errorf("%s %s = %d, want %d", tt.method, tt.path, resp.StatusCode, tt.want)
		}
	}
}

func TestGetAllURLs_RequiresToken(t *testing.T) {
	tests := []struct {
		name, serverToken, header string
		want                      int
	}{
		{"disabled without token", "", "Bearer ", http.StatusUnauthorized},
		{"no header", "secret", "", http.StatusUnauthorized},
		{"wrong token", "secret", "Bearer nope", http.StatusUnauthorized},
		{"valid token", "secret", "Bearer secret", http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newTestServer(t, tt.serverToken)
			req, _ := http.NewRequest(http.MethodGet, srv.URL+"/urls", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			if resp.StatusCode != tt.want {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.want)
			}
		})
	}
}

func TestConcurrentRedirectAndInfo(t *testing.T) {
	srv := newTestServer(t, "secret")

	var created URLResponse
	json.NewDecoder(shorten(t, srv, `{"url":"https://example.com"}`).Body).Decode(&created)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			if r, err := noRedirect.Get(srv.URL + "/" + created.Code); err == nil {
				r.Body.Close()
			}
		}()
		go func() {
			defer wg.Done()
			if r, err := http.Get(srv.URL + "/info/" + created.Code); err == nil {
				r.Body.Close()
			}
		}()
		go func() {
			defer wg.Done()
			req, _ := http.NewRequest(http.MethodGet, srv.URL+"/urls", nil)
			req.Header.Set("Authorization", "Bearer secret")
			if r, err := http.DefaultClient.Do(req); err == nil {
				r.Body.Close()
			}
		}()
	}
	wg.Wait()
}

func TestListURLs_Pagination(t *testing.T) {
	srv := newTestServer(t, "secret")
	for i := 0; i < 3; i++ {
		shorten(t, srv, `{"url":"https://example.com"}`)
	}

	get := func(query string) (*http.Response, ListURLsResponse) {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/urls"+query, nil)
		req.Header.Set("Authorization", "Bearer secret")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var body ListURLsResponse
		json.NewDecoder(resp.Body).Decode(&body)
		return resp, body
	}

	if resp, body := get("?limit=2&offset=1"); resp.StatusCode != http.StatusOK || len(body.Items) != 2 || body.Limit != 2 || body.Offset != 1 {
		t.Errorf("limit=2&offset=1: status %d, body %+v", resp.StatusCode, body)
	}
	if _, body := get(""); len(body.Items) != 3 || body.Limit != defaultListLimit {
		t.Errorf("defaults: body %+v", body)
	}
	for _, q := range []string{"?limit=0", "?limit=101", "?limit=x", "?offset=-1"} {
		if resp, _ := get(q); resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400", q, resp.StatusCode)
		}
	}
}

type failingService struct{ URLService }

var errDBDown = errors.New("db down: secret connection details")

func (failingService) Health(context.Context) error { return errDBDown }

func (failingService) GetURLInfo(context.Context, string) (*domain.URL, error) {
	return nil, errDBDown
}

func TestStorageFailure(t *testing.T) {
	srv := serve(t, failingService{}, "")

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("/health status = %d, want 503", resp.StatusCode)
	}

	resp, err = http.Get(srv.URL + "/info/abc")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError || strings.Contains(string(body), "secret") {
		t.Errorf("/info: status %d, body %s - want 500 without internal details", resp.StatusCode, body)
	}
}
