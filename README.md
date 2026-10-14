# Frame — Go media upload base app

A local video library with a Go backend and plain HTML, CSS, and JavaScript. No Python, external Go packages, or frontend build tools are required.

## Run

Install Go 1.22 or newer, then open a terminal in this folder:

```sh
go run .
```

Open http://127.0.0.1:8000. Stop with Ctrl+C. If the port is occupied:

```sh
go run . -port 8080
```

Videos are saved in `uploads/` relative to your working directory. To choose another folder:

```sh
go run . -uploads /path/to/videos
```

## Build a standalone app

macOS/Linux:

```sh
go build -o frame .
./frame
```

Windows:

```powershell
go build -o frame.exe .
.\frame.exe
```

The interface is embedded in the executable; no separate static folder is needed beside a built app. Build for the operating system where you will run it.

## Included

- Select videos or drag and drop them into the browser.
- Multiple selected files upload one after another.
- Files persist in one folder and appear in a searchable gallery after restart.
- Hover to silently preview the first 10 seconds; moving away resets the preview. Short videos stop at their end.
- Click for full video playback with sound and controls; close with X or Escape.
- File sizes, dates, upload results, and playback error messages.

Supported extensions: MP4, WebM, OGG, MOV, M4V. Maximum: 500 MB per file. Browser codec support determines playback; try H.264 MP4 or WebM. Extension checks do not validate the video contents. Files are saved without conversion or compression.

## What happens internally

1. `static/app.js` sends each selected file to `POST /api/videos` sequentially.
2. `main.go` streams the body to a temporary `.part` file, keeping memory use small.
3. Once complete, the file receives a random storage name. A JSON sidecar keeps the original filename. Repeated filenames do not overwrite earlier uploads.
4. `GET /api/videos` reads the saved library; incomplete uploads are excluded.
5. `GET /videos/<id>` uses Go's file serving with byte-range support for video playback and seeking.

No producer/consumer workers, user-created goroutines, queue, synchronization logic, p/c/q settings, gRPC, VMs, duplicate detection, or compression are included. Go's standard HTTP server internally manages connections concurrently; this app adds no concurrency exercise logic. Browser uploads are sequential. This is a local learning app, listening on `127.0.0.1` without authentication.

## Source files

- `main.go`: Go server, upload saving, library listing, and embedded interface.
- `static/index.html`: upload interface, gallery, and player dialog.
- `static/style.css`: responsive styling.
- `static/app.js`: uploads, search, hover previews, and playback.
- `main_test.go`: upload, persistence, range serving, validation, and route tests.
- `uploads/`: saved videos and original-name metadata.

## Verify

```sh
go test ./...
```

For a manual demo: upload a playable MP4, hover for 10 seconds, click to watch, search its filename, restart the app, and confirm it remains. Also try multiple uploads and an unsupported file type. Browser hover and actual codec playback require manual testing. Slides and a recorded demonstration are not included in this base app.
