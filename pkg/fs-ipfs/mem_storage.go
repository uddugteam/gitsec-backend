package fs

import (
	"fmt"
	"os"
	"path/filepath"
)

type memStorage struct {
	files    map[string]*IPFSFile
	children map[string]map[string]*IPFSFile
}

func newMemStorage() *memStorage {
	return &memStorage{
		files:    make(map[string]*IPFSFile, 0),
		children: make(map[string]map[string]*IPFSFile, 0),
	}
}

func (s *memStorage) Has(path string) bool {
	path = clean(path)

	_, ok := s.files[path]
	return ok
}

func (s *memStorage) New(path string, mode os.FileMode, flag int) (*IPFSFile, error) {
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
	}

	s.files[path] = f
	s.createParent(path, mode, f)
	return f, nil
}

func (s *memStorage) createParent(path string, mode os.FileMode, f *IPFSFile) error {
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

func (s *memStorage) Children(path string) []*IPFSFile {
	path = clean(path)

	l := make([]*IPFSFile, 0)
	for _, f := range s.children[path] {
		l = append(l, f)
	}

	return l
}

func (s *memStorage) MustGet(path string) *IPFSFile {
	f, ok := s.Get(path)
	if !ok {
		panic(fmt.Errorf("couldn't find %q", path))
	}

	return f
}

func (s *memStorage) Get(path string) (*IPFSFile, bool) {
	path = clean(path)
	if !s.Has(path) {
		return nil, false
	}

	file, ok := s.files[path]
	return file, ok
}

func (s *memStorage) Rename(from, to string) error {
	from = clean(from)
	to = clean(to)

	if !s.Has(from) {
		return os.ErrNotExist
	}

	move := [][2]string{{from, to}}

	for pathFrom := range s.files {
		if pathFrom == from || !filepath.HasPrefix(pathFrom, from) {
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

func (s *memStorage) move(from, to string) error {
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

func (s *memStorage) Remove(path string) error {
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
