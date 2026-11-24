package qrpdf

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	mrand "math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/makiuchi-d/gozxing"
	gzqr "github.com/makiuchi-d/gozxing/qrcode"

	"github.com/jakobneri/sandyadvent/internal/advent"
	"github.com/jakobneri/sandyadvent/internal/tokens"
)

// testTokens returns a deterministic token map so every run renders the same
// codes (set QRPDF_RANDOM_TOKENS=1 to stress-test with fresh random tokens).
func testTokens(t *testing.T) map[string]int {
	t.Helper()
	var r io.Reader = mrand.NewChaCha8([32]byte{'s', 'a', 'n', 'd', 'y'})
	if os.Getenv("QRPDF_RANDOM_TOKENS") != "" {
		r = nil
	}
	m, err := tokens.Generate(r)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func decode(t *testing.T, img image.Image) string {
	t.Helper()
	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		t.Fatal(err)
	}
	// The crops are clean, axis-aligned renders, so PURE_BARCODE (module grid
	// from the bounding box) is the right mode; gozxing's finder-pattern
	// detector alone misses a few percent of valid codes.
	res, err := gzqr.NewQRCodeReader().Decode(bmp, map[gozxing.DecodeHintType]interface{}{gozxing.DecodeHintType_PURE_BARCODE: true})
	if err != nil {
		res, err = gzqr.NewQRCodeReader().Decode(bmp, map[gozxing.DecodeHintType]interface{}{gozxing.DecodeHintType_TRY_HARDER: true})
	}
	if err != nil {
		t.Fatalf("QR nicht lesbar: %v", err)
	}
	return res.GetText()
}

func TestCodesValidation(t *testing.T) {
	m := testTokens(t)
	codes, err := Codes(m, "http://pi.local:8080/")
	if err != nil {
		t.Fatal(err)
	}
	if len(codes) != 17 {
		t.Fatal(len(codes))
	}
	for i, c := range codes {
		if c.Day != advent.QRDays[i] || !strings.HasPrefix(c.URL, "http://pi.local:8080/d/") || strings.Contains(c.URL, "//d/") {
			t.Fatalf("%+v", c)
		}
	}
	if _, err := Codes(m, ""); err == nil {
		t.Error("ohne Base-URL akzeptiert")
	}
	if _, err := Codes(m, "pi.local"); err == nil {
		t.Error("Base-URL ohne Schema akzeptiert")
	}
	missing := map[string]int{}
	for k, v := range m {
		if v != 24 {
			missing[k] = v
		}
	}
	if _, err := Codes(missing, "http://x"); err == nil || !strings.Contains(err.Error(), "24") {
		t.Errorf("fehlender Tag: %v", err)
	}
	bad := map[string]int{"abcdefghij": 2}
	for k, v := range m {
		bad[k] = v
	}
	if _, err := Codes(bad, "http://x"); err == nil {
		t.Error("Gutschein-Tag akzeptiert")
	}
}

func TestBitmapRoundTrip(t *testing.T) {
	url := "http://192.0.2.10/d/AbCdEf-_09"
	bm, err := Bitmap(url)
	if err != nil {
		t.Fatal(err)
	}
	const scale = 8
	img := image.NewGray(image.Rect(0, 0, len(bm)*scale, len(bm)*scale))
	for y := range img.Bounds().Dy() {
		for x := range img.Bounds().Dx() {
			c := color.Gray{255}
			if bm[y/scale][x/scale] {
				c = color.Gray{0}
			}
			img.SetGray(x, y, c)
		}
	}
	if got := decode(t, img); got != url {
		t.Fatalf("got %q", got)
	}
}

func TestLayoutAndMinSize(t *testing.T) {
	if err := WritePDF(&bytes.Buffer{}, []Code{{1, "http://x/d/abcdefghij"}}, Options{SizeMM: 29}); err == nil {
		t.Error("29 mm akzeptiert")
	}
	cells, err := Layout(17, 48)
	if err != nil {
		t.Fatal(err)
	}
	for i, c := range cells {
		if c.X < margin-0.01 || c.X+c.Size > pageW-margin+0.01 || c.Y+c.Size+labelH > pageH-margin+0.01 {
			t.Errorf("Zelle %d außerhalb: %+v", i, c)
		}
		for j := 0; j < i; j++ {
			o := cells[j]
			if o.Page == c.Page && c.X < o.X+o.Size && o.X < c.X+c.Size && c.Y < o.Y+o.Size+labelH && o.Y < c.Y+c.Size+labelH {
				t.Errorf("Zellen %d und %d überlappen", i, j)
			}
		}
	}
}

// Renders the PDF with pdftoppm (poppler) and decodes every code from the PNG.
func TestPDFDecodes(t *testing.T) {
	codes, err := Codes(testTokens(t), "http://192.0.2.10")
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := WritePDF(&buf, codes, Options{SizeMM: 40, CutMarks: true, Title: "test"}); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF-")) {
		t.Fatal("kein PDF")
	}
	if _, err := exec.LookPath("pdftoppm"); err != nil {
		t.Skip("pdftoppm (poppler-utils) nicht installiert – PNG-Prüfung übersprungen")
	}
	dir := t.TempDir()
	pdfPath := filepath.Join(dir, "qr.pdf")
	if err := os.WriteFile(pdfPath, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	const dpi = 300 // 150 dpi was too coarse for gozxing on some codes (flaky)
	if out, err := exec.Command("pdftoppm", "-r", fmt.Sprint(dpi), "-png", pdfPath, filepath.Join(dir, "p")).CombinedOutput(); err != nil {
		t.Fatalf("pdftoppm: %v %s", err, out)
	}
	bm, _ := Bitmap(codes[0].URL)
	outer := 40 * float64(len(bm)) / float64(len(bm)-2*QuietModules)
	cells, _ := Layout(len(codes), outer)
	pages := map[int]image.Image{}
	matches, _ := filepath.Glob(filepath.Join(dir, "p-*.png"))
	if len(matches) != cells[len(cells)-1].Page+1 {
		t.Fatalf("%d Seiten gerendert", len(matches))
	}
	for i, m := range matches {
		f, _ := os.Open(m)
		img, err := png.Decode(f)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		pages[i] = img
	}
	px := func(mm float64) int { return int(mm / 25.4 * dpi) }
	// Exact check: every module centre in the rendered page matches the bitmap.
	for i, c := range cells {
		bm, _ := Bitmap(codes[i].URL)
		m := c.Size / float64(len(bm))
		for r := range bm {
			for col := range bm[r] {
				x := px(c.X + (float64(col)+0.5)*m)
				y := px(c.Y + (float64(r)+0.5)*m)
				dark := color.GrayModel.Convert(pages[c.Page].At(x, y)).(color.Gray).Y < 128
				if dark != bm[r][col] {
					t.Fatalf("Tag %d: Modul (%d,%d) falsch gerendert", codes[i].Day, r, col)
				}
			}
		}
	}
	// Crop each code (outer size incl. quiet zone) and paste it onto a white
	// canvas with extra margin: a crop exactly at the cell edge can shave
	// pixels off the quiet zone because px() truncates.
	const pad = 20
	for i, c := range cells {
		src := image.Rect(px(c.X), px(c.Y), px(c.X+c.Size)+1, px(c.Y+c.Size)+1)
		canvas := image.NewGray(image.Rect(0, 0, src.Dx()+2*pad, src.Dy()+2*pad))
		draw.Draw(canvas, canvas.Bounds(), image.White, image.Point{}, draw.Src)
		draw.Draw(canvas, image.Rect(pad, pad, pad+src.Dx(), pad+src.Dy()), pages[c.Page], src.Min, draw.Src)
		if got := decode(t, canvas); got != codes[i].URL {
			t.Errorf("Tag %d: %q, erwartet %q", codes[i].Day, got, codes[i].URL)
		}
	}
}

func TestCheckPrintable(t *testing.T) {
	good := testTokens(t)
	if err := CheckPrintable(good, "http://192.0.2.10"); err != nil {
		t.Errorf("echte Tokens + LAN-IP abgelehnt: %v", err)
	}
	if err := CheckPrintable(good, "http://pi.fritz.box:8080/"); err != nil {
		t.Errorf("Hostname abgelehnt: %v", err)
	}
	for _, u := range []string{"http://localhost:8081", "http://LOCALHOST", "http://127.0.0.1", "http://127.1.2.3:80/", "http://[::1]:8081", "http://0.0.0.0", "http://foo.localhost"} {
		if err := CheckPrintable(good, u); err == nil {
			t.Errorf("%s akzeptiert", u)
		}
	}
	ex := map[string]int{"EXAMPLE-01": 1}
	if err := CheckPrintable(ex, "http://192.0.2.10"); err == nil || !strings.Contains(err.Error(), "Beispiel-Tokens") {
		t.Errorf("Beispiel-Tokens: %v", err)
	}
}

func TestSizeErrorShowsExactValue(t *testing.T) {
	err := WritePDF(&bytes.Buffer{}, []Code{{1, "http://x/d/abcdefghij"}}, Options{SizeMM: 29.5})
	if err == nil || !strings.Contains(err.Error(), "29.5") {
		t.Errorf("%v", err)
	}
}
