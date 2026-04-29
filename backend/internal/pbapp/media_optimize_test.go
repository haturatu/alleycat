package pbapp

import "testing"

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
