// Command gen writes the synthetic WhatsApp example exports
// (tools/waparse/testdata/android.txt and ios.txt). Both contain the same
// invented conversation; the tests check that they produce identical stats.
//
//	go run ./tools/waparse/gen [-dir tools/waparse/testdata]
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	_ "time/tzdata"

	"github.com/jakobneri/sandyadvent/internal/waparse"
)

func main() {
	dir := flag.String("dir", "tools/waparse/testdata", "Zielordner")
	flag.Parse()
	msgs := waparse.Synthetic()
	if err := os.MkdirAll(*dir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "Fehler:", err)
		os.Exit(1)
	}
	for name, data := range map[string][]byte{"android.txt": waparse.RenderAndroid(msgs), "ios.txt": waparse.RenderIOS(msgs)} {
		p := filepath.Join(*dir, name)
		if err := os.WriteFile(p, data, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "Fehler:", err)
			os.Exit(1)
		}
		fmt.Printf("%s: %d Nachrichten, %d Bytes\n", p, len(msgs), len(data))
	}
}
