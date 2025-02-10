//go:build linux
// +build linux

package docxgen

import (
	"os"
	"syscall"
)

func isMemoryBasedFS(path string) bool {
	if stat, err := os.Stat(path); err == nil {
		if sys := stat.Sys(); sys != nil {
			if statT, ok := sys.(*syscall.Statfs_t); ok {
				return statT.Type == 0x01021994 // TMPFS_MAGIC
			}
		}
	}
	return false
}
