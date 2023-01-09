package fs

import "os"

// ByName is a type that implements the
// sort.Interface interface for a slice
// of os.FileInfo values.
type ByName []os.FileInfo

// Len returns the length of the slice.
func (a ByName) Len() int {
	return len(a)
}

// Less compares the names of the file infos at
// the given indices and returns true if the
// name at index i is lexicographically less
// than the name at index j.
func (a ByName) Less(i, j int) bool {
	return a[i].Name() < a[j].Name()
}

// Swap swaps the elements at the given indices.
func (a ByName) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

// isCreate returns true if the O_CREATE flag
// is set in the given flag.
func isCreate(flag int) bool {
	return flag&os.O_CREATE != 0
}

// isExclusive returns true if the O_EXCL flag
// is set in the given flag.
func isExclusive(flag int) bool {
	return flag&os.O_EXCL != 0
}

// isAppend returns true if the O_APPEND flag
// is set in the given flag.
func isAppend(flag int) bool {
	return flag&os.O_APPEND != 0
}

// isTruncate returns true if the O_TRUNC flag
// is set in the given flag.
func isTruncate(flag int) bool {
	return flag&os.O_TRUNC != 0
}

// isReadAndWrite returns true if the O_RDWR flag
// is set in the given flag.
func isReadAndWrite(flag int) bool {
	return flag&os.O_RDWR != 0
}

// isReadOnly checks if the given flag is set to read-only mode.
// Read-only mode is specified with the os.O_RDONLY flag.
func isReadOnly(flag int) bool {
	return flag == os.O_RDONLY
}

// isWriteOnly checks if the given flag is set to write-only mode.
// Write-only mode is specified with the os.O_WRONLY flag.s
func isWriteOnly(flag int) bool {
	return flag&os.O_WRONLY != 0
}

// isSymlink checks if the given file mode indicates that
// the file is a symbolic link.
// A symbolic link is indicated by the os.ModeSymlink flag.
func isSymlink(m os.FileMode) bool {
	return m&os.ModeSymlink != 0
}
