// Package qrpdf renders the 17 door QR codes onto A4 pages for printing.
package qrpdf

import (
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/url"
	"sort"
	"strings"

	"github.com/go-pdf/fpdf"
	qrcode "github.com/skip2/go-qrcode"

	"github.com/jakobneri/sandyadvent/internal/advent"
	"github.com/jakobneri/sandyadvent/internal/config"
	"github.com/jakobneri/sandyadvent/internal/doors"
)

// MinSizeMM is the minimum printed code size (CLAUDE.md: mind. 3 cm).
const MinSizeMM = 30

// Page geometry (A4, millimetres).
const (
	pageW, pageH = 210.0, 297.0
	margin       = 10.0
	headerH      = 10.0
	labelH       = 9.0
)

// Code is one QR code to print.
type Code struct {
	Day int
	URL string
}

// Codes builds the door URLs. It requires exactly one valid token for each of
// the 17 QR days (and nothing else).
func Codes(tokens map[string]int, baseURL string) ([]Code, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, errors.New("Base-URL fehlt (-base-url oder base_url in der Config)")
	}
	if !doors.IsHTTPURL(baseURL) {
		return nil, fmt.Errorf("Base-URL %q muss mit http:// oder https:// beginnen", baseURL)
	}
	if errs := config.ValidateTokens(tokens); len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	byDay := map[int]string{}
	for tok, day := range tokens {
		byDay[day] = tok
	}
	var missing []int
	for _, d := range advent.QRDays {
		if byDay[d] == "" {
			missing = append(missing, d)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("Tokens fehlen für Tag %v – 'sandyadvent gen-tokens -write' ausführen", missing)
	}
	if len(tokens) != len(advent.QRDays) {
		return nil, fmt.Errorf("erwartet genau %d Tokens, gefunden %d", len(advent.QRDays), len(tokens))
	}
	codes := make([]Code, 0, len(advent.QRDays))
	for _, d := range advent.QRDays {
		codes = append(codes, Code{Day: d, URL: baseURL + "/d/" + byDay[d]})
	}
	sort.Slice(codes, func(i, j int) bool { return codes[i].Day < codes[j].Day })
	return codes, nil
}

// CheckPrintable reports why codes for these tokens and this base URL would
// not work on Sandy's phone: a loopback/unspecified host (e.g. the
// http://localhost:8081 from config.example.json) or the public EXAMPLE-
// placeholder tokens. Callers allow it only for explicit test prints.
func CheckPrintable(tokens map[string]int, baseURL string) error {
	var errs []error
	if u, err := url.Parse(strings.TrimSpace(baseURL)); err == nil && IsLocalHost(u.Hostname()) {
		errs = append(errs, fmt.Errorf("Base-URL %q zeigt auf diesen Rechner (%s) – das Handy erreicht sie nicht; -base-url http://<Pi-IP> angeben", baseURL, u.Hostname()))
	}
	for tok := range tokens {
		if strings.HasPrefix(tok, config.ExampleTokenPrefix) {
			errs = append(errs, errors.New("Beispiel-Tokens (EXAMPLE-…) aus config.example.json – zuerst 'make tokens' bzw. 'sandyadvent gen-tokens -write' ausführen"))
			break
		}
	}
	return errors.Join(errs...)
}

// IsLocalHost reports whether host is localhost, a *.localhost name or a
// loopback/unspecified IP address.
func IsLocalHost(host string) bool {
	h := strings.ToLower(strings.TrimSuffix(host, "."))
	if h == "localhost" || strings.HasSuffix(h, ".localhost") {
		return true
	}
	ip := net.ParseIP(h)
	return ip != nil && (ip.IsLoopback() || ip.IsUnspecified())
}

// Bitmap returns the QR modules (error correction H) including the standard
// 4-module quiet zone.
func Bitmap(content string) ([][]bool, error) {
	q, err := qrcode.New(content, qrcode.Highest)
	if err != nil {
		return nil, err
	}
	return q.Bitmap(), nil
}

// QuietModules is the quiet zone width that go-qrcode adds on each side.
const QuietModules = 4

// Options control the layout.
type Options struct {
	SizeMM   float64 // edge length of the code itself (dark symbol, without quiet zone)
	CutMarks bool
	Title    string // small header line (e.g. base URL)
}

// Cell is the position of one code incl. quiet zone on a page (mm, from top left).
type Cell struct {
	Page       int // 0-based
	X, Y, Size float64
}

// Layout computes the cell positions for n codes whose outer size (symbol +
// quiet zone) is sizeMM.
func Layout(n int, sizeMM float64) ([]Cell, error) {
	gap := 6.0 // room for cut marks outside the quiet zones
	cols := int((pageW - 2*margin + gap) / (sizeMM + gap))
	rows := int((pageH - 2*margin - headerH + gap) / (sizeMM + labelH + gap))
	if cols < 1 || rows < 1 {
		return nil, fmt.Errorf("QR-Code mit %g mm passt nicht auf A4", sizeMM)
	}
	perPage := cols * rows
	gridW := float64(cols)*sizeMM + float64(cols-1)*gap
	x0 := (pageW - gridW) / 2
	cells := make([]Cell, n)
	for i := range cells {
		p, k := i/perPage, i%perPage
		r, c := k/cols, k%cols
		cells[i] = Cell{
			Page: p,
			X:    x0 + float64(c)*(sizeMM+gap),
			Y:    margin + headerH + float64(r)*(sizeMM+labelH+gap),
			Size: sizeMM,
		}
	}
	return cells, nil
}

// WritePDF renders the codes to w.
func WritePDF(w io.Writer, codes []Code, o Options) error {
	if o.SizeMM == 0 {
		o.SizeMM = 40
	}
	if o.SizeMM < MinSizeMM {
		return fmt.Errorf("-size-mm %g zu klein (mindestens %d mm)", o.SizeMM, MinSizeMM)
	}
	if len(codes) == 0 {
		return errors.New("keine Codes")
	}
	bitmaps := make([][][]bool, len(codes))
	modules := 0
	for i, code := range codes {
		bm, err := Bitmap(code.URL)
		if err != nil {
			return fmt.Errorf("Tag %d: %w", code.Day, err)
		}
		bitmaps[i] = bm
		modules = max(modules, len(bm))
	}
	// Outer size so that the symbol itself is SizeMM (for the largest version).
	outer := o.SizeMM * float64(modules) / float64(modules-2*QuietModules)
	cells, err := Layout(len(codes), outer)
	if err != nil {
		return err
	}
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetCreator("sandyadvent qrpdf", false)
	pdf.SetAutoPageBreak(false, 0)
	pages := 0
	for i, code := range codes {
		cell := cells[i]
		if cell.Page >= pages {
			pdf.AddPage()
			pages++
			pdf.SetFont("Helvetica", "", 8)
			pdf.SetTextColor(120, 120, 120)
			pdf.SetXY(margin, margin)
			title := o.Title
			if pages > 1 {
				title += fmt.Sprintf("  (Seite %d)", pages)
			}
			pdf.CellFormat(pageW-2*margin, 5, title, "", 0, "L", false, 0, "")
		}
		bm := bitmaps[i]
		// keep the module size identical for all codes
		drawQR(pdf, bm, cell.X, cell.Y, cell.Size*float64(len(bm))/float64(modules))
		pdf.SetFont("Helvetica", "B", 22)
		pdf.SetTextColor(0, 0, 0)
		pdf.SetXY(cell.X, cell.Y+cell.Size)
		pdf.CellFormat(cell.Size, labelH, fmt.Sprintf("%d", code.Day), "", 0, "C", false, 0, "")
		if o.CutMarks {
			drawCutMarks(pdf, cell.X, cell.Y, cell.Size, cell.Size+labelH)
		}
	}
	if err := pdf.Error(); err != nil {
		return err
	}
	return pdf.Output(w)
}

// drawQR draws the bitmap (which already contains the quiet zone) as an
// sizeMM square; dark modules are merged into horizontal runs.
func drawQR(pdf *fpdf.Fpdf, bm [][]bool, x, y, size float64) {
	n := len(bm)
	m := size / float64(n)
	pdf.SetFillColor(255, 255, 255)
	pdf.Rect(x, y, size, size, "F")
	pdf.SetFillColor(0, 0, 0)
	for r := 0; r < n; r++ {
		for c := 0; c < n; {
			if !bm[r][c] {
				c++
				continue
			}
			start := c
			for c < n && bm[r][c] {
				c++
			}
			// tiny overlap avoids hairline gaps in some PDF viewers
			pdf.Rect(x+float64(start)*m, y+float64(r)*m, float64(c-start)*m+0.01, m+0.01, "F")
		}
	}
}

func drawCutMarks(pdf *fpdf.Fpdf, x, y, w, h float64) {
	const l = 3.0
	pdf.SetDrawColor(170, 170, 170)
	pdf.SetLineWidth(0.15)
	for _, p := range [][2]float64{{x, y}, {x + w, y}, {x, y + h}, {x + w, y + h}} {
		dx := math.Copysign(l, p[0]-(x+w/2))
		dy := math.Copysign(l, p[1]-(y+h/2))
		pdf.Line(p[0], p[1], p[0]+dx, p[1])
		pdf.Line(p[0], p[1], p[0], p[1]+dy)
	}
}
