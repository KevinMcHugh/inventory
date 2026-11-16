package idp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"
)

// fakeGoogleToken is Google's token endpoint, accepting only client:secret
// sent as HTTP Basic.
func fakeGoogleToken(t *testing.T) *httptest.Server {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, secret, _ := r.BasicAuth()
		if r.URL.Path != "/token" || id != "client" || secret != "secret" || r.FormValue("client_secret") != "" || r.FormValue("code") != "c" {
			http.Error(w, `{"error":"invalid_client"}`, http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "at"})
	}))
	t.Cleanup(ts.Close)
	return ts
}

// fakeConnector is a Sprites custom API connector in front of upstream: it
// fails the test if the server sent a credential of its own, and adds the
// Basic credential it holds.
func fakeConnector(t *testing.T, upstream string) *httptest.Server {
	target, err := url.Parse(upstream)
	if err != nil {
		t.Fatal(err)
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a := r.Header.Get("Authorization"); a != "" {
			t.Errorf("the server sent a credential to the connector: %q", a)
		}
		r.SetBasicAuth("client", "secret")
		proxy.ServeHTTP(w, r)
	}))
	t.Cleanup(ts.Close)
	return ts
}

func TestGoogleToken(t *testing.T) {
	google := fakeGoogleToken(t)
	connector := fakeConnector(t, google.URL)
	for name, cfg := range map[string]Config{
		"secret held by the server": {GoogleClientID: "client", GoogleClientSecret: "secret", GoogleTokenURL: google.URL + "/token"},
		"secret in a connector":     {GoogleClientID: "client", GoogleTokenURL: connector.URL + "/token"},
	} {
		t.Run(name, func(t *testing.T) {
			cfg.Issuer = "https://inventory.example"
			h := &Handler{Cfg: cfg}
			if !h.GoogleEnabled() {
				t.Fatal("not enabled")
			}
			body, err := h.googleToken(context.Background(), "c")
			if err != nil {
				t.Fatal(err)
			}
			var tok struct {
				AccessToken string `json:"access_token"`
			}
			if json.Unmarshal(body, &tok); tok.AccessToken != "at" {
				t.Fatalf("got %s", body)
			}
		})
	}
	if (&Handler{Cfg: Config{GoogleClientID: "client"}}).GoogleEnabled() {
		t.Fatal("enabled with no secret and no connector")
	}
}
