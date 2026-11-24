package web

import (
	"bytes"
	"io/fs"
	"os"
	"testing"
)

// The embedded brand files are copies of ../shared/schatzogram-brand
// (source of truth: repo schatzogram). Run `make sync-brand` if this fails.
func TestBrandCopiesInSync(t *testing.T) {
	pairs := map[string]string{
		"static/brand/tokens.css": "../shared/schatzogram-brand/tokens.css",
		"brand/logo-mark.svg":     "../shared/schatzogram-brand/logo-mark.svg",
	}
	for embedded, shared := range pairs {
		a, err := fs.ReadFile(FS, embedded)
		if err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(shared)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(a, b) {
			t.Errorf("%s weicht von %s ab – 'make sync-brand' ausführen", embedded, shared)
		}
	}
}

func TestFontsAndLicensesEmbedded(t *testing.T) {
	for _, p := range []string{"static/fonts/LICENSE-Fraunces.txt", "static/fonts/LICENSE-Nunito.txt",
		"static/fonts/fraunces-latin-600-normal.woff2", "static/fonts/nunito-latin-400-normal.woff2"} {
		if _, err := fs.Stat(FS, p); err != nil {
			t.Error(err)
		}
	}
}
