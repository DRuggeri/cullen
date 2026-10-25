# Cullen

Cullen is a small web application for culling a folder of images, videos and audio files. Browse to a folder, open a file, then **Keep** or **Discard** it. Discarding deletes the file (or moves it to another folder) and moves straight on to the next one. It is built for working through a large pile of media quickly, from a desktop or a phone.

It is a single Go binary with the web UI embedded. There is no database and no external assets.

> **By default, Discard permanently deletes files** (`os.Remove`, no trash or undo). Use `--discard move:/path` for a recoverable trash folder. Point Cullen at a copy, or at a folder you are happy to prune.

## Features

- Browse subfolders of a root directory and list supported media in each.
- Culling view for images, video and audio:
  - Images are shown fitted to the screen.
  - Video and audio autoplay with the browser's standard controls (play/pause, seek, volume).
  - Large **Discard** and **Keep** buttons along the bottom.
- **Discard** sends an async `POST` to delete the file (or move it to another folder, configurable), then advances to the next file.
- **Keep** either just advances, or moves the file to another folder (configurable).
- Sort by **Modified** time (default, newest first) or **Name**, ascending or descending. The culling order follows the list order, and the choice is remembered in the browser.
- Deep linking: the current folder is in the URL, and back/forward work.
- Touch and mouse support: swipe gestures, pinch zoom and pan, mouse wheel zoom and middle-button pan.
- Mobile-friendly layout with large tap targets.

## Requirements

- Go 1.22 or newer.
- A modern browser. Which formats play depends on your browser's codecs.

## Install and run

```sh
git clone https://github.com/DRuggeri/cullen
cd cullen
go run . --path /path/to/media
```

Then open <http://localhost:8080/cullen/>.

To build a binary:

```sh
go build -o cullen .
./cullen --path /path/to/media --listen 0.0.0.0:8080
```

The static files are embedded at build time, so rebuild or restart after changing anything under `static/`.

## Command-line options

| Flag | Default | Description |
| --- | --- | --- |
| `--path` | `/var/www/html` | Root directory containing the media. Must exist. Nothing outside it is served or deleted. |
| `--listen` | `:8080` | Address to listen on, for example `0.0.0.0:8085` or `127.0.0.1:9000`. |
| `--baseurl` | `/cullen` | URL prefix for every endpoint. Use an empty string to serve at `/`. |
| `--keep` | `next` | What **Keep** does: `next` or `move:/path/to/folder` (see below). |
| `--discard` | `delete` | What **Discard** does: `delete` or `move:/path/to/folder` (see below). |

### Discard modes

- `--discard delete` (default): Discard permanently deletes the file, then shows the next one.
- `--discard move:/path/to/folder`: Discard moves the file into that folder instead. The folder must already exist.

Moves behave like Keep moves (below). Example: `cullen --path ~/inbox --keep move:$HOME/keepers --discard move:$HOME/trash`

### Keep modes

- `--keep next` (default): Keep only advances to the next file. Nothing changes on disk.
- `--keep move:/path/to/folder`: Keep moves the file into that folder, then shows the next file. The folder must already exist.
  - The destination can be anywhere, including outside `--path`.
  - Existing files are never overwritten: a name clash gets a numeric suffix (`photo.jpg` becomes `photo-1.jpg`).
  - The modified time is preserved, and moves across filesystems work.

Example: cull `~/inbox`, keeping good files in `~/keepers`:

```sh
cullen --path ~/inbox --keep move:$HOME/keepers
```

## Using the UI

### File list

- Click a folder to enter it. **Up** goes to the parent.
- The **Modified** and **Name** buttons set the sort. Tap the active one again to flip direction (↓ or ↑).
- Folders are always listed first. Hidden files (names starting with `.`) are skipped, and unsupported files are shown greyed out.
- Click a supported file to open the culling view. Navigation order is the displayed order.

### Culling view

| Action | Touch | Mouse / keyboard |
| --- | --- | --- |
| Discard (delete or move, then next) | Discard button | Discard button, `Delete` key |
| Keep | Keep button | Keep button |
| Next file (no delete or move) | Swipe left | `→` |
| Previous file | Swipe right | `←` |
| Back to the list | Swipe up | Close button, `Esc`, browser back |
| Zoom | Pinch | Mouse wheel (zooms at the cursor) |
| Pan (when zoomed) | One-finger drag | Middle-button drag |

Notes:

- Swiping or pressing `→` past the last file, or `←` before the first, returns to the list. Keeping or discarding the last file does too.
- Next and previous only navigate. They never delete or move files, even in `move` mode.
- Zoom ranges from 0.25× to 20×, where 1× means fitted to the screen. Zoom resets when you change file.
- On video and audio, gestures that start on the bottom strip are ignored so the seek bar keeps working.
- The browser back button closes the culling view first, then steps back through folders.

### Deep links

The current folder is kept in the URL hash, for example `http://host:8080/cullen/#/trips/2024/day1`. Open or bookmark such a link to land directly in that folder. Unknown folders fall back to the root.

## Supported file types

Files are recognised by extension:

| Type | Extensions |
| --- | --- |
| Image | `.jpg` `.jpeg` `.png` `.gif` `.webp` `.avif` `.bmp` `.svg` |
| Video | `.mp4` `.m4v` `.webm` `.ogv` `.mov` |
| Audio | `.mp3` `.wav` `.ogg` `.oga` `.m4a` `.aac` `.flac` `.opus` |

A listed file may still not play if your browser lacks the codec (for example HEVC video or some `.mov` files).

## HTTP API

All endpoints are under `--baseurl`. Paths are relative to `--path` and use `/` separators.

| Method | Endpoint | Description |
| --- | --- | --- |
| `GET` | `/` | The embedded web UI. |
| `GET` | `/api/list?path=<dir>` | JSON array of entries: `name`, `is_dir`, `rel_path`, `type` (`image`, `video`, `audio` or empty), `modified` (Unix ms). |
| `GET` | `/api/config` | JSON `{"keep": "next" or "move", "discard": "delete" or "move"}`. |
| `GET` | `/media/<path>` | Streams a supported media file, with range requests for seeking. |
| `POST` | `/api/delete` | Body `{"path": "<file>"}`. Discards one supported media file: deletes it, or moves it in `--discard move:` mode. |
| `POST` | `/api/keep` | Body `{"path": "<file>"}`. Moves the file in `move` mode, otherwise a no-op. |

## Security

Cullen has **no authentication**. Anyone who can reach the port can browse, view and delete files under `--path`. Bind to `127.0.0.1`, or put it behind a reverse proxy with authentication, TLS and access control if it is reachable by others.

Built-in safeguards:

- Every path is resolved and checked to stay inside `--path`, including through symlinks. `../` and absolute paths are rejected.
- Delete and move only act on regular files with a supported media extension. They never remove directories.
- Only supported media types are served from `/media`.
- Request bodies are capped at 1 MiB.

## Project layout

```
main.go          CLI flags and startup validation
server.go        HTTP server, API handlers, path safety, file move
static/index.html  Single-page UI (HTML, CSS, JS), embedded into the binary
tmp/             Sample files for manual testing
```

## Development

```sh
go vet ./...
go build ./...
go run . --path tmp --listen 127.0.0.1:8080
```

The UI is plain HTML, CSS and JavaScript in [static/index.html](static/index.html), with no build step or dependencies.
