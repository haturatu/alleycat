package pbapp

import (
	"encoding/binary"
	"hash/crc32"
	"testing"
)

func TestWebPUploadName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want string
	}{
		{name: "photo.jpg", want: "photo.webp"},
		{name: "archive.photo.png", want: "archive.photo.webp"},
		{name: "", want: "image.webp"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := webpUploadName(tt.name); got != tt.want {
				t.Fatalf("webpUploadName(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestIsCWebPSupportedUpload(t *testing.T) {
	t.Parallel()

	png := []byte{0x89, 0x50, 0x4e, 0x47, '\r', '\n', 0x1a, '\n', 0, 0, 0, 0}
	if !isCWebPSupportedUpload(png, "image.png") {
		t.Fatalf("png should be optimized")
	}

	if isCWebPSupportedUpload([]byte("<svg xmlns=\"http://www.w3.org/2000/svg\"></svg>"), "image.svg") {
		t.Fatalf("svg should not be optimized")
	}

	if !isCWebPSupportedUpload([]byte("unknown bytes"), "photo.jpg") {
		t.Fatalf("known cwebp-supported extensions should be optimized when sniffing is inconclusive")
	}

	if isCWebPSupportedUpload([]byte("GIF89a"), "image.gif") {
		t.Fatalf("gif should not be optimized because animated images would lose frames")
	}

	if isCWebPSupportedUpload([]byte("plain text"), "note.txt") {
		t.Fatalf("plain text should not be optimized")
	}
}

func TestValidateMediaDimensions(t *testing.T) {
	valid := minimalPNG(2000, 2000)
	if err := validateMediaDimensions(valid); err != nil {
		t.Fatalf("valid dimensions rejected: %v", err)
	}

	tooLarge := minimalPNG(5001, 5000)
	if err := validateMediaDimensions(tooLarge); err == nil {
		t.Fatal("oversized dimensions should be rejected")
	}
}

func minimalPNG(width, height uint32) []byte {
	data := make([]byte, 33)
	copy(data, []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a})
	binary.BigEndian.PutUint32(data[8:12], 13)
	copy(data[12:16], []byte("IHDR"))
	binary.BigEndian.PutUint32(data[16:20], width)
	binary.BigEndian.PutUint32(data[20:24], height)
	data[24] = 8
	data[25] = 2
	binary.BigEndian.PutUint32(data[29:33], crc32.ChecksumIEEE(data[12:29]))
	return data
}
