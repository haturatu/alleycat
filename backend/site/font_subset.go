package main

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

const (
	fontSubsetScriptPath = "/app/font-subset/subset_fonts.py"
	fontSubsetConfigPath = "/app/font-subset/font-subset.config.json"
	fontSubsetCorpusFile = ".font-subset-content.txt"
)

func maybeSubsetRuntimeFonts(snapshotRoot string) error {
	if snapshotRoot == "" {
		return nil
	}
	if _, err := os.Stat(fontSubsetScriptPath); err != nil {
		return nil
	}
	if _, err := os.Stat(fontSubsetConfigPath); err != nil {
		return nil
	}
	if _, err := os.Stat(publicDir); err != nil {
		return nil
	}

	slog.Info("font subset start", "snapshot_root", snapshotRoot, "public_dir", publicDir)
	cmd := exec.Command("python3", fontSubsetScriptPath, "--root", filepath.Dir(fontSubsetConfigPath), "--config", fontSubsetConfigPath)
	cmd.Env = append(os.Environ(),
		"PUBLIC_DIR="+publicDir,
		"SNAPSHOT_DIR="+snapshotRoot,
	)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("subset fonts: %w: %s", err, output.String())
	}
	slog.Info("font subset completed", "snapshot_root", snapshotRoot)
	return nil
}

type fontSubsetCorpus struct {
	mu    sync.Mutex
	seen  map[rune]struct{}
	runes []rune
}

func newFontSubsetCorpus() *fontSubsetCorpus {
	return &fontSubsetCorpus{seen: make(map[rune]struct{})}
}

func loadFontSubsetCorpus(snapshotRoot string) (*fontSubsetCorpus, error) {
	corpus := newFontSubsetCorpus()
	if snapshotRoot == "" {
		return corpus, nil
	}

	target := filepath.Join(snapshotRoot, fontSubsetCorpusFile)
	data, err := os.ReadFile(target)
	if err != nil {
		if os.IsNotExist(err) {
			return corpus, nil
		}
		return nil, err
	}
	corpus.AddText(string(data))
	return corpus, nil
}

func (c *fontSubsetCorpus) AddHTML(body []byte) {
	if c == nil || len(body) == 0 {
		return
	}
	c.AddText(stripHTML(string(body)))
}

func (c *fontSubsetCorpus) AddText(value string) {
	if c == nil {
		return
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	for _, r := range trimmed {
		if _, ok := c.seen[r]; ok {
			continue
		}
		c.seen[r] = struct{}{}
		c.runes = append(c.runes, r)
	}
}

func (c *fontSubsetCorpus) Write(snapshotRoot string) error {
	if c == nil || snapshotRoot == "" {
		return nil
	}

	target := filepath.Join(snapshotRoot, fontSubsetCorpusFile)
	c.mu.Lock()
	data := []byte(string(c.runes))
	c.mu.Unlock()
	if len(data) == 0 {
		return os.WriteFile(target, nil, 0o644)
	}
	return os.WriteFile(target, data, 0o644)
}
