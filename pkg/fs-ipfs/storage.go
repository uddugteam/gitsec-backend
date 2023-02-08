package fs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	ipfs "github.com/ipfs/go-ipfs-api"
)

const separator = filepath.Separator

// storage is a type that represents a file storage.
type storage struct {
	// Files is a map that stores the Files in the
	// storage by their path.
	Files map[string]*IPFSFile

	// Children is a map that stores the Children
	// of a directory by their parent directory path.
	Children map[string]map[string]*IPFSFile

	// client is an instance of the IPFS client.
	client *ipfs.Shell
}

// newStorage creates a new storage instance.
func newStorage(client *ipfs.Shell) *storage {
	return &storage{
		Files:    make(map[string]*IPFSFile),
		Children: make(map[string]map[string]*IPFSFile),
		client:   client,
	}
}

// MarshalJSON marshals the storage instance
// into a JSON representation.
func (s *storage) MarshalJSON() ([]byte, error) {
	storageJson := struct {
		Files    map[string]*IPFSFile
		Children map[string]map[string]*IPFSFile
	}{
		Files:    s.Files,
		Children: s.Children,
	}

	return json.Marshal(storageJson)
}

// UnmarshalJSON unmarshals a JSON representation
// into a storage instance.
func (s *storage) UnmarshalJSON(bytes []byte) error {
	var objmap map[string]*json.RawMessage

	if err := json.Unmarshal(bytes, &objmap); err != nil {
		return fmt.Errorf("failed to unmarshal given data to temp map: %w", err)
	}

	if err := json.Unmarshal(*objmap["Files"], &s.Files); err != nil {
		return fmt.Errorf("failed to unmarshal Files data to storage: %w", err)
	}

	if err := json.Unmarshal(*objmap["Children"], &s.Children); err != nil {
		return fmt.Errorf("failed to unmarshal Children data to storage: %w", err)
	}

	return nil
}

// Has checks if a file or directory with the given path exists in the storage.
// It returns a boolean value indicating whether the file or directory exists.
func (s *storage) Has(path string) bool {
	path = clean(path)

	_, ok := s.Files[path]
	return ok
}

// New creates a new file at the specified path. If the file already exists, it
// returns an error. If the file is a directory, it returns nil.
func (s *storage) New(path string, mode os.FileMode, flag int) (*IPFSFile, error) {
	// Clean the path by removing any leading or trailing whitespace
	// and resolving any relative path elements.
	path = clean(path)

	// Check if the file already exists in the storage.
	if s.Has(path) {
		// If the file is not a directory, return an error.
		if !s.MustGet(path).Mode.IsDir() {
			return nil, fmt.Errorf("file already exists %q", path)
		}

		// Otherwise, return nil as the file is a directory.
		return nil, nil
	}

	// Extract the base name of the file from the path.
	name := filepath.Base(path)

	// Create a new IPFSFile with the specified name, content, mode, flag, and client.
	f := &IPFSFile{
		FileName: name,
		content:  &content{name: name},
		Mode:     mode,
		Flag:     flag,
		client:   s.client,
	}

	// Add the new file to the storage.
	s.Files[path] = f

	// Create the parent directory for the file if it doesn't already exist.
	if err := s.createParent(path, mode, f); err != nil {
		return nil, fmt.Errorf("failed to create parent directory for file %q: %w", path, err)
	}

	f.fillContent()
	return f, nil
}

// createParent creates the parent directory for the given path if it does not already exist.
// It also adds the file to the Children map for the parent directory.
func (s *storage) createParent(path string, mode os.FileMode, f *IPFSFile) error {
	base := filepath.Dir(path)
	base = clean(base)
	if f.Name() == string(separator) {
		return nil
	}

	if _, err := s.New(base, mode.Perm()|os.ModeDir, 0); err != nil {
		return fmt.Errorf("failed to create parent directory %q: %w", base, err)
	}

	if _, ok := s.Children[base]; !ok {
		s.Children[base] = make(map[string]*IPFSFile, 0)
	}

	s.Children[base][f.Name()] = f
	return nil
}

// Children returns a slice of IPFSFiles that are Children of the specified directory path.
// If the path does not exist, or is not a directory, an empty slice is returned.
func (s *storage) Childrens(path string) []*IPFSFile {
	// Clean the path to remove any leading or trailing whitespace, and ensure that
	// it uses the correct separator for the current operating system.
	path = clean(path)

	// Check if the specified path exists in the Children map. If it does not, return
	// an empty slice.
	children, ok := s.Children[path]
	if !ok {
		return []*IPFSFile{}
	}

	// Initialize an empty slice of IPFSFiles.
	l := make([]*IPFSFile, 0)
	// Iterate over the Children map and append each child IPFSFile to the slice.
	for _, f := range children {
		l = append(l, f)
	}

	// Return the slice of IPFSFiles.
	return l
}

// MustGet is similar to Get, but panics if the file does not exist. This is
// useful for cases where a file is expected to exist and the caller does not
// want to explicitly check for its existence.
func (s *storage) MustGet(path string) *IPFSFile {
	f, ok := s.Get(path)
	if !ok {
		panic(fmt.Errorf("couldn't find %q", path))
	}

	return f
}

// Get retrieves a file from the storage by its path.
// If the file doesn't exist, it returns a nil value and a false boolean value.
func (s *storage) Get(path string) (*IPFSFile, bool) {
	path = clean(path)

	// Check if the file exists in the storage.
	if !s.Has(path) {
		return nil, false
	}

	// Retrieve the file from the storage.
	file, ok := s.Files[path]
	if !ok {
		return nil, false
	}

	// Reset the file's state to open and assign the client to it.
	file.client = s.client
	file.isClosed = false

	return file, ok
}

// Rename renames a file or directory located at `from` path to `to` path. If `from` path refers to a
// directory, all its Children will be also renamed to keep the directory tree structure.
// If a file or directory with the same name as `to` path already exists, the function will return an error.
func (s *storage) Rename(from, to string) error {
	from = clean(from)
	to = clean(to)

	if !s.Has(from) {
		return os.ErrNotExist
	}

	move := [][2]string{{from, to}}

	for pathFrom := range s.Files {
		if pathFrom == from || !strings.HasPrefix(pathFrom, from) {
			continue
		}

		rel, _ := filepath.Rel(from, pathFrom)
		pathTo := filepath.Join(to, rel)

		move = append(move, [2]string{pathFrom, pathTo})
	}

	for _, ops := range move {
		from := ops[0]
		to := ops[1]

		if err := s.move(from, to); err != nil {
			return fmt.Errorf("failed to move file from %q to %q: %w", from, to, err)
		}
	}

	return nil
}

// move renames a file or directory from `from` to `to`.
// If `to` exists and is not a directory, it is replaced.
func (s *storage) move(from, to string) error {
	// move the file from `from` to `to` in the `Files` map
	s.Files[to] = s.Files[from]
	// update the file's name to the new name
	s.Files[to].FileName = filepath.Base(to)
	// move the file's Children from `from` to `to` in the `Children` map
	s.Children[to] = s.Children[from]

	// delete the file's old location in the `Children` map
	defer func() {
		delete(s.Children, from)
		// delete the file from the `Files` map
		delete(s.Files, from)
		// delete the file from the `Children` map of its parent directory
		delete(s.Children[filepath.Dir(from)], filepath.Base(from))
	}()

	// create the parent directories for the new location of the file
	return s.createParent(to, 0644, s.Files[to])
}

// Remove removes a file or directory from the storage.
// If the path does not exist, os.ErrNotExist is returned.
// If the path refers to a non-empty directory, an error is returned.
func (s *storage) Remove(path string) error {
	path = clean(path)

	f, has := s.Get(path)
	if !has {
		return os.ErrNotExist
	}

	if f.Mode.IsDir() && len(s.Children[path]) != 0 {
		return fmt.Errorf("dir: %s contains Files", path)
	}

	base, file := filepath.Split(path)
	base = filepath.Clean(base)

	delete(s.Children[base], file)
	delete(s.Files, path)
	return nil
}

const fileSystemBackupPath = ".fs.json"

/*
func (s *storage) LoadStorage() error {
	// Open a RO file
	decodeFile, err := os.Open("map.gob")
	if err != nil {
		fmt.Println(err)
		fmt.Printf("%T\n", err)
		return nil
	}

	if decodeFile != nil {
		defer decodeFile.Close()
	}

	// Create a decoder
	decoder := gob.NewDecoder(decodeFile)

	// Decode -- We need to pass a pointer otherwise accounts2 isn't modified
	if err := decoder.Decode(&s); err != nil {
		panic(err)
	}

	for _, f := range s.Files {
		if f == nil {
			continue
		}

		f.content = &content{name: f.Name()}

		if f.IpfsPath == "" {
			continue
		}

		f.client = s.client

		if err := f.fillContent(); err != nil {
			return fmt.Errorf("failed to fill content for file: %w", err)
		}

	}

	fmt.Println(s)

	logger.Log().Info("opening fs storage...")
	return nil

	fsFile, err := os.Open(fileSystemBackupPath)
	if err != nil {

		if os.IsNotExist(err) {
			//return fmt.Errorf("failed to open fs file: %w", err)
			return nil
		}

		fsFile, err = os.Create(fileSystemBackupPath)
		if err != nil {
			return fmt.Errorf("failed to create fs file: %w", err)
		}

	}

	defer fsFile.Close()

	readedFs, err := io.ReadAll(fsFile)
	if err != nil {
		return fmt.Errorf("failed to read fs file: %w", err)
	}

	if err := json.Unmarshal(readedFs, &s); err != nil {
		return fmt.Errorf("failed to unmarshal fs file in storage: %w", err)
	}

	for _, f := range s.Files {
		if f == nil {
			continue
		}

		f.content = &content{name: f.Name()}

		if f.IpfsPath == "" {
			continue
		}

		f.client = s.client

		if err := f.fillContent(); err != nil {
			return fmt.Errorf("failed to fill content for file: %w", err)
		}
	}

	for k, c := range s.Children {
		fmt.Println(k)

		for k2, d := range c {
			fmt.Println(k2)

			fmt.Println(d.IpfsPath)
		}


	logger.Log().Info("storage opened")

	fmt.Println(s)

	for _, f := range s.Files {
		if f == nil {
			continue
		}
		logger.Log().Infof("Name: %s, IPFS: %s, Position: %d, Flag: %d, Mode: %s", f.Name(), f.IpfsPath, f.Position, f.Flag, f.Mode)
	}
	return nil
}*/

/*func (s *storage) SaveStorage() error {
	encodeFile, err := os.Create("map.gob")
	if err != nil {
		panic(err)
	}

	encoder := gob.NewEncoder(encodeFile)

	// Write to the file
	if err := encoder.Encode(s); err != nil {

		fmt.Println(err)
		panic(err)
	}

	if err := encodeFile.Close(); err != nil {
		panic(err)
	}

	logger.Log().Info("saving storage...")
	fmt.Println(s)
	return nil

	storageJson, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("error marshaling: %w", err)
	}

	fsFile, err := os.Create(fileSystemBackupPath)
	if err != nil {
		return fmt.Errorf("failed to create fs file: %w", err)
	}
	defer fsFile.Close()

	if _, err := fsFile.Write(storageJson); err != nil {
		return fmt.Errorf("failed to write fs file: %w", err)
	}

	logger.Log().Info("storage saved")

	fmt.Println(s)

	for _, f := range s.Files {
		logger.Log().Infof("Name: %s, IPFS: %s, Position: %d, Flag: %d, Mode: %s", f.Name(), f.IpfsPath, f.Position, f.Flag, f.Mode)
	}

	return nil
}*/

// clean is a helper function that converts the given path to a cleaned and consistent form.
// It converts all slashes to the local file system's separator and removes any trailing separator.
func clean(path string) string {
	return filepath.Clean(filepath.FromSlash(path))
}
