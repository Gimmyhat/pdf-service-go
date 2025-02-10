//go:build windows
// +build windows

package docxgen

import (
	"os"
	"path/filepath"
	"strings"
)

// isMemoryBasedFS проверяет, является ли путь memory-based хранилищем.
// В Windows нет прямого аналога tmpfs, но мы можем проверить, находится ли путь
// в RAM-диске или специальном временном каталоге.
func isMemoryBasedFS(path string) bool {
	// Проверяем, является ли путь абсолютным
	if !filepath.IsAbs(path) {
		var err error
		path, err = filepath.Abs(path)
		if err != nil {
			return false
		}
	}

	// Нормализуем путь
	path = strings.ToUpper(path)

	// Проверяем известные пути RAM-дисков
	knownRAMDrives := []string{
		`R:`,
		`M:`,
		`RAMDISK:`,
	}

	for _, drive := range knownRAMDrives {
		if strings.HasPrefix(path, drive) {
			return true
		}
	}

	// Проверяем, находится ли путь во временной директории
	tmpDir := os.TempDir()
	if tmpDir != "" {
		tmpDir = strings.ToUpper(tmpDir)
		return strings.HasPrefix(path, tmpDir)
	}

	return false
}
