package fs

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	ipfs "github.com/ipfs/go-ipfs-api"
	"github.com/misnaged/annales/logger"
)

const separator = filepath.Separator

type storage struct {
	files    map[string]*IPFSFile
	children map[string]map[string]*IPFSFile

	client *ipfs.Shell
}

func newStorage(client *ipfs.Shell) *storage {
	return &storage{
		files:    make(map[string]*IPFSFile),
		children: make(map[string]map[string]*IPFSFile),
		client:   client,
	}
}

func (s *storage) MarshalJSON() ([]byte, error) {
	storageJson := struct {
		Files    map[string]*IPFSFile
		Children map[string]map[string]*IPFSFile
	}{
		Files:    s.files,
		Children: s.children,
	}

	return json.Marshal(storageJson)
}

func (s *storage) UnmarshalJSON(bytes []byte) error {
	var objmap map[string]*json.RawMessage

	if err := json.Unmarshal(bytes, &objmap); err != nil {
		return fmt.Errorf("failed to unmarshal given data to temp map: %w", err)
	}

	if err := json.Unmarshal(*objmap["Files"], &s.files); err != nil {
		return fmt.Errorf("failed to unmarshal files data to storage: %w", err)
	}

	if err := json.Unmarshal(*objmap["Children"], &s.files); err != nil {
		return fmt.Errorf("failed to unmarshal children data to storage: %w", err)
	}

	return nil
}

func (s *storage) Has(path string) bool {
	path = clean(path)

	_, ok := s.files[path]
	return ok
}

func (s *storage) New(path string, mode os.FileMode, flag int) (*IPFSFile, error) {
	path = clean(path)
	if s.Has(path) {
		if !s.MustGet(path).Mode.IsDir() {
			return nil, fmt.Errorf("file already exists %q", path)
		}

		return nil, nil
	}

	name := filepath.Base(path)

	f := &IPFSFile{
		FileName: name,
		content:  &content{name: name},
		Mode:     mode,
		Flag:     flag,
		client:   s.client,
	}

	s.files[path] = f
	s.createParent(path, mode, f)
	return f, nil
}

func (s *storage) createParent(path string, mode os.FileMode, f *IPFSFile) error {
	base := filepath.Dir(path)
	base = clean(base)
	if f.Name() == string(separator) {
		return nil
	}

	if _, err := s.New(base, mode.Perm()|os.ModeDir, 0); err != nil {
		return err
	}

	if _, ok := s.children[base]; !ok {
		s.children[base] = make(map[string]*IPFSFile, 0)
	}

	s.children[base][f.Name()] = f
	return nil
}

func (s *storage) Children(path string) []*IPFSFile {
	path = clean(path)

	l := make([]*IPFSFile, 0)
	for _, f := range s.children[path] {
		l = append(l, f)
	}

	return l
}

func (s *storage) MustGet(path string) *IPFSFile {
	f, ok := s.Get(path)
	if !ok {
		panic(fmt.Errorf("couldn't find %q", path))
	}

	return f
}

func (s *storage) Get(path string) (*IPFSFile, bool) {
	path = clean(path)
	if !s.Has(path) {
		return nil, false
	}

	file, ok := s.files[path]
	if !ok {
		return nil, false
	}

	//file.content = &content{
	//	name: file.Name(),
	//}
	file.client = s.client
	file.isClosed = false
	return file, ok
}

func (s *storage) Rename(from, to string) error {
	from = clean(from)
	to = clean(to)

	if !s.Has(from) {
		return os.ErrNotExist
	}

	move := [][2]string{{from, to}}

	for pathFrom := range s.files {
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
			return err
		}
	}

	return nil
}

func (s *storage) move(from, to string) error {
	s.files[to] = s.files[from]
	s.files[to].FileName = filepath.Base(to)
	s.children[to] = s.children[from]

	defer func() {
		delete(s.children, from)
		delete(s.files, from)
		delete(s.children[filepath.Dir(from)], filepath.Base(from))
	}()

	return s.createParent(to, 0644, s.files[to])
}

func (s *storage) Remove(path string) error {
	path = clean(path)

	f, has := s.Get(path)
	if !has {
		return os.ErrNotExist
	}

	if f.Mode.IsDir() && len(s.children[path]) != 0 {
		return fmt.Errorf("dir: %s contains files", path)
	}

	base, file := filepath.Split(path)
	base = filepath.Clean(base)

	delete(s.children[base], file)
	delete(s.files, path)
	return nil
}

const fileSystemBackupPath = ".fs.json"

func (s *storage) LoadStorage() error {
	logger.Log().Info("opening fs storage...")

	fsFile, err := os.Open(fileSystemBackupPath)
	if err != nil {

		fmt.Println(err)
		fmt.Printf("%T\n", err)

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

	logger.Log().Info("storage opened")
	return nil
}

func (s *storage) SaveStorage() error {
	logger.Log().Info("saving storage...")

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
	return nil
}

func clean(path string) string {
	return filepath.Clean(filepath.FromSlash(path))
}
