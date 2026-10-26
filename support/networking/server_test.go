package networking

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testEndpoint struct {
	path      string
	authLevel AuthLevel
	body      string
}

func (e testEndpoint) GetHandlerFunc() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(e.body))
	}
}

func (e testEndpoint) GetAuthLevel() AuthLevel {
	return e.authLevel
}

func (e testEndpoint) GetPath() string {
	return e.path
}

func serve(t *testing.T, ws WebServer, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	ws.(*server).router.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
	return rec
}

func TestMakeServerWithGoogleAuth_NoAuthEndpoint(t *testing.T) {
	ws, e := MakeServerWithGoogleAuth(&Config{}, []Endpoint{
		testEndpoint{path: "/health", authLevel: NoAuth, body: "ok"},
	})
	if !assert.NoError(t, e) {
		return
	}

	rec := serve(t, ws, "/health")
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())

	// nothing is registered on other paths when no endpoint needs google auth
	assert.Equal(t, http.StatusNotFound, serve(t, ws, "/metrics").Code)
}

func TestMakeServerWithGoogleAuth_MissingCredentials(t *testing.T) {
	for _, cfg := range []*Config{
		{},
		{GoogleClientID: "id"},
		{GoogleClientSecret: "secret"},
	} {
		_, e := MakeServerWithGoogleAuth(cfg, []Endpoint{
			testEndpoint{path: "/metrics", authLevel: GoogleAuth, body: "secret metrics"},
		})
		assert.Error(t, e, "config %+v", cfg)
	}
}

func TestMakeServerWithGoogleAuth_GoogleAuthEndpoint(t *testing.T) {
	ws, e := MakeServerWithGoogleAuth(&Config{
		GoogleClientID:     "id",
		GoogleClientSecret: "secret",
		PermittedEmails:    map[string]bool{"someone@example.com": true},
	}, []Endpoint{
		testEndpoint{path: "/health", authLevel: NoAuth, body: "ok"},
		testEndpoint{path: "/metrics", authLevel: GoogleAuth, body: "secret metrics"},
	})
	if !assert.NoError(t, e) {
		return
	}

	// the unauthenticated endpoint is still served directly
	assert.Equal(t, "ok", serve(t, ws, "/health").Body.String())

	// the protected endpoint sends an unauthenticated request to sign in instead of serving the content
	rec := serve(t, ws, "/metrics")
	assert.NotEqual(t, http.StatusOK, rec.Code)
	assert.NotContains(t, rec.Body.String(), "secret metrics")
}

func TestStartServer_MissingTLSFiles(t *testing.T) {
	dir := t.TempDir()
	e := startServer(http.NewServeMux(), 0, dir+"/missing.crt", dir+"/missing.key")
	if assert.Error(t, e) {
		assert.Contains(t, e.Error(), "tls cert file cannot be found")
	}
}
