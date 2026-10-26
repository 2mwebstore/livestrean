package services

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func pngImage(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		img.Set(x, h/2, color.NRGBA{R: 255, A: 255})
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestProcessThumbnailSizes(t *testing.T) {
	cases := []struct {
		name         string
		w, h         int
		wantW, wantH int
	}{
		{"1080p is scaled down", 1920, 1080, 1280, 720},
		{"portrait is center-cropped to 16:9", 1080, 1920, 1080, 607},
		{"small 4:3 is cropped, not enlarged", 400, 300, 400, 225},
	}
	for _, tc := range cases {
		out, version, err := processThumbnail(bytes.NewReader(pngImage(t, tc.w, tc.h)))
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		cfg, format, err := image.DecodeConfig(bytes.NewReader(out))
		if err != nil || format != "jpeg" {
			t.Fatalf("%s: output is not a JPEG: %v %s", tc.name, err, format)
		}
		if cfg.Width != tc.wantW || cfg.Height != tc.wantH {
			t.Errorf("%s: got %dx%d, want %dx%d", tc.name, cfg.Width, cfg.Height, tc.wantW, tc.wantH)
		}
		if len(version) != 12 {
			t.Errorf("%s: version %q", tc.name, version)
		}
	}
}

func TestProcessThumbnailRejects(t *testing.T) {
	big := make([]byte, MaxThumbnailBytes+10)
	cases := map[string][]byte{
		"not an image":  []byte("<script>alert(1)</script>"),
		"too small":     pngImage(t, 100, 50),
		"too many MB":   big,
		"truncated PNG": pngImage(t, 400, 300)[:60],
	}
	for name, data := range cases {
		if _, _, err := processThumbnail(bytes.NewReader(data)); err == nil {
			t.Errorf("%s: expected an error", name)
		} else if svcErr, ok := err.(*Error); !ok || svcErr.Kind != KindValidation {
			t.Errorf("%s: expected a validation error, got %v", name, err)
		}
	}
}

func TestProcessThumbnailStripsMetadata(t *testing.T) {
	// A JPEG with an APP1 (EXIF) segment carrying "GPS" text.
	var src bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 320, 180))
	if err := jpeg.Encode(&src, img, nil); err != nil {
		t.Fatal(err)
	}
	exif := append([]byte{0xFF, 0xE1, 0x00, 0x10}, []byte("Exif\x00\x00GPS-SECRET")...)
	withExif := append(append([]byte{0xFF, 0xD8}, exif...), src.Bytes()[2:]...)

	out, _, err := processThumbnail(bytes.NewReader(withExif))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(out, []byte("GPS-SECRET")) {
		t.Fatal("metadata survived re-encoding")
	}
}
