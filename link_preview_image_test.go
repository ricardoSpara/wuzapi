package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/vincent-petithory/dataurl"
)

func TestApplyLinkPreviewImage(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 1200, 800))
	for x := 0; x < 1200; x++ {
		for y := 0; y < 800; y++ {
			src.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, src); err != nil {
		t.Fatal(err)
	}

	result := openGraphResult{Title: "kept", Description: "kept"}
	if err := applyLinkPreviewImage(context.Background(), dataurl.New(encoded.Bytes(), "image/png").String(), &result); err != nil {
		t.Fatalf("applyLinkPreviewImage: %v", err)
	}

	if result.Title != "kept" || result.Description != "kept" {
		t.Errorf("metadata changed: %q %q", result.Title, result.Description)
	}
	if result.HQWidth != 600 || result.HQHeight != 400 {
		t.Errorf("HQ dimensions = %dx%d; want 600x400", result.HQWidth, result.HQHeight)
	}
	for name, data := range map[string][]byte{"hq": result.HQImageData, "inline": result.ImageData} {
		cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("%s thumbnail is not decodable: %v", name, err)
		}
		if format != "jpeg" || cfg.Width == 0 || cfg.Height == 0 {
			t.Errorf("%s thumbnail = %s %dx%d; want jpeg", name, format, cfg.Width, cfg.Height)
		}
	}
	inline, _, err := image.DecodeConfig(bytes.NewReader(result.ImageData))
	if err != nil {
		t.Fatal(err)
	}
	if inline.Width > openGraphThumbnailWidth || inline.Height > openGraphThumbnailHeight {
		t.Errorf("inline thumbnail %dx%d exceeds bound", inline.Width, inline.Height)
	}

	rawResult := openGraphResult{}
	if err := applyLinkPreviewImage(context.Background(), base64.StdEncoding.EncodeToString(encoded.Bytes()), &rawResult); err != nil {
		t.Fatalf("applyLinkPreviewImage with bare base64: %v", err)
	}
	if rawResult.HQWidth != 600 || rawResult.HQHeight != 400 {
		t.Errorf("bare base64 HQ dimensions = %dx%d; want 600x400", rawResult.HQWidth, rawResult.HQHeight)
	}
}

func TestApplyLinkPreviewImageRejectsInvalidSource(t *testing.T) {
	err := applyLinkPreviewImage(context.Background(), "file:///etc/passwd", &openGraphResult{})
	if err == nil {
		t.Fatal("expected an error for a non-http/data source")
	}
}

func TestPreviewTitleFallback(t *testing.T) {
	for input, want := range map[string]string{
		"https://example.com/path?q=1": "example.com",
		"not a url":                    "not a url",
	} {
		if got := previewTitleFallback(input); got != want {
			t.Errorf("previewTitleFallback(%q) = %q; want %q", input, got, want)
		}
	}
}
