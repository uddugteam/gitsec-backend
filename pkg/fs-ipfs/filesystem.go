package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/helper/chroot"
	"github.com/go-git/go-billy/v5/util"
	ipfs "github.com/ipfs/go-ipfs-api"
)

// IPFSFilesystem is a filesystem implementation
// using IPFS as the storage backend.
type IPFSFilesystem struct {
	// s is the storage for the filesystem.
	s *storage
	// stop is a channel used to signal the
	// filesystem to stop and save the storage.
	stop chan struct{}
}

// NewIPFSFilesystem creates a new IPFSFilesystem instance.
func NewIPFSFilesystem(clientAddr string, stop chan struct{}) (billy.Filesystem, error) {
	fs := &IPFSFilesystem{
		s:    newStorage(ipfs.NewShell(clientAddr)),
		stop: stop,
	}

	/*if err := fs.s.LoadStorage(); err != nil {
		return nil, fmt.Errorf("failed to load storage: %w", err)
	}*/

	go fs.waitStop()

	//return fs, nil
	return chroot.New(fs, "./"), nil
}

// Create creates a new file with the specified filename.
// If successful, methods on the returned File can be used for I/O;
// the associated file descriptor has mode O_RDWR.
// If there is an error, it should be of type *PathError.
func (fs *IPFSFilesystem) Create(filename string) (billy.File, error) {
	// open the file with the specified flag and permission
	// the O_RDWR flag allows the file to be opened for reading and writing
	// the O_CREATE flag creates the file if it does not already exist
	// the O_TRUNC flag truncates the file if it already exists
	return fs.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
}

// Open opens the named file for reading. If successful, methods
// on the returned file can be used for reading; the associated
// file descriptor has mode O_RDONLY.
// If there is an error, it should be of type *PathError.
func (fs *IPFSFilesystem) Open(filename string) (billy.File, error) {
	return fs.OpenFile(filename, os.O_RDONLY, 0)
}

// OpenFile is the generalized open call; most users will use Open or Create instead.
// It opens the named file with specified flag (O_RDONLY etc.) and perm, (0666 etc.) if applicable.
// If successful, methods on the returned File can be used for I/O.
// If there is an error, it should be of type *PathError.
func (fs *IPFSFilesystem) OpenFile(filename string, flag int, perm os.FileMode) (billy.File, error) {
	// retrieve the file from the storage
	f, has := fs.s.Get(filename)
	if !has {
		if !isCreate(flag) {
			return nil, os.ErrNotExist
		}

		if flag&os.O_WRONLY != 0 || flag&os.O_RDWR != 0 {
			// handle cases where the file should be created and opened
			// for writing or reading and writing
			var err error
			f, err = fs.s.New(filename, perm, flag)
			if err != nil {
				return nil, fmt.Errorf("failed to create new file in filesystem: %w", err)
			}
		} else {
			// handle case where the file should be created but not opened for writing
			return nil, fmt.Errorf("creating file without write permissions is not allowed")
		}
	} else {
		if isExclusive(flag) {
			return nil, os.ErrExist
		}

		if target, isLink := fs.resolveLink(filename, f); isLink {
			return fs.OpenFile(target, flag, perm)
		}
	}

	if f.Mode.IsDir() {
		return nil, fmt.Errorf("cannot open directory: %s", filename)
	}

	// return a duplicate of the file with the specified filename, permission, and flag
	return f.Duplicate(filename, perm, flag), nil
}

// Stat returns the FileInfo structure describing file.
// If there is an error, it should return a non-nil error.
func (fs *IPFSFilesystem) Stat(filename string) (os.FileInfo, error) {
	// retrieve the file from the storage
	f, has := fs.s.Get(filename)
	if !has {
		return nil, os.ErrNotExist
	}

	// retrieve the file's metadata from the storage
	fi, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve file metadata: %w", err)
	}

	if target, isLink := fs.resolveLink(filename, f); isLink {
		// file is a link, follow the link and retrieve the
		// metadata of the target file
		fi, err = fs.Stat(target)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve file metadata: %w", err)
		}
	}

	// the name of the file should always the name of the stated file, so we
	// overwrite the Stat returned from the storage with it, since the
	// filename may belong to a link.
	fi.(*fileInfo).name = filepath.Base(filename)
	return fi, nil
}

// Rename renames (moves) oldpath to newpath. If newpath already exists and
// is not a directory, Rename replaces it. OS-specific restrictions may apply
// when oldpath and newpath are in different directories.
// If there is an error, it should be of type *LinkError.
func (fs *IPFSFilesystem) Rename(oldpath, newpath string) error {
	return fs.s.Rename(oldpath, newpath)
}

// Remove removes the named file or directory.
func (fs *IPFSFilesystem) Remove(filename string) error {
	// retrieve the file from the storage
	f, has := fs.s.Get(filename)
	if !has {
		// file does not exist in the storage, return an error
		return fmt.Errorf("failed to remove file: %w", os.ErrNotExist)
	}

	if f.Mode.IsDir() {
		// file is a directory, return an error
		return fmt.Errorf("cannot remove directory: %s", filename)
	}

	if target, isLink := fs.resolveLink(filename, f); isLink {
		// file is a link, follow the link and remove the target file
		return fs.Remove(target)
	}

	// remove the file from the storage
	if err := fs.s.Remove(filename); err != nil {
		// there was an error removing the file, return the error
		return fmt.Errorf("failed to remove file: %w", err)
	}
	return nil
}

// Join joins any number of path elements into a single path, adding a
// Separator if necessary. Join calls Clean on the result; in particular,
// all empty strings are ignored.
// On Windows, the result is a UNC path if and only if the first path
// element is a UNC path.
func (fs *IPFSFilesystem) Join(elem ...string) string {
	return filepath.Join(elem...)
}

// TempFile creates a new temporary file in the directory dir with a name
// beginning with prefix, opens the file for reading and writing, and
// returns the resulting *os.File. If dir is the empty string, TempFile
// uses the default directory for temporary Files (see os.TempDir).
// Multiple programs calling TempFile simultaneously will not choose the same file.
// The caller can use f.Name() to find the pathname of the file. It is the caller's
// responsibility to remove the file when no longer needed.
func (fs *IPFSFilesystem) TempFile(dir, prefix string) (billy.File, error) {
	return util.TempFile(fs, dir, prefix)
}

// ReadDir reads the directory named by dirname and returns a list of directory
// entries sorted by filename. If dirname is a symlink, ReadDir follows the symlink
// and returns the list of entries for the directory the symlink points to.
// If dirname is "", ReadDir returns entries for the current working directory.
// If the directory cannot be opened or read, ReadDir returns an empty list
// and an error of type *PathError.
func (fs *IPFSFilesystem) ReadDir(path string) ([]os.FileInfo, error) {
	// retrieve the directory from the storage
	if f, has := fs.s.Get(path); has {
		if target, isLink := fs.resolveLink(path, f); isLink {
			return fs.ReadDir(target)
		}
	}

	var entries []os.FileInfo
	for _, f := range fs.s.Childrens(path) {
		fi, _ := f.Stat()
		entries = append(entries, fi)
	}

	sort.Sort(ByName(entries))

	return entries, nil
}

// MkdirAll creates a directory named path, along with any necessary parents,
// and returns nil, or else returns an error. The permission bits perm are used
// for all directories that MkdirAll creates. If path is already a directory,
// MkdirAll does nothing and returns nil.
func (fs *IPFSFilesystem) MkdirAll(filename string, perm os.FileMode) error {
	// create a new directory in the storage
	_, err := fs.s.New(filename, perm|os.ModeDir, 0)
	return err
}

// Lstat returns a FileInfo describing the named file. If the file is a symbolic link,
// the returned FileInfo describes the symbolic link. Lstat makes no attempt to
// follow the link.
// If there is an error, it should be of type *PathError.
func (fs *IPFSFilesystem) Lstat(filename string) (os.FileInfo, error) {
	// retrieve the file from the storage
	f, has := fs.s.Get(filename)
	if !has {
		return nil, os.ErrNotExist
	}

	return f.Stat()
}

// Symlink creates newname as a symbolic link to oldname.
// If there is an error, it should be of type *LinkError.
func (fs *IPFSFilesystem) Symlink(target, link string) error {
	_, err := fs.Stat(link)
	if err == nil {
		return os.ErrExist
	}

	if !os.IsNotExist(err) {
		return os.ErrNotExist
	}

	return util.WriteFile(fs, link, []byte(target), 0777|os.ModeSymlink)
}

// Readlink returns the destination of the named symbolic link.
// If there is an error, it should be of type *PathError.
func (fs *IPFSFilesystem) Readlink(link string) (string, error) {
	// retrieve the file from the storage
	f, has := fs.s.Get(link)
	if !has {
		return "", os.ErrNotExist
	}

	if !isSymlink(f.Mode) {
		return "", &os.PathError{
			Op:   "readlink",
			Path: link,
			Err:  fmt.Errorf("not a symlink"),
		}
	}

	return string(f.content.bytes), nil
}

// Capabilities implements the Capable interface.
func (fs *IPFSFilesystem) Capabilities() billy.Capability {
	return billy.DefaultCapabilities
}

// resolveLink resolves the target of the given symbolic link
// and returns the target and a boolean indicating whether
// the file is a symbolic link.
func (fs *IPFSFilesystem) resolveLink(fullpath string, f *IPFSFile) (target string, isLink bool) {
	if !isSymlink(f.Mode) {
		return fullpath, false
	}

	target = string(f.content.bytes)
	if !isAbs(target) {
		target = fs.Join(filepath.Dir(fullpath), target)
	}

	return target, true
}

// waitStop waits for the stop channel to be closed
// and saves the storage to the filesystem.
func (fs *IPFSFilesystem) waitStop() {
	<-fs.stop
	/*if err := fs.s.SaveStorage(); err != nil {
		logger.Log().Errorf("failed to save storage: %v", err)
	}*/
}

// On Windows OS, IsAbs validates if a path is valid based on if stars with a
// unit (eg.: `C:\`)  to assert that is absolute, but in this mem implementation
// any path starting by `separator` is also considered absolute.
func isAbs(path string) bool {
	return filepath.IsAbs(path) || strings.HasPrefix(path, string(separator))
}
