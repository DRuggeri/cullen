package main

import (
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed static/*
var staticFiles embed.FS

type FileInfo struct {
	Name    string `json:"name"`
	IsDir   bool   `json:"is_dir"`
	RelPath string `json:"rel_path"`
	// Type is "image", "video", "audio", or empty when unsupported.
	Type string `json:"type"`
	// Modified is the modification time in Unix milliseconds.
	Modified int64 `json:"modified"`
}

var mediaTypes = map[string]string{
	".jpg": "image", ".jpeg": "image", ".png": "image", ".gif": "image",
	".webp": "image", ".avif": "image", ".bmp": "image", ".svg": "image",
	".mp4": "video", ".m4v": "video", ".webm": "video", ".ogv": "video", ".mov": "video",
	".mp3": "audio", ".wav": "audio", ".ogg": "audio", ".oga": "audio",
	".m4a": "audio", ".aac": "audio", ".flac": "audio", ".opus": "audio",
}

func mediaType(name string) string {
	return mediaTypes[strings.ToLower(filepath.Ext(name))]
}

// resolve maps a client-supplied relative path to an absolute path that is
// guaranteed to stay inside baseDir (including via symlinks).
func resolve(rel string) (string, error) {
	abs := filepath.Join(baseDir, filepath.Clean("/"+filepath.FromSlash(rel)))
	root, err := filepath.EvalSymlinks(baseDir)
	if err != nil {
		return "", err
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	if real != root && !strings.HasPrefix(real, root+string(filepath.Separator)) {
		return "", errors.New("path escapes base directory")
	}
	return abs, nil
}

func startServer() error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+baseURL+"/api/list", listHandler)
	mux.HandleFunc("POST "+baseURL+"/api/delete", deleteHandler)
	mux.HandleFunc("GET "+baseURL+"/media/{path...}", mediaHandler)

	subFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return err
	}
	static := http.StripPrefix(baseURL+"/", http.FileServer(http.FS(subFS)))
	mux.HandleFunc("GET "+baseURL+"/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		static.ServeHTTP(w, r)
	})

	logged := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info("incoming request", "method", r.Method, "uri", r.URL.Path)
		mux.ServeHTTP(w, r)
	})

	srv := &http.Server{Addr: listenAddr, Handler: logged}
	slog.Info("server starting", "address", listenAddr, "baseURL", baseURL, "path", baseDir)
	return srv.ListenAndServe()
}

func mediaHandler(w http.ResponseWriter, r *http.Request) {
	abs, err := resolve(r.PathValue("path"))
	if err != nil || mediaType(abs) == "" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, abs) // supports range requests for seeking
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	rel := strings.Trim(filepath.ToSlash(filepath.Clean("/"+r.URL.Query().Get("path"))), "/")
	abs, err := resolve(rel)
	if err != nil {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	files := []FileInfo{}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		isDir := e.IsDir()
		info, err := e.Info()
		if err != nil {
			continue // removed since ReadDir
		}
		if e.Type()&fs.ModeSymlink != 0 {
			if st, err := os.Stat(filepath.Join(abs, e.Name())); err == nil {
				isDir = st.IsDir()
				info = st
			}
		}
		files = append(files, FileInfo{
			Name:    e.Name(),
			IsDir:   isDir,
			RelPath: filepath.ToSlash(filepath.Join(rel, e.Name())),
			Type:    mediaType(e.Name()),

			Modified: info.ModTime().UnixMilli(),
		})
	}
	sort.SliceStable(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	abs, err := resolve(payload.Path)
	if err != nil {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	st, err := os.Lstat(abs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if !st.Mode().IsRegular() || mediaType(abs) == "" {
		http.Error(w, "only supported media files can be deleted", http.StatusBadRequest)
		return
	}
	if err := os.Remove(abs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	slog.Info("deleted", "path", abs)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}
