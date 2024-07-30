package utils

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
)

func ReadFile(path string, replacer ...map[string]string) []byte {
	path, err := filepath.Abs(path)
	if err != nil {
		slog.Error("filepath.Abs", "error", err)
		return []byte("404")
	}

	buf, err := os.ReadFile(path)
	if err != nil {
		slog.Error("os.ReadFile", "error", err)
		return []byte("404")
	}

	if len(replacer) == 0 {
		return buf
	}

	for key, value := range replacer[0] {
		buf = bytes.ReplaceAll(buf, []byte(key), []byte(value))
	}

	return buf
}
