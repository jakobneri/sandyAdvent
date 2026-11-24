package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"
)

// staticFiles serves the embedded static dir with content-hash cache busting:
// templates link /static/css/app.css?v=<hash>; a matching ?v= is cached
// for a year (immutable), everything else briefly.
type staticFiles struct {
	fsys   fs.FS
	hashes map[string]string
}

func newStaticFiles(fsys fs.FS) (*staticFiles, error) {
	sf := &staticFiles{fsys: fsys, hashes: map[string]string{}}
	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(b)
		sf.hashes[p] = hex.EncodeToString(sum[:])[:12]
		return nil
	})
	return sf, err
}

// URL returns the versioned URL of a static file.
func (sf *staticFiles) URL(p string) string {
	if h, ok := sf.hashes[p]; ok {
		return "/static/" + p + "?v=" + h
	}
	return "/static/" + p
}

func (sf *staticFiles) handler(notFound http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.PathValue("path")
		hash, ok := sf.hashes[p]
		if !ok || path.Clean(p) != p {
			notFound(w, r)
			return
		}
		b, err := fs.ReadFile(sf.fsys, p)
		if err != nil {
			notFound(w, r)
			return
		}
		h := w.Header()
		switch {
		case r.URL.Query().Get("v") == hash:
			h.Set("Cache-Control", "public, max-age=31536000, immutable")
		case strings.HasPrefix(p, "fonts/"):
			h.Set("Cache-Control", "public, max-age=2592000")
		default:
			h.Set("Cache-Control", "public, max-age=3600")
		}
		h.Set("ETag", `"`+hash+`"`)
		if ct := contentType(p); ct != "" {
			h.Set("Content-Type", ct)
		}
		http.ServeContent(w, r, p, time.Time{}, bytes.NewReader(b))
	})
}

// contentTypes is explicit so results do not depend on /etc/mime.types.
var contentTypes = map[string]string{
	".html":  "text/html; charset=utf-8",
	".css":   "text/css; charset=utf-8",
	".js":    "text/javascript; charset=utf-8",
	".json":  "application/json",
	".svg":   "image/svg+xml",
	".png":   "image/png",
	".jpg":   "image/jpeg",
	".jpeg":  "image/jpeg",
	".webp":  "image/webp",
	".gif":   "image/gif",
	".avif":  "image/avif",
	".heic":  "image/heic",
	".ico":   "image/x-icon",
	".woff2": "font/woff2",
	".woff":  "font/woff",
	".txt":   "text/plain; charset=utf-8",
	".m4a":   "audio/mp4",
	".aac":   "audio/aac",
	".mp3":   "audio/mpeg",
	".wav":   "audio/wav",
	".ogg":   "audio/ogg",
	".opus":  "audio/ogg",
	".mp4":   "video/mp4",
	".m4v":   "video/mp4",
	".mov":   "video/quicktime",
	".webm":  "video/webm",
	".vtt":   "text/vtt; charset=utf-8",
	".pdf":   "application/pdf",
}

func contentType(p string) string {
	return contentTypes[strings.ToLower(path.Ext(p))]
}
