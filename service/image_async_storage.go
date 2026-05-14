package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var imageAsyncTaskIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func GetImageAsyncStorageDir() string {
	if dir := os.Getenv("IMAGE_ASYNC_STORAGE_DIR"); dir != "" {
		return dir
	}
	return "/data/image-async"
}

func GetImageAsyncMaxBodyBytes() int64 {
	mb := int64(64)
	if raw := os.Getenv("IMAGE_ASYNC_MAX_BODY_MB"); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil && parsed > 0 {
			mb = parsed
		}
	}
	return mb << 20
}

func SaveImageAsyncRequestBody(taskID string, body []byte) (path string, sha256Hex string, size int64, err error) {
	if !imageAsyncTaskIDPattern.MatchString(taskID) {
		return "", "", 0, fmt.Errorf("invalid task id")
	}
	if int64(len(body)) > GetImageAsyncMaxBodyBytes() {
		return "", "", 0, fmt.Errorf("image async request body exceeds limit")
	}

	sum := sha256.Sum256(body)
	sha256Hex = hex.EncodeToString(sum[:])
	size = int64(len(body))

	dir := filepath.Join(GetImageAsyncStorageDir(), time.Now().Format("20060102"), taskID)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", "", 0, err
	}

	path = filepath.Join(dir, "request.body")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, body, 0600); err != nil {
		return "", "", 0, err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return "", "", 0, err
	}
	return path, sha256Hex, size, nil
}

func ReadImageAsyncRequestBody(path string, expectedSHA256 string, maxBytes int64) ([]byte, error) {
	clean, err := cleanImageAsyncBodyPath(path)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(clean)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("image async body exceeds limit")
	}
	sum := sha256.Sum256(data)
	if expectedSHA256 != "" && hex.EncodeToString(sum[:]) != expectedSHA256 {
		return nil, fmt.Errorf("image async body checksum mismatch")
	}
	return data, nil
}

func DeleteImageAsyncRequestBody(path string) error {
	clean, err := cleanImageAsyncBodyPath(path)
	if err != nil {
		return err
	}
	if err := os.Remove(clean); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func CleanupExpiredImageAsyncBodies(ttl time.Duration) error {
	root, err := filepath.Abs(GetImageAsyncStorageDir())
	if err != nil {
		return err
	}
	cutoff := time.Now().Add(-ttl)
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return nil
		}
		if d.IsDir() || filepath.Base(path) != "request.body" {
			return nil
		}
		info, statErr := d.Info()
		if statErr != nil {
			return nil
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(path)
		}
		return nil
	})
}

func cleanImageAsyncBodyPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty image async body path")
	}
	root, err := filepath.Abs(GetImageAsyncStorageDir())
	if err != nil {
		return "", err
	}
	clean, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, clean)
	if err != nil {
		return "", err
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("image async body path escapes storage dir")
	}
	return clean, nil
}
