// Frame is a local video upload app using only the Go standard library.
package main

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

//go:embed static/*
var assets embed.FS

const maxSize int64 = 500 * 1024 * 1024

var videoID = regexp.MustCompile(`^[a-f0-9]{32}\.(mp4|webm|ogg|mov|m4v)$`)
var extensions = map[string]bool{".mp4": true, ".webm": true, ".ogg": true, ".mov": true, ".m4v": true}

type video struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	Uploaded int64  `json:"uploaded"`
	URL      string `json:"url"`
}
type app struct{ uploads string }

func respond(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func fail(w http.ResponseWriter, status int, message string) {
	respond(w, status, map[string]string{"error": message})
}
func (a app) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api/videos" {
		switch r.Method {
		case http.MethodGet:
			a.list(w, r)
		case http.MethodPost:
			a.upload(w, r)
		default:
			w.Header().Set("Allow", "GET, POST")
			fail(w, 405, "Method not allowed")
		}
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		fail(w, 405, "Method not allowed")
		return
	}
	if strings.HasPrefix(r.URL.Path, "/videos/") {
		id := strings.TrimPrefix(r.URL.Path, "/videos/")
		if !videoID.MatchString(id) {
			fail(w, 404, "Not found")
			return
		}
		http.ServeFile(w, r, filepath.Join(a.uploads, id))
		return
	}
	names := map[string]string{"/": "index.html", "/app.js": "app.js", "/style.css": "style.css"}
	name, ok := names[r.URL.Path]
	if !ok {
		fail(w, 404, "Not found")
		return
	}
	data, err := assets.ReadFile("static/" + name)
	if err != nil {
		fail(w, 500, "Could not load page")
		return
	}
	types := map[string]string{"index.html": "text/html; charset=utf-8", "app.js": "text/javascript; charset=utf-8", "style.css": "text/css; charset=utf-8"}
	w.Header().Set("Content-Type", types[name])
	http.ServeContent(w, r, name, time.Time{}, strings.NewReader(string(data)))
}
func (a app) list(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(a.uploads)
	if err != nil {
		fail(w, 500, "Could not read library")
		return
	}
	videos := []video{}
	for _, entry := range entries {
		if entry.IsDir() || !videoID.MatchString(entry.Name()) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		item := video{ID: entry.Name(), Name: entry.Name(), Size: info.Size(), Uploaded: info.ModTime().Unix(), URL: "/videos/" + entry.Name()}
		metadata, err := os.ReadFile(filepath.Join(a.uploads, entry.Name()+".json"))
		var original struct {
			Name string `json:"name"`
		}
		if err == nil && json.Unmarshal(metadata, &original) == nil && original.Name != "" {
			item.Name = original.Name
		}
		videos = append(videos, item)
	}
	sort.Slice(videos, func(i, j int) bool {
		if videos[i].Uploaded == videos[j].Uploaded {
			return videos[i].ID < videos[j].ID
		}
		return videos[i].Uploaded > videos[j].Uploaded
	})
	respond(w, 200, videos)
}
func (a app) upload(w http.ResponseWriter, r *http.Request) {
	name, err := url.PathUnescape(r.Header.Get("X-Filename"))
	if err != nil {
		fail(w, 400, "Invalid filename")
		return
	}
	name = filepath.Base(strings.ReplaceAll(name, `\`, "/"))
	extension := strings.ToLower(filepath.Ext(name))
	if !extensions[extension] {
		fail(w, 415, "Choose an MP4, WebM, OGG, MOV, or M4V video.")
		return
	}
	if r.ContentLength <= 0 || r.ContentLength > maxSize {
		fail(w, 413, "Video must be nonempty and no larger than 500 MB.")
		return
	}
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		fail(w, 500, "Could not create filename")
		return
	}
	id := hex.EncodeToString(random[:]) + extension
	target := filepath.Join(a.uploads, id)
	temporary := target + ".part"
	file, err := os.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		fail(w, 500, "Could not save video")
		return
	}
	defer os.Remove(temporary)
	written, copyErr := io.Copy(file, http.MaxBytesReader(w, r.Body, maxSize))
	closeErr := file.Close()
	if copyErr != nil || closeErr != nil || written != r.ContentLength {
		fail(w, 400, "Upload incomplete or could not be saved. Try again.")
		return
	}
	metadata, _ := json.Marshal(map[string]string{"name": name})
	if err := os.WriteFile(target+".json", metadata, 0600); err != nil {
		fail(w, 500, "Could not save video details")
		return
	}
	if err := os.Rename(temporary, target); err != nil {
		_ = os.Remove(target + ".json")
		fail(w, 500, "Could not finish saving video")
		return
	}
	respond(w, 201, map[string]string{"id": id, "name": name})
}
func main() {
	port := flag.Int("port", 8000, "Local HTTP port")
	folder := flag.String("uploads", "uploads", "Folder for saved videos")
	flag.Parse()
	if err := os.MkdirAll(*folder, 0700); err != nil {
		log.Fatal(err)
	}
	server := &http.Server{Addr: fmt.Sprintf("127.0.0.1:%d", *port), Handler: app{uploads: *folder}, ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second}
	fmt.Printf("Frame running at http://127.0.0.1:%d (Ctrl+C to stop)\n", *port)
	log.Fatal(server.ListenAndServe())
}
