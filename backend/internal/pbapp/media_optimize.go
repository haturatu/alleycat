package pbapp

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

const mediaWebPQuality = 82

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
	reader, err := file.Reader.Open()
	if err != nil {
		return nil, fmt.Errorf("open uploaded media: %w", err)
	}
	defer reader.Close()

	input, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read uploaded media: %w", err)
	}
	if !isCWebPSupportedUpload(input, file.OriginalName) {
		return nil, nil
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
	defer os.RemoveAll(dir)

	inputPath := filepath.Join(dir, "input"+inputExtension(originalName))
	outputPath := filepath.Join(dir, "output.webp")
	if err := os.WriteFile(inputPath, input, 0o600); err != nil {
		return nil, fmt.Errorf("write media temp input: %w", err)
	}

	cmd := exec.Command("cwebp", "-quiet", "-q", fmt.Sprintf("%d", mediaWebPQuality), "-metadata", "none", inputPath, "-o", outputPath)
	if out, err := cmd.CombinedOutput(); err != nil {
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
