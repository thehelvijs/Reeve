package main

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const maxImageBytes = 1 << 20 // 1 MiB

// imageExts maps an accepted image content type to its stored file extension.
var imageExts = map[string]string{
	"image/png":  ".png",
	"image/jpeg": ".jpg",
	"image/webp": ".webp",
}

// readImageUpload pulls a single uploaded image from a multipart form, enforcing
// the size cap and content-type allowlist. It writes the error response itself
// and returns ok=false on failure.
func readImageUpload(w http.ResponseWriter, r *http.Request, field string) (data []byte, ext string, ok bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxImageBytes+4096)
	file, _, err := formFile(r, field)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "an image file is required")
		return nil, "", false
	}
	defer file.Close()

	data, err = io.ReadAll(io.LimitReader(file, maxImageBytes+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "could not read upload")
		return nil, "", false
	}
	if len(data) > maxImageBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "file_too_large", "image must be at most 1 MB")
		return nil, "", false
	}
	ext, allowed := imageExts[http.DetectContentType(data)]
	if !allowed {
		writeError(w, http.StatusBadRequest, "unsupported_type", "image must be a PNG, JPEG, or WebP image")
		return nil, "", false
	}
	return data, ext, true
}

// writeImageFile stores data at dir/base+ext, creating dir as needed and
// removing a prior file whose extension differed.
func writeImageFile(dir, base, ext string, data []byte, oldPath string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	dest := filepath.Join(dir, base+ext)
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return "", err
	}
	if oldPath != "" && oldPath != dest {
		os.Remove(oldPath)
	}
	return dest, nil
}

// assetURL builds the public URL for an entity's icon or thumbnail, empty when
// no file is stored for it.
func assetURL(kind, id, leaf, path string) string {
	if path == "" {
		return ""
	}
	return "/api/v1/" + kind + "/" + id + "/" + leaf
}
