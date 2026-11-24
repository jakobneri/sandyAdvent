// Package tokens generates door tokens and merges them into config.json
// without touching anything else in the file.
package tokens

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/jakobneri/sandyadvent/internal/advent"
	"github.com/jakobneri/sandyadvent/internal/config"
)

const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"

// New returns a random URL-safe token of config.TokenLength characters.
// 64 symbols => 6 bits each, so masking a random byte is unbiased.
func New(r io.Reader) (string, error) {
	if r == nil {
		r = rand.Reader
	}
	buf := make([]byte, config.TokenLength)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", err
	}
	for i, b := range buf {
		buf[i] = alphabet[b&63]
	}
	return string(buf), nil
}

// Generate returns a fresh token map (token -> day) for all QR days.
func Generate(r io.Reader) (map[string]int, error) {
	return Fill(nil, r, false)
}

// Fill returns a copy of existing with tokens added for every QR day that has
// none. Placeholder tokens (EXAMPLE-…) are replaced; force replaces all.
// Tokens for non-QR days are dropped only when force is set.
func Fill(existing map[string]int, r io.Reader, force bool) (map[string]int, error) {
	out := map[string]int{}
	have := map[int]bool{}
	if !force {
		for tok, day := range existing {
			if strings.HasPrefix(tok, config.ExampleTokenPrefix) {
				continue
			}
			out[tok] = day
			have[day] = true
		}
	}
	for _, day := range advent.QRDays {
		if have[day] {
			continue
		}
		for {
			tok, err := New(r)
			if err != nil {
				return nil, err
			}
			if _, dup := out[tok]; dup {
				continue
			}
			out[tok] = day
			break
		}
	}
	return out, nil
}

// MarshalMap renders a token map as indented JSON sorted by day.
func MarshalMap(m map[string]int, indent string) []byte {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if m[keys[i]] != m[keys[j]] {
			return m[keys[i]] < m[keys[j]]
		}
		return keys[i] < keys[j]
	})
	var b bytes.Buffer
	b.WriteString("{")
	for i, k := range keys {
		if i > 0 {
			b.WriteString(",")
		}
		kj, _ := json.Marshal(k)
		fmt.Fprintf(&b, "\n%s  %s: %s", indent, kj, strconv.Itoa(m[k]))
	}
	if len(keys) > 0 {
		b.WriteString("\n" + indent)
	}
	b.WriteString("}")
	return b.Bytes()
}

// WriteConfig merges tokens into the config file at path, keeping all other
// top-level keys in their original order. Returns the days that got new tokens.
func WriteConfig(path string, r io.Reader, force bool) ([]int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg, err := config.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if errs := config.ValidateTokens(cfg.Tokens); len(errs) > 0 && !force {
		return nil, fmt.Errorf("vorhandene Tokens ungültig (mit -force neu erzeugen): %w", errors.Join(errs...))
	}
	filled, err := Fill(cfg.Tokens, r, force)
	if err != nil {
		return nil, err
	}
	var added []int
	for tok, day := range filled {
		if _, ok := cfg.Tokens[tok]; !ok {
			added = append(added, day)
		}
	}
	sort.Ints(added)

	out, err := replaceTopLevelKey(raw, "tokens", MarshalMap(filled, "  "))
	if err != nil {
		return nil, err
	}
	mode := os.FileMode(0o600)
	if st, err := os.Stat(path); err == nil {
		mode = st.Mode().Perm()
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".config-*.json")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(out); err != nil {
		tmp.Close()
		return nil, err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return nil, err
	}
	if err := tmp.Close(); err != nil {
		return nil, err
	}
	return added, os.Rename(tmp.Name(), path)
}

// replaceTopLevelKey rewrites a JSON object, replacing (or appending) key.
// Other keys keep their order and raw formatting is re-indented.
func replaceTopLevelKey(raw []byte, key string, value []byte) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	if t, err := dec.Token(); err != nil || t != json.Delim('{') {
		return nil, errors.New("config: Objekt erwartet")
	}
	var b bytes.Buffer
	b.WriteString("{")
	found := false
	n := 0
	write := func(k string, v []byte) error {
		if n > 0 {
			b.WriteString(",")
		}
		n++
		kj, _ := json.Marshal(k)
		b.WriteString("\n  ")
		b.Write(kj)
		b.WriteString(": ")
		if k == key {
			b.Write(v)
			return nil
		}
		var ind bytes.Buffer
		if err := json.Indent(&ind, v, "  ", "  "); err != nil {
			return err
		}
		b.Write(ind.Bytes())
		return nil
	}
	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			return nil, err
		}
		k, _ := t.(string)
		var v json.RawMessage
		if err := dec.Decode(&v); err != nil {
			return nil, err
		}
		if k == key {
			found = true
			v = value
		}
		if err := write(k, v); err != nil {
			return nil, err
		}
	}
	if !found {
		if err := write(key, value); err != nil {
			return nil, err
		}
	}
	b.WriteString("\n}\n")
	return b.Bytes(), nil
}
