package fs

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/go-git/go-billy/v5"
	ipfs "github.com/ipfs/go-ipfs-api"
)

// IPFSFile is a go-billy file that stores data on IPFS.
type IPFSFile struct {
	// FileName is the name of the file.
	FileName string
	// IpfsPath is the IPFS path of the file.
	IpfsPath string
	// Position is the current position in the file.
	Position int64
	// Flag is the file mode flag, such as os.O_RDONLY, os.O_WRONLY, os.O_RDWR, etc.
	Flag int
	// Mode is the file mode, such as os.FileMode.
	Mode os.FileMode

	// isClosed indicates whether the file has been closed.
	isClosed bool

	// content is the file content.
	content *content

	mu sync.Mutex

	// client is the IPFS client used to store and retrieve file data.
	client *ipfs.Shell

	// isDuplicate indicates whether this file is a duplicate of another file.
	isDuplicate bool
	// updateOriginal is a function that updates the original file when this file is a duplicate.
	updateOriginal func(ipfsPath string)
}

// Name returns the name of the file.
func (f *IPFSFile) Name() string {
	return f.FileName
}

// Read reads data from the file. If the file has not yet been loaded from IPFS, it will be
// fetched and stored in memory. The file's position is then advanced by the number of bytes
// read. If the end of the file is reached and some data was still read, io.EOF is returned.
func (f *IPFSFile) Read(b []byte) (int, error) {
	defer f.mu.Unlock()
	f.mu.Lock()

	if f.content.bytes == nil && f.IpfsPath != "" {
		// Fetch file from IPFS and store in memory.
		ipfsFileReader, err := f.client.Cat(f.IpfsPath)
		if err != nil {
			return 0, fmt.Errorf("can't find file content on %s path", f.IpfsPath)
		}
		defer ipfsFileReader.Close()

		var buffer bytes.Buffer
		if _, err := buffer.ReadFrom(ipfsFileReader); err != nil {
			return 0, fmt.Errorf("error to read file: %w", err)
		}

		f.content.bytes = buffer.Bytes()
	}

	n, err := f.ReadAt(b, f.Position)
	f.Position += int64(n)

	if err == io.EOF && n != 0 {
		err = nil
	}

	return n, err
}

// Write writes data to the file.
// It stores the data in a bytes.Buffer and stores the data on
// IPFS using the ipfs.Add function.
// If the file was opened for writing or appending and the Flag value
// is os.O_TRUNC, the file is truncated before writing.
// If the file is a duplicate, the original file is updated with the
// new IPFS hash using the updateOriginal function.
func (f *IPFSFile) Write(b []byte) (int, error) {
	defer f.mu.Unlock()
	f.mu.Lock()

	if f.isClosed {
		return 0, os.ErrClosed
	}

	if !isReadAndWrite(f.Flag) && !isWriteOnly(f.Flag) {
		return 0, errors.New("write not supported")
	}

	n, err := f.content.WriteAt(b, f.Position)
	f.Position += int64(n)

	hash, err := f.client.Add(bytes.NewReader(b))
	if err != nil {
		return 0, fmt.Errorf("failed to add new file to ipfs: %w", err)
	}

	f.IpfsPath = hash

	if f.isDuplicate {
		f.updateOriginal(hash)
	}

	return n, err
}

// ReadAt reads data from the file at a specific offset.
// It returns an error if the file is closed or if the file was not opened for reading or reading and writing.
// It also returns an error if the requested offset is out of bounds.
func (f *IPFSFile) ReadAt(b []byte, offset int64) (int, error) {
	if f.isClosed {
		return 0, os.ErrClosed
	}

	if !isReadAndWrite(f.Flag) && !isReadOnly(f.Flag) {
		return 0, errors.New("read not supported")
	}

	if offset < 0 || offset > int64(f.content.Len()) {
		return 0, errors.New("offset out of bounds")
	}

	return f.content.ReadAt(b, offset)
}

// Seek sets the offset for the next read or write to the file.
// If the file has already been closed, it returns an error.
// It uses the whence value to determine the new position:
//
//	io.SeekStart: the new position is offset bytes from the start of the file
//	io.SeekCurrent: the new position is the current position plus the offset
//	io.SeekEnd: the new position is the end of the file plus the offset
//
// It returns the new position.
func (f *IPFSFile) Seek(offset int64, whence int) (int64, error) {
	if f.isClosed {
		return 0, os.ErrClosed
	}

	switch whence {
	case io.SeekCurrent:
		f.Position += offset
	case io.SeekStart:
		f.Position = offset
	case io.SeekEnd:
		f.Position = int64(f.content.Len()) + offset
	default:
		return 0, errors.New("invalid whence value")
	}

	return f.Position, nil
}

// Lock locks the file for writing.
// This method is a no-op and always returns nil.
func (f *IPFSFile) Lock() error {
	return nil
}

// Unlock unlocks the file for writing.
// This method is a no-op and always returns nil.
func (f *IPFSFile) Unlock() error {
	return nil
}

// Truncate changes the size of the file.
func (f *IPFSFile) Truncate(size int64) error {
	if f.isClosed {
		return os.ErrClosed
	}

	if size < int64(len(f.content.bytes)) {
		f.content.bytes = f.content.bytes[:size]
	} else if more := int(size) - len(f.content.bytes); more > 0 {
		f.content.bytes = append(f.content.bytes, make([]byte, more)...)
	}

	return nil
}

// Stat returns the FileInfo structure describing file.
func (f *IPFSFile) Stat() (os.FileInfo, error) {
	if f.isClosed {
		return nil, os.ErrClosed
	}

	return &fileInfo{
		name: f.Name(),
		mode: f.Mode,
		size: f.content.Len(),
	}, nil
}

// Duplicate returns a new instance of the file with the same file name, mode, and flag.
// The new instance shares the same IPFS client and content as the original file.
func (f *IPFSFile) Duplicate(filename string, mode os.FileMode, flag int) billy.File {
	updateIpfsPath := func(ipfsPath string) {
		f.IpfsPath = ipfsPath
	}

	new := &IPFSFile{
		FileName:       filename,
		content:        f.content,
		Mode:           mode,
		Flag:           flag,
		client:         f.client,
		IpfsPath:       f.IpfsPath,
		isDuplicate:    true,
		updateOriginal: updateIpfsPath,
	}

	if isAppend(flag) {
		new.Position = int64(new.content.Len())
	}

	if isTruncate(flag) {
		new.content.Truncate()
	}

	return new
}

// Close closes the file.
func (f *IPFSFile) Close() error {
	if f.isClosed {
		return os.ErrClosed
	}

	f.isClosed = true
	return nil
}

// content represents the contents of
// a file stored on IPFS.
type content struct {
	name  string
	bytes []byte
}

// WriteAt writes data to the content at a specific offset.
// If the offset is negative, it returns a PathError with
// a negative offset error.
// If the offset is greater than the length of the content,
// it appends zeros to the content to make it long enough to
// accommodate the write.
// If the write extends beyond the current length of the content,
// it appends the data to the content.
// If the write is shorter than the current length of the content,
// it truncates the content to the length of the write.
// It returns the number of bytes written and any error that occurred.
func (c *content) WriteAt(b []byte, off int64) (int, error) {
	if off < 0 {
		return 0, &os.PathError{
			Op:   "writeat",
			Path: c.name,
			Err:  errors.New("negative offset"),
		}
	}

	prev := len(c.bytes)

	diff := int(off) - prev
	if diff > 0 {
		c.bytes = append(c.bytes, make([]byte, diff)...)
	}

	c.bytes = append(c.bytes[:off], b...)
	if len(c.bytes) < prev {
		c.bytes = c.bytes[:prev]
	}

	return len(b), nil
}

// ReadAt reads data from the content at a specific offset.
// It returns the number of bytes read and any error encountered.
// If the offset is negative, it returns an os.PathError with the
// "negative offset" message.
// If the offset is greater or equal to the size of the content,
// it returns io.EOF.
// Otherwise, it copies the data from the content to the provided
// buffer and returns the number of bytes copied and any error
// encountered. If the number of bytes copied is less than the
// length of the buffer, it returns io.EOF.
func (c *content) ReadAt(b []byte, off int64) (n int, err error) {
	if off < 0 {
		return 0, &os.PathError{
			Op:   "readat",
			Path: c.name,
			Err:  errors.New("negative offset"),
		}
	}

	size := int64(len(c.bytes))
	if off >= size {
		return 0, io.EOF
	}

	l := int64(len(b))
	if off+l > size {
		l = size - off
	}

	btr := c.bytes[off : off+l]
	if len(btr) < len(b) {
		err = io.EOF
	}
	n = copy(b, btr)

	return
}

// Truncate resets the content slice to an empty slice.
func (c *content) Truncate() {
	c.bytes = make([]byte, 0)
}

// Len returns the length of the content slice.
func (c *content) Len() int {
	return len(c.bytes)
}
