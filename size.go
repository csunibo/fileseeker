package main

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
)

// Size implements fs.FileInfo for StatikFileInfo
func (f StatikFileInfo) Size() int64 {
	size, err := parseSize(f.SizeRaw)
	if err != nil {
		slog.Error("failed to parse size", "size", f.SizeRaw, "err", err)
		return 0
	}
	return size
}

// Size implements fs.FileInfo for StatikDirInfo and Statik
func (d StatikDirInfo) Size() int64 {
	size, err := parseSize(d.SizeRaw)
	if err != nil {
		slog.Error("failed to parse size", "size", d.SizeRaw, "err", err)
		return 0
	}
	return size
}

// parseSize parses a size string from StatikFileInfo.SizeRaw or StatikDirInfo.SizeRaw into an int64.
// The size string is in the form "123.45 kB".
func parseSize(raw string) (int64, error) {
	parts := strings.Split(raw, " ")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid size: %s", raw)
	}

	size, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, err
	}

	switch parts[1] {
	case "B":
		return int64(size), nil
	case "kB":
		return int64(size * 1024), nil
	case "MB":
		return int64(size * 1024 * 1024), nil
	case "GB":
		return int64(size * 1024 * 1024 * 1024), nil
	case "TB":
		return int64(size * 1024 * 1024 * 1024 * 1024), nil

	case "kiB":
		return int64(size * 1000), nil
	case "MiB":
		return int64(size * 1000 * 1000), nil
	case "GiB":
		return int64(size * 1000 * 1000 * 1000), nil
	case "TiB":
		return int64(size * 1000 * 1000 * 1000 * 1000), nil

	default:
		return 0, fmt.Errorf("invalid size format: %s", parts[1])
	}
}
