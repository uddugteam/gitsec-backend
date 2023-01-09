package fs

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/go-git/go-billy/v5"
	ipfs "github.com/ipfs/go-ipfs-api"
)

// IPFSFile is a go-billy file that stores data on IPFS.
type IPFSFile struct {
	FileName string      `json:"fileName"`
	IpfsPath string      `json:"ipfsPath"`
	Position int64       `json:"position"`
	Flag     int         `json:"flag"`
	Mode     os.FileMode `json:"mode"`

	isClosed bool

	content *content

	client *ipfs.Shell

	isDuplicate    bool
	updateOriginal func(ipfsPath string)

	saveStorage func() error
}

func (f *IPFSFile) Name() string {
	return f.FileName
}

// Read reads data from the file.
func (f *IPFSFile) Read(b []byte) (int, error) {
	if f.IpfsPath == "" {
		//return 0, fmt.Errorf("nil content")
	}

	if f.content.bytes == nil && f.IpfsPath != "" {
		ipfsFileReader, err := f.client.Cat(f.IpfsPath)
		if err != nil {
			return 0, fmt.Errorf("can't find file content on %s path", f.IpfsPath)
		}

		contentFile, err := io.ReadAll(ipfsFileReader)
		if err != nil {
			return 0, fmt.Errorf("error to read file: %w", err)
		}

		f.content.bytes = contentFile
	}

	n, err := f.ReadAt(b, f.Position)
	f.Position += int64(n)

	if err == io.EOF && n != 0 {
		err = nil
	}

	return n, err
}

// Write writes data to the file.
func (f *IPFSFile) Write(b []byte) (int, error) {
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
		return 0, fmt.Errorf("add new file to ipfs: %w", err)
	}

	f.IpfsPath = hash

	if f.isDuplicate {
		f.updateOriginal(hash)
	}

	/*go func() {
		if err := f.saveStorage; err != nil {
			logger.Log().Error(fmt.Errorf("failed to update storage: %w", err()))
		}
	}()*/

	return n, err
}

// Close closes the file.
func (f *IPFSFile) Close() error {
	if f.isClosed {
		return os.ErrClosed
	}

	f.isClosed = true
	return nil
}

// ReadAt reads data from the file at a specific offset.
func (f *IPFSFile) ReadAt(b []byte, offset int64) (int, error) {
	if f.isClosed {
		return 0, os.ErrClosed
	}

	if !isReadAndWrite(f.Flag) && !isReadOnly(f.Flag) {
		return 0, errors.New("read not supported")
	}

	n, err := f.content.ReadAt(b, offset)

	return n, err
}

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
	}

	return f.Position, nil
}

func (f *IPFSFile) Lock() error {
	return nil
}

func (f *IPFSFile) Unlock() error {
	return nil
}

func (f *IPFSFile) Truncate(size int64) error {
	if size < int64(len(f.content.bytes)) {
		f.content.bytes = f.content.bytes[:size]
	} else if more := int(size) - len(f.content.bytes); more > 0 {
		f.content.bytes = append(f.content.bytes, make([]byte, more)...)
	}

	return nil
}

// Stat gets information about the file.
func (f *IPFSFile) Stat() (os.FileInfo, error) {
	return &fileInfo{
		name: f.Name(),
		mode: f.Mode,
		size: f.content.Len(),
	}, nil
}

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
		saveStorage:    f.saveStorage,
	}

	if isAppend(flag) {
		new.Position = int64(new.content.Len())
	}

	if isTruncate(flag) {
		new.content.Truncate()
	}

	return new
}

type content struct {
	name  string
	bytes []byte
}

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

func (c *content) Truncate() {
	c.bytes = make([]byte, 0)
}

func (c *content) Len() int {
	return len(c.bytes)
}
