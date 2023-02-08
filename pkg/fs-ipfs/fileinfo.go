package fs

import (
	"os"
	"time"
)

// fileInfo is an implementation of the os.FileInfo
// interface for IPFS Files.
type fileInfo struct {
	// name is the name of the file.
	name string
	// size is the size of the file in bytes.
	size int
	// mode is the file mode of the file.
	mode os.FileMode
}

// Name returns the name of the file.
func (fi *fileInfo) Name() string {
	return fi.name
}

// Size returns the size of the file in bytes.
func (fi *fileInfo) Size() int64 {
	return int64(fi.size)
}

// Mode returns the file mode of the file.
func (fi *fileInfo) Mode() os.FileMode {
	return fi.mode
}

// ModTime returns the modification time of the file.
// In this implementation, the modification time is
// always the current time.
func (*fileInfo) ModTime() time.Time {
	return time.Now()
}

// IsDir returns whether the file is a directory.
func (fi *fileInfo) IsDir() bool {
	return fi.mode.IsDir()
}

// Sys returns metadata associated with the file.
// In this implementation, there is no metadata
// associated with the file, so this method returns nil.
func (*fileInfo) Sys() interface{} {
	return nil
}
