// Package photostore saves item photos to local disk — this app runs on one
// small VPS with very few photos, so a plain directory is enough; there is no
// S3/object storage in this stack (see CLAUDE.md). Every image is decoded,
// downsized to a fixed longest edge and re-encoded as a JPEG at a fixed
// quality on the way in, which is what actually keeps files small — the
// caller never sees the original upload again, only the compressed copy.
package photostore

import (
	"bytes"
	"image"
	_ "image/gif" // decode support only
	"image/jpeg"
	_ "image/png" // decode support only
	"os"
	"path/filepath"

	"budget_me/backend/internal/ids"
)

const (
	maxSide = 1024
	quality = 70
)

// Store roots every saved/removed path under dir.
type Store struct {
	dir string
}

func New(dir string) Store {
	return Store{dir: dir}
}

// Save decodes an uploaded image, downsizes it to maxSide on its longest edge
// (never upscales) and writes it as a JPEG under <dir>/<subdir>/<uuid>.jpg.
// It returns the path relative to dir, which is what callers persist in the
// DB and later hand back to Remove.
func (s Store) Save(subdir string, data []byte) (string, error) {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	full := filepath.Join(s.dir, subdir)
	if err := os.MkdirAll(full, 0o755); err != nil {
		return "", err
	}

	rel := filepath.Join(subdir, ids.New().String()+".jpg")

	f, err := os.Create(filepath.Join(s.dir, rel))
	if err != nil {
		return "", err
	}

	defer func() { _ = f.Close() }()

	if err := jpeg.Encode(f, downsize(src, maxSide), &jpeg.Options{Quality: quality}); err != nil {
		return "", err
	}

	return rel, nil
}

// Remove deletes a previously-saved file. A missing file is not an error —
// callers use this best-effort, both to replace a photo and to clean up
// after the owning item is deleted.
func (s Store) Remove(rel string) error {
	if rel == "" {
		return nil
	}

	err := os.Remove(filepath.Join(s.dir, rel))
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// downsize nearest-neighbour samples src down to fit within max×max on its
// longest edge. Plain nearest-neighbour (no library) is enough here: the
// JPEG re-encode that follows dominates the final quality/size far more than
// the resampling method does, and it keeps this package dependency-free.
func downsize(src image.Image, max int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()

	if w <= max && h <= max {
		return src
	}

	var nw, nh int
	if w >= h {
		nw, nh = max, h*max/w
	} else {
		nh, nw = max, w*max/h
	}

	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	for y := range nh {
		for x := range nw {
			dst.Set(x, y, src.At(b.Min.X+x*w/nw, b.Min.Y+y*h/nh))
		}
	}

	return dst
}
