package main

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUploadAndPlayback(t *testing.T) {
	folder := t.TempDir()
	handler := app{uploads: folder}
	send := func(method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		for k, v := range headers {
			r.Header.Set(k, v)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	if w := send("GET", "/api/videos", "", nil); w.Code != 200 || strings.TrimSpace(w.Body.String()) != "[]" {
		t.Fatal(w)
	}
	payload := "example-video-bytes"
	w := send("POST", "/api/videos", payload, map[string]string{"X-Filename": "My%20Video.mp4"})
	if w.Code != 201 {
		t.Fatal(w)
	}
	var uploaded struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &uploaded); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(folder, uploaded.ID))
	if err != nil || string(data) != payload {
		t.Fatal("file not saved correctly", err)
	}
	// A new app instance reads the same persisted library.
	listing := httptest.NewRecorder()
	app{uploads: folder}.ServeHTTP(listing, httptest.NewRequest("GET", "/api/videos", nil))
	var videos []video
	if err := json.Unmarshal(listing.Body.Bytes(), &videos); err != nil || len(videos) != 1 || videos[0].Name != "My Video.mp4" {
		t.Fatal(listing.Body.String(), err)
	}
	for _, test := range []struct{ rangeHeader, expected string }{{"bytes=2-6", payload[2:7]}, {"bytes=-4", payload[len(payload)-4:]}} {
		w := send("GET", "/videos/"+uploaded.ID, "", map[string]string{"Range": test.rangeHeader})
		if w.Code != 206 || w.Body.String() != test.expected {
			t.Fatal(w)
		}
	}
	for _, test := range []struct {
		method, path, body, name string
		status                   int
	}{
		{"POST", "/api/videos", "text", "bad.txt", 415},
		{"POST", "/api/videos", "", "empty.mp4", 413},
		{"GET", "/videos/../../main.go", "", "", 404},
		{"DELETE", "/api/videos", "", "", 405},
		{"GET", "/missing", "", "", 404},
	} {
		w := send(test.method, test.path, test.body, map[string]string{"X-Filename": test.name})
		if w.Code != test.status {
			t.Fatalf("%s: got %d want %d", test.path, w.Code, test.status)
		}
	}
	w = send("GET", "/videos/"+uploaded.ID, "", map[string]string{"Range": "bytes=9999-"})
	if w.Code != 416 {
		t.Fatal(w)
	}
	for _, path := range []string{"/", "/app.js", "/style.css"} {
		if w := send("GET", path, "", nil); w.Code != 200 {
			t.Fatal(path, w)
		}
	}
}
