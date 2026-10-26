package services

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/draw"
	_ "image/gif" // registered decoders for uploads
	"image/jpeg"
	_ "image/png"
	"io"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const (
	// MaxThumbnailBytes is the largest upload accepted.
	MaxThumbnailBytes = 5 << 20
	// Refuse images that would take too much memory to decode (decompression bombs).
	maxThumbnailSourcePixels = 40_000_000
	thumbnailWidth           = 1280
	thumbnailHeight          = 720
)

var errThumbnailFormat = validationError("thumbnail", "Upload a JPEG, PNG, WebP or GIF image")

// processThumbnail turns any accepted upload into a 16:9 JPEG of at most
// 1280x720: cropped to the center, scaled down (never up) and re-encoded.
// Re-encoding drops metadata (EXIF, GPS) and anything hidden in the file.
// Returns the JPEG and a short version string for cache-busting URLs.
func processThumbnail(r io.Reader) ([]byte, string, error) {
	data, err := io.ReadAll(io.LimitReader(r, MaxThumbnailBytes+1))
	if err != nil {
		return nil, "", validationError("thumbnail", "The upload didn't finish, please try again")
	}
	if len(data) > MaxThumbnailBytes {
		return nil, "", validationError("thumbnail", "The image must be 5 MB or smaller")
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, "", errThumbnailFormat
	}
	if cfg.Width < 160 || cfg.Height < 90 {
		return nil, "", validationError("thumbnail", "The image must be at least 160x90 pixels")
	}
	if cfg.Width*cfg.Height > maxThumbnailSourcePixels {
		return nil, "", validationError("thumbnail", "The image is too large (max 40 megapixels)")
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", errThumbnailFormat
	}

	// Largest centered 16:9 area of the source.
	b := src.Bounds()
	cropW, cropH := b.Dx(), b.Dx()*9/16
	if cropH > b.Dy() {
		cropW, cropH = b.Dy()*16/9, b.Dy()
	}
	x0 := b.Min.X + (b.Dx()-cropW)/2
	y0 := b.Min.Y + (b.Dy()-cropH)/2
	crop := image.Rect(x0, y0, x0+cropW, y0+cropH)

	outW, outH := thumbnailWidth, thumbnailHeight
	if cropW < outW {
		outW, outH = cropW, cropW*9/16
	}
	dst := image.NewRGBA(image.Rect(0, 0, outW, outH))
	draw.Draw(dst, dst.Bounds(), image.Black, image.Point{}, draw.Src) // background for transparent images
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, crop, xdraw.Over, nil)

	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: 85}); err != nil {
		return nil, "", err
	}
	sum := sha256.Sum256(out.Bytes())
	return out.Bytes(), hex.EncodeToString(sum[:])[:12], nil
}
