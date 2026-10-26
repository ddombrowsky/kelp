package networking

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func makeTestServer(contentType string, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		_, _ = w.Write([]byte(body))
	}))
}

func TestJSONRequest_ParsesResponse(t *testing.T) {
	for _, contentType := range []string{"application/json", "application/hal+json; charset=utf-8"} {
		ts := makeTestServer(contentType, `{"name":"kelp","count":3}`)

		var output struct {
			Name  string `json:"name"`
			Count int    `json:"count"`
		}
		e := JSONRequest(http.DefaultClient, "GET", ts.URL, "", map[string]string{}, &output, "error")
		ts.Close()

		if assert.NoError(t, e, contentType) {
			assert.Equal(t, "kelp", output.Name)
			assert.Equal(t, 3, output.Count)
		}
	}
}

func TestJSONRequest_SendsHeadersAndBody(t *testing.T) {
	var gotHeader, gotBody string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Test")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	e := JSONRequest(http.DefaultClient, "POST", ts.URL, `{"a":1}`, map[string]string{"X-Test": "value"}, nil, "")
	assert.NoError(t, e)
	assert.Equal(t, "value", gotHeader)
	assert.Equal(t, `{"a":1}`, gotBody)
}

func TestJSONRequest_Errors(t *testing.T) {
	testCases := []struct {
		name        string
		contentType string
		body        string
		errorKey    string
		wantErr     string
	}{
		{
			name:        "non-json content type",
			contentType: "text/html",
			body:        "<html></html>",
			wantErr:     "invalid 'Content-Type' header",
		}, {
			name:        "error key present in response",
			contentType: "application/json",
			body:        `{"error":"bad request"}`,
			errorKey:    "error",
			wantErr:     "error in response",
		}, {
			name:        "malformed json",
			contentType: "application/json",
			body:        `{"name":`,
			wantErr:     "could not unmarshall",
		},
	}

	for _, k := range testCases {
		t.Run(k.name, func(t *testing.T) {
			ts := makeTestServer(k.contentType, k.body)
			defer ts.Close()

			var output map[string]interface{}
			e := JSONRequest(http.DefaultClient, "GET", ts.URL, "", map[string]string{}, &output, k.errorKey)
			if assert.Error(t, e) {
				assert.Contains(t, e.Error(), k.wantErr)
			}
		})
	}
}
