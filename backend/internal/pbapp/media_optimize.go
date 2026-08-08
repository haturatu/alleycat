package pbapp

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

const (
	mediaWebPQuality     = 82
	maxMediaUploadBytes  = 10 * 1024 * 1024
	maxMediaUploadPixels = 25_000_000
	mediaConvertTimeout  = 30 * time.Second
)

func registerMediaOptimizationHooks(app *pocketbase.PocketBase) {
	app.OnRecordValidate("media").BindFunc(func(e *core.RecordEvent) error {
		if err := convertUnsavedMediaFilesToWebP(e.Record); err != nil {
			return err
		}
		return e.Next()
	})
}

func convertUnsavedMediaFilesToWebP(record *core.Record) error {
	for _, file := range record.GetUnsavedFiles("file") {
		if file == nil {
			continue
		}
		optimized, err := convertUploadedImageToWebP(file)
		if err != nil {
			slog.Warn("media image optimization failed", "name", file.OriginalName, "size", file.Size, "error", err)
			return err
		}
		if optimized == nil {
			continue
		}
		*file = *optimized
	}
	return nil
}

func convertUploadedImageToWebP(file *filesystem.File) (*filesystem.File, error) {
	if file.Size > maxMediaUploadBytes {
		return nil, fmt.Errorf("uploaded media exceeds %d bytes", maxMediaUploadBytes)
	}
	reader, err := file.Reader.Open()
	if err != nil {
		return nil, fmt.Errorf("open uploaded media: %w", err)
	}
	defer func() { _ = reader.Close() }()

	input, err := io.ReadAll(io.LimitReader(reader, maxMediaUploadBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read uploaded media: %w", err)
	}
	if len(input) > maxMediaUploadBytes {
		return nil, fmt.Errorf("uploaded media exceeds %d bytes", maxMediaUploadBytes)
	}
	if !isCWebPSupportedUpload(input, file.OriginalName) {
		return nil, nil
	}
	if err := validateMediaDimensions(input); err != nil {
		return nil, err
	}

	output, err := convertImageBytesWithCWebP(input, file.OriginalName)
	if err != nil {
		return nil, err
	}

	converted, err := filesystem.NewFileFromBytes(output, webpUploadName(file.OriginalName))
	if err != nil {
		return nil, fmt.Errorf("create optimized media file: %w", err)
	}
	return converted, nil
}

func convertImageBytesWithCWebP(input []byte, originalName string) ([]byte, error) {
	if _, err := exec.LookPath("cwebp"); err != nil {
		return nil, fmt.Errorf("cwebp command not found: %w", err)
	}

	dir, err := os.MkdirTemp("", "alleycat-media-*")
	if err != nil {
		return nil, fmt.Errorf("create media temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	inputPath := filepath.Join(dir, "input"+inputExtension(originalName))
	outputPath := filepath.Join(dir, "output.webp")
	if err := os.WriteFile(inputPath, input, 0o600); err != nil {
		return nil, fmt.Errorf("write media temp input: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), mediaConvertTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "cwebp", "-quiet", "-q", fmt.Sprintf("%d", mediaWebPQuality), "-metadata", "none", inputPath, "-o", outputPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("convert media to webp: %w", ctx.Err())
		}
		return nil, fmt.Errorf("convert media to webp: %w: %s", err, strings.TrimSpace(string(out)))
	}

	output, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("read media temp output: %w", err)
	}
	if len(output) == 0 {
		return nil, fmt.Errorf("convert media to webp: empty output")
	}
	return output, nil
}

func validateMediaDimensions(data []byte) error {
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("decode uploaded media dimensions: %w", err)
	}
	if config.Width <= 0 || config.Height <= 0 {
		return fmt.Errorf("uploaded media has invalid dimensions")
	}
	if int64(config.Width)*int64(config.Height) > maxMediaUploadPixels {
		return fmt.Errorf("uploaded media exceeds %d pixels", maxMediaUploadPixels)
	}
	return nil
}

func isCWebPSupportedUpload(data []byte, originalName string) bool {
	contentType := http.DetectContentType(data)
	if contentType == "image/svg+xml" || bytes.Contains(bytes.ToLower(data[:min(len(data), 512)]), []byte("<svg")) {
		return false
	}
	switch contentType {
	case "image/jpeg", "image/png", "image/webp", "image/tiff":
		return true
	}
	switch strings.ToLower(filepath.Ext(originalName)) {
	case ".jpg", ".jpeg", ".png", ".webp", ".tif", ".tiff":
		return true
	default:
		return false
	}
}

func inputExtension(originalName string) string {
	ext := strings.ToLower(filepath.Ext(originalName))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".tif", ".tiff":
		return ext
	default:
		return ".img"
	}
}

func webpUploadName(originalName string) string {
	base := strings.TrimSuffix(filepath.Base(originalName), filepath.Ext(originalName))
	base = strings.TrimSpace(base)
	if base == "" || base == "." {
		base = "image"
	}
	return base + ".webp"
}
