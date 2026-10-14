"""Single-process local video library. Python 3.10+, no dependencies."""
from http.server import HTTPServer, BaseHTTPRequestHandler
from pathlib import Path
from urllib.parse import urlsplit, unquote
import argparse
import json
import mimetypes
import re
import uuid

ROOT = Path(__file__).resolve().parent
UPLOADS = ROOT / 'uploads'
EXTENSIONS = {'.mp4', '.webm', '.ogg', '.mov', '.m4v'}
MAX_SIZE = 500 * 1024 * 1024

class Handler(BaseHTTPRequestHandler):
    def json_response(self, status, value):
        body = json.dumps(value).encode()
        self.send_response(status)
        self.send_header('Content-Type', 'application/json')
        self.send_header('Content-Length', str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_POST(self):
        if self.path != '/api/videos':
            return self.json_response(404, {'error': 'Not found'})
        name = Path(unquote(self.headers.get('X-Filename', '')).replace('\\', '/')).name
        extension = Path(name).suffix.lower()
        try:
            size = int(self.headers.get('Content-Length', '0'))
        except ValueError:
            size = 0
        if extension not in EXTENSIONS:
            self.close_connection = True
            return self.json_response(415, {'error': 'Choose an MP4, WebM, OGG, MOV, or M4V video.'})
        if not 0 < size <= MAX_SIZE:
            self.close_connection = True
            return self.json_response(413, {'error': 'Video must be nonempty and no larger than 500 MB.'})
        identifier = uuid.uuid4().hex
        target = UPLOADS / (identifier + extension)
        temporary = target.with_suffix('.part')
        try:
            remaining = size
            with temporary.open('wb') as output:
                while remaining:
                    chunk = self.rfile.read(min(1024 * 1024, remaining))
                    if not chunk:
                        raise OSError('Upload interrupted')
                    output.write(chunk)
                    remaining -= len(chunk)
            target.with_suffix('.json').write_text(json.dumps({'name': name}), encoding='utf-8')
            temporary.rename(target)
        except OSError:
            temporary.unlink(missing_ok=True)
            target.with_suffix('.json').unlink(missing_ok=True)
            return self.json_response(500, {'error': 'Could not save the video. Check disk space and try again.'})
        self.json_response(201, {'id': target.name, 'name': name})

    def do_GET(self):
        path = unquote(urlsplit(self.path).path)
        if path == '/api/videos':
            videos = []
            for file in sorted(UPLOADS.iterdir(), key=lambda p: p.stat().st_mtime, reverse=True):
                if file.suffix.lower() in EXTENSIONS:
                    try:
                        name = json.loads(file.with_suffix('.json').read_text(encoding='utf-8'))['name']
                    except (OSError, ValueError, KeyError):
                        name = file.name
                    videos.append({'id': file.name, 'name': name, 'size': file.stat().st_size,
                                   'uploaded': file.stat().st_mtime, 'url': '/videos/' + file.name})
            return self.json_response(200, videos)
        if path.startswith('/videos/'):
            filename = path.removeprefix('/videos/')
            if not re.fullmatch(r'[a-f0-9]{32}\.(mp4|webm|ogg|mov|m4v)', filename):
                return self.json_response(404, {'error': 'Not found'})
            file = UPLOADS / filename
        else:
            names = {'/': 'index.html', '/app.js': 'app.js', '/style.css': 'style.css'}
            if path not in names:
                return self.json_response(404, {'error': 'Not found'})
            file = ROOT / 'static' / names[path]
        if not file.is_file():
            return self.json_response(404, {'error': 'Not found'})
        total = file.stat().st_size
        start, end, status = 0, total - 1, 200
        requested = self.headers.get('Range')
        if requested:
            match = re.fullmatch(r'bytes=(\d*)-(\d*)', requested)
            if not match or not any(match.groups()):
                return self.range_error(total)
            left, right = match.groups()
            if left:
                start = int(left)
                end = min(int(right), total - 1) if right else total - 1
            else:
                start = max(0, total - int(right))
            if start > end or start >= total:
                return self.range_error(total)
            status = 206
        self.send_response(status)
        self.send_header('Content-Type', mimetypes.guess_type(file.name)[0] or 'application/octet-stream')
        self.send_header('Content-Length', str(end - start + 1))
        self.send_header('Accept-Ranges', 'bytes')
        if status == 206:
            self.send_header('Content-Range', f'bytes {start}-{end}/{total}')
        self.end_headers()
        try:
            with file.open('rb') as stream:
                stream.seek(start)
                remaining = end - start + 1
                while remaining > 0:
                    chunk = stream.read(min(65536, remaining))
                    if not chunk:
                        break
                    self.wfile.write(chunk)
                    remaining -= len(chunk)
        except (BrokenPipeError, ConnectionResetError):
            pass

    def range_error(self, size):
        self.send_response(416)
        self.send_header('Content-Range', f'bytes */{size}')
        self.send_header('Content-Length', '0')
        self.end_headers()

if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='Run the local media upload base app')
    parser.add_argument('--port', type=int, default=8000)
    args = parser.parse_args()
    UPLOADS.mkdir(exist_ok=True)
    server = HTTPServer(('127.0.0.1', args.port), Handler)
    server.timeout = 30
    print(f'Media library running at http://127.0.0.1:{args.port}', flush=True)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print('\nServer stopped.')
    finally:
        server.server_close()
