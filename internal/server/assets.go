package server

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/jakobneri/sandyadvent/internal/doors"
)

// serveAsset serves one file of a door from assets_dir. The caller has already
// done the token and date-lock checks. Only files listed in the door's
// "assets" are served; os.OpenInRoot additionally prevents escaping the
// directory (also via symlinks). Range requests are handled by
// http.ServeContent, which iOS Safari needs for audio and video.
func (s *Server) serveAsset(w http.ResponseWriter, r *http.Request, door *doors.Door, rel string) {
	clean, err := doors.CleanAssetPath(rel)
	if err != nil || !door.AllowsAsset(clean) {
		s.notFound(w, r)
		return
	}
	f, err := os.OpenInRoot(s.cfg.AssetsDir, filepath.FromSlash(clean))
	if err != nil {
		s.notFound(w, r)
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() {
		s.notFound(w, r)
		return
	}
	h := w.Header()
	if ct := contentType(clean); ct != "" {
		h.Set("Content-Type", ct)
	} else {
		h.Set("Content-Type", "application/octet-stream")
	}
	// Doors never close again once open, so a private cache is safe.
	h.Set("Cache-Control", "private, max-age=86400")
	http.ServeContent(w, r, clean, st.ModTime(), f)
}
