# Frame — Media upload base app

A beginner-friendly local starting point for the producer–consumer exercise.

## Run

Install Python 3.10 or newer. No packages, build step, or compilation needed.

```sh
cd media-upload-base
python3 server.py
```

Open http://127.0.0.1:8000 in your browser. Stop the server with Ctrl+C.
If port 8000 is busy, use `python3 server.py --port 8080`.

## Features

- Upload one or multiple videos through a file picker or drag and drop.
- Save all uploaded videos in the `uploads/` folder; the library survives restarts.
- Search the library by original filename.
- Hover over a card to play its first 10 seconds silently, then stop and reset. Shorter videos stop at their end.
- Click a card to open the full video with sound and playback controls. Close with the X button or Escape.
- Display upload results, file sizes, and dates.

MP4, WebM, OGG, MOV, and M4V files are accepted, up to 500 MB each. Playback depends on your browser's supported codecs; H.264 MP4 is a good choice. Files are preserved as uploaded. Extension checks do not verify the video contents.

## How it works

The browser sends each chosen file sequentially to `POST /api/videos`. The Python server reads the request in chunks and writes it to a temporary `.part` file. Only completed uploads appear in the library. A random filename avoids accidental overwrites, while a small JSON sidecar preserves the original name. `GET /api/videos` lists saved videos. `GET /videos/<filename>` serves video bytes with HTTP Range support for playback and seeking.

`HTTPServer` handles one request at a time. No threads, worker pools, queues, synchronization, producer instances, gRPC, separate VMs, duplicate detection, or compression are implemented. A long upload can delay playback or library loading; that is intentional for this sequential base version. The p, c, and q inputs belong to the later concurrency version and are omitted here.

## Source layout

- `server.py` — local server, upload saving, library listing, and video serving.
- `static/index.html` — gallery, upload interface, and video dialog.
- `static/style.css` — responsive visual styling.
- `static/app.js` — sequential uploads, search, hover previews, and playback.
- `uploads/` — locally saved videos and filename metadata.

The server listens only on this computer. This is a local learning app, without authentication or production deployment configuration.

## Manual demo checklist

1. Start with an empty library and upload a playable MP4.
2. Check that the video appears and its file is saved in `uploads/`.
3. Hover: preview stops and resets at 10 seconds. Leaving the card resets sooner.
4. Click: full playback opens with controls; Escape closes it.
5. Upload several videos: they are sent one after another.
6. Restart the server: saved videos remain available.
7. Search by filename and try a search with no matches.
8. Try a disallowed file type and an empty video file: errors appear.

The video demonstration and slides can be added when the final test cases and concurrency requirements are introduced.
